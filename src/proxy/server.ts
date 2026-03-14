import * as http from "node:http";
import * as https from "node:https";
import { Router } from "../routing/router.js";
import { logger } from "../utils/logger.js";
import {
  register,
  requestCounter,
  requestDurationHistogram,
} from "../utils/metrics.js";

export class ProxyServer {
  private router: Router;
  private server: https.Server;

  constructor(router: Router, tlsOptions: https.ServerOptions) {
    this.router = router;
    this.server = https.createServer(tlsOptions, this.handleRequest.bind(this));
  }

  // Safely update the router for hot-reload
  public updateRouter(newRouter: Router) {
    this.router = newRouter;
    logger.info("ProxyServer internal router has been hot-swapped.");
  }

  public listen(port: number, callback?: () => void) {
    this.server.listen(port, callback);
  }

  private async handleRequest(
    clientReq: http.IncomingMessage,
    clientRes: http.ServerResponse,
  ) {
    // Get the destination url from client request
    const url = clientReq.url || "/";
    const method = clientReq.method || "GET";

    // --- METRICS INTERCEPTOR ---
    // If Prometheus is scraping, return the metrics directly. Do not route
    if (url === "/metrics" && method === "GET") {
      clientRes.setHeader("Content-Type", register.contentType);
      const metrics = await register.metrics();
      clientRes.end(metrics);
      return;
    }

    // Start the latency timer for normal traffic
    const endTimer = requestDurationHistogram.startTimer();

    // 1. Ask the router for a destination
    const targetServer = this.router.routeRequest(url);

    // 2. If no destination, return 502 and end connection
    if (!targetServer) {
      clientRes.writeHead(502, { "Content-Type": "text/plain" });
      clientRes.end("502 Bad Gateway: No available upstream servers.");
      return;
    }

    // 3. Construct the outbound request options
    const options: http.RequestOptions = {
      hostname: targetServer.host,
      port: targetServer.port,
      path: url,
      method: clientReq.method,
      headers: {
        ...clientReq.headers,
        "x-forwarded-for": clientReq.socket.remoteAddress || "", // This injects the original client IP
        "x-forwarded-proto": "https",
      },
    };

    // 4. Create a outbound request for the target server
    const proxyReq = http.request(options, (proxyRes: http.IncomingMessage) => {
      const status = proxyRes.statusCode || 200;

      // Forward the server's http status code and headers to client
      clientRes.writeHead(status, proxyRes.headers);

      // PIPE: Stream the server's response directly to the client
      proxyRes.pipe(clientRes);

      // Record the success status metric when the response finishes
      proxyRes.on("end", () => {
        requestCounter.inc({ method, status });
        endTimer({ method, status });
      });

      proxyRes.on("error", (err: Error) => {
        logger.error(
          {
            err: err,
          },
          "Proxy Response Error: Failed to connect to backend",
        );
        if (!clientRes.headersSent) {
          clientRes.writeHead(502);
        }
        clientRes.end();

        // Record the error status metric when the response finishes
        requestCounter.inc({ method, status: 502 });
        endTimer({ method, status: 502 });
      });
    });

    // 5. Handle errors on the outbound connection (e.g., if backend server collapses)
    proxyReq.on("error", (err: Error) => {
      logger.error(
        {
          host: targetServer.host,
          port: targetServer.port,
          err: err,
        },
        "Proxy Request Error: Failed to connect to backend",
      );
      // If headers are not already sent, send a 502
      if (!clientRes.headersSent) {
        clientRes.writeHead(502, { "Content-Type": "text/plain" });
      }
      // End the connection
      clientRes.end("502 Bad Gateway");

      // Record the error status metric when the response finishes
      requestCounter.inc({ method, status: 502 });
      endTimer({ method, status: 502 });
    });

    // 6. PIPE: Stream the incoming client payload directly to the server
    clientReq.pipe(proxyReq);

    // 7. Handle errors on the inbound connection (e.g., if client disconnects)
    clientReq.on("error", (err: Error) => {
      logger.error(
        {
          err: err,
        },
        "Client Request Error: Failed to connect to backend",
      );
      proxyReq.destroy();
    });
  }
}
