import * as http from "node:http";
import * as https from "node:https";
import net from "node:net";
import { Router } from "../routing/router.js";
import { logger } from "../utils/logger.js";
import {
  register,
  requestCounter,
  requestDurationHistogram,
  activeWebSockets,
  wsUpgradesTotal,
} from "../utils/metrics.js";
import { RedisRateLimiter } from "../security/rateLimiter.js";

export class ProxyServer {
  private router: Router;
  private server: https.Server;
  private rateLimiter: RedisRateLimiter;

  constructor(router: Router, tlsOptions: https.ServerOptions) {
    this.router = router;
    this.server = https.createServer(tlsOptions, this.handleRequest.bind(this));
    this.server.on("upgrade", this.handleUpgrade.bind(this));
    this.rateLimiter = new RedisRateLimiter();
  }

  // Safely update the router for hot-reload
  public updateRouter(newRouter: Router) {
    this.router = newRouter;
    logger.info("ProxyServer internal router has been hot-swapped.");
  }

  public listen(port: number, callback?: () => void) {
    this.server.listen(port, callback);
  }

  /* --------------------
     HTTP REQUEST HANDLER
   -------------------- */
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

    // --- RATE LIMITING ---
    const clientIp = clientReq.socket.remoteAddress || "unknown";

    // Capacity: 100 requests
    // Refill Rate: 10 requests per second
    /* TODO: Enterprise API Gateways doesnt hardcode rate limiting. It defines them per-route inside the YAML file.*/
    const isAllowed = await this.rateLimiter.consume(clientIp, 100, 10);

    if (!isAllowed) {
      logger.warn({ ip: clientIp }, "Traffic Dropped: Rate limit exceeded");
      clientRes.writeHead(429, {
        "Content-Type": "text/plain",
        "Retry-After": "1",
      });
      clientRes.end("429 Too Many Requests: Rate limit exceeded.");
      requestCounter.inc({ method, status: 429 });
      return; // Terminate the connection
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

  /* --------------------
     WEBSOCKET REQUEST HANDLER
   -------------------- */
  private async handleUpgrade(
    req: http.IncomingMessage,
    clientSocket: net.Socket,
    head: Buffer,
  ) {
    const clientIp = req.socket.remoteAddress || "unknown";

    // 1. Rate Limiting
    try {
      const isAllowed = await this.rateLimiter.consume(clientIp, 100, 10);
      if (!isAllowed) {
        logger.warn(
          { ip: clientIp, path: req.url },
          "Rate limit exceeded on WS upgrade",
        );
        clientSocket.write("HTTP/1.1 429 Too Many Requests\r\n\r\n");
        clientSocket.destroy();
        return;
      }
    } catch (err) {
      logger.error(
        { err },
        "Rate limiter failed during WS upgrade. Failing open.",
      );
    }

    // 2. Routing
    const targetUrl = req.url || "/";
    const backend = this.router.routeRequest(targetUrl);

    if (!backend) {
      logger.warn({ path: targetUrl }, "No upstream found for WS upgrade");
      clientSocket.write("HTTP/1.1 404 Not Found\r\n\r\n");
      clientSocket.destroy();
      return;
    }

    // 3. Open backend TCP socket
    const backendSocket = net.connect(backend.port, backend.host, () => {
      // 4. Construct and send the Upgrade headers to the backend
      let upgradeHeaders = `${req.method} ${req.url} HTTP/${req.httpVersion}\r\n`;
      for (let i = 0; i < req.rawHeaders.length; i += 2) {
        upgradeHeaders += `${req.rawHeaders[i]}: ${req.rawHeaders[i + 1]}\r\n`;
      }
      upgradeHeaders += "\r\n"; // Terminate headers

      backendSocket.write(upgradeHeaders);
      if (head && head.length > 0) {
        backendSocket.write(head); // Write the first chunk of data if the client sent any
      }

      // 5. Pipe the streams bidirectionally
      clientSocket.pipe(backendSocket);
      backendSocket.pipe(clientSocket);

      // --- METRICS: Connection established ---
      wsUpgradesTotal.inc({ status: "success" });
      activeWebSockets.inc();
    });

    // 6. Error Handling (Prevent Worker Crashes)
    backendSocket.on("error", (err) => {
      logger.error(
        { err, backend: `${backend.host}:${backend.port}` },
        "Backend WS socket error",
      );
      clientSocket.write("HTTP/1.1 502 Bad Gateway\r\n\r\n");
      clientSocket.destroy();
    });

    clientSocket.on("error", (err) => {
      logger.error({ err }, "Client WS socket error");
      backendSocket.destroy();
    });

    // --- METRICS: Connection Closed ---
    clientSocket.on("close", () => {
      activeWebSockets.dec();
    });
  }
}
