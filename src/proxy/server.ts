import * as http from "node:http";
import { Router } from "../routing/router.js";

export class ProxyServer {
  private router: Router;
  private server: http.Server;

  constructor(router: Router) {
    this.router = router;
    this.server = http.createServer(this.handleRequest.bind(this));
  }

  public listen(port: number, callback?: () => void) {
    this.server.listen(port, callback);
  }

  private handleRequest(
    clientReq: http.IncomingMessage,
    clientRes: http.ServerResponse,
  ) {
    // 1. Get the destination url from client request
    const url = clientReq.url || "/";

    // 2. Ask the router for a destination
    const targetServer = this.router.routeRequest(url);

    // 3. If no destination, return 502 and end connection
    if (!targetServer) {
      clientRes.writeHead(502, { "Content-Type": "text/plain" });
      clientRes.end("502 Bad Gateway: No available upstream servers.");
      return;
    }

    // 4. Construct the outbound request options
    const options: http.RequestOptions = {
      hostname: targetServer.host,
      port: targetServer.port,
      path: url,
      method: clientReq.method,
      headers: {
        ...clientReq.headers,
        "x-forwarded-for": clientReq.socket.remoteAddress || "", // This injects the original client IP
      },
    };

    // 5. Create a outbound request for the target server
    const proxyReq = http.request(options, (proxyRes: http.IncomingMessage) => {
      // Forward the server's http status code and headers to client
      clientRes.writeHead(proxyRes.statusCode || 200, proxyRes.headers);

      // PIPE: Stream the server's response directly to the client
      proxyRes.pipe(clientRes);

      proxyRes.on("error", (err: Error) => {
        console.error(`[Proxy Response Error] ${err.message}`);
        clientRes.end();
      });
    });

    // 6. Handle errors on the outbound connection (e.g., if backend server collapses)
    proxyReq.on("error", (err: Error) => {
      console.error(
        `[Proxy Request Error] Failed to connect to ${targetServer.host}:${targetServer.port} - ${err.message}`,
      );
      // If headers are not already sent, send a 502
      if (!clientRes.headersSent) {
        clientRes.writeHead(502, { "Content-Type": "text/plain" });
      }
      // End the connection
      clientRes.end("502 Bad Gateway");
    });

    // 7. PIPE: Stream the incoming client payload directly to the server
    clientReq.pipe(proxyReq);

    // 8. Handle errors on the inbound connection (e.g., if client disconnects)
    clientReq.on("error", (err: Error) => {
      console.error(`[Client Request Error] ${err.message}`);
      proxyReq.destroy();
    });
  }
}
