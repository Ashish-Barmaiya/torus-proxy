import { jest } from "@jest/globals";
import http from "node:http";
import https from "node:https";
import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import os from "node:os";
import tls from "node:tls";
import net from "node:net";
import crypto from "node:crypto";

// 1. Mock Redis so the test doesn't fail if Docker isn't running
jest.unstable_mockModule("../../security/rateLimiter.js", () => ({
  RedisRateLimiter: class {
    consume = (
      jest.fn() as jest.Mock<() => Promise<boolean>>
    ).mockResolvedValue(true); // Always allow traffic
    disconnect = jest.fn();
  },
}));

// Mute the logger to keep test output clean
jest.unstable_mockModule("../../utils/logger.js", () => ({
  logger: { info: jest.fn(), error: jest.fn(), warn: jest.fn() },
}));

const { ProxyServer } = await import("../server.js");
const { Router } = await import("../../routing/router.js");
const { BackendPool } = await import("../../routing/pool.js");
const { BackendServer } = await import("../../routing/backend.js");
const { RoundRobinStrategy } = await import("../../routing/roundRobin.js");

describe("ProxyServer Integration Network Tests", () => {
  let backendServer: http.Server;
  let proxyServer: any;
  let backendPort: number;
  let proxyPort: number = 8081;
  let tlsOptions: any;
  let tmpDir: string;
  let activeSockets = new Set<net.Socket>();

  const TEST_SECRET = "test-secret-key-123";
  let validJwt: string;

  // Helper to bypass strict TLS engine
  const fetchInsecure = (
    url: string,
    customHeaders: Record<string, string> = {},
  ): Promise<{ status: number; body: string }> => {
    return new Promise((resolve, reject) => {
      const parsedUrl = new URL(url);

      const options = {
        hostname: parsedUrl.hostname,
        port: parsedUrl.port,
        path: parsedUrl.pathname,
        method: "GET",
        rejectUnauthorized: false,
        headers: customHeaders,
      };

      const req = https.request(options, (res) => {
        let data = "";
        res.on("data", (chunk) => (data += chunk));
        res.on("end", () =>
          resolve({ status: res.statusCode || 500, body: data }),
        );
      });

      req.on("error", reject);
      req.end();
    });
  };

  beforeAll(() => {
    // Inject the environment variable so the proxy can boot
    process.env.JWT_SECRET = TEST_SECRET;

    // Mathematically forge a valid JWT
    const header = Buffer.from(
      JSON.stringify({ alg: "HS256", typ: "JWT" }),
    ).toString("base64url");
    // Set expiration 1 hour into the future
    const payload = Buffer.from(
      JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600 }),
    ).toString("base64url");
    const signature = crypto
      .createHmac("sha256", TEST_SECRET)
      .update(`${header}.${payload}`)
      .digest("base64url");
    validJwt = `Bearer ${header}.${payload}.${signature}`;

    // Dynamically generate a REAL x509 certificate using OpenSSL
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "torus-tls-"));
    const keyPath = path.join(tmpDir, "key.pem");
    const certPath = path.join(tmpDir, "cert.pem");

    execSync(
      `openssl req -x509 -newkey rsa:2048 -keyout "${keyPath}" -out "${certPath}" -days 1 -nodes -subj "/CN=localhost"`,
      { stdio: "ignore" },
    );

    tlsOptions = {
      key: fs.readFileSync(keyPath),
      cert: fs.readFileSync(certPath),
    };

    // Tell Node to accept our self-signed cert for local fetch requests
    process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
  });

  afterAll(() => {
    delete process.env.NODE_TLS_REJECT_UNAUTHORIZED;
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  afterEach((done) => {
    /// 1. Destroy all tracked rogue sockets first
    for (const socket of activeSockets) {
      if (!socket.destroyed) socket.destroy();
    }
    activeSockets.clear();

    // 2. Then close the proxy gracefully
    const closeProxy = () => {
      if (proxyServer && proxyServer.server) {
        proxyServer.server.close(done);
        proxyServer = null;
      } else {
        done();
      }
    };

    // 3. Close backend gracefully
    if (backendServer) {
      backendServer.close(closeProxy);
      backendServer = null as any;
    } else {
      closeProxy();
    }
  });

  it("should successfully proxy an HTTP request to the backend", (done) => {
    // 1. Boot the Dummy Backend
    backendServer = http.createServer((req, res) => {
      expect(req.headers["x-forwarded-proto"]).toBe("https");
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ success: true, message: "Routed via Torus" }));
    });

    backendServer.listen(0, () => {
      backendPort = (backendServer.address() as any).port;

      // 2. Configure Torus Routing
      const backend = new BackendServer("127.0.0.1", backendPort);
      const pool = new BackendPool(new RoundRobinStrategy());
      pool.addServer(backend);

      const router = new Router();
      router.addRoute("/api", pool);

      // 3. Boot Torus Proxy
      proxyServer = new ProxyServer(router, tlsOptions);
      proxyServer.server.listen(proxyPort, async () => {
        // 4. Fire the actual network request
        try {
          const response = await fetchInsecure(
            `https://127.0.0.1:${proxyPort}/api/data`,
            { Authorization: validJwt },
          );
          const body = JSON.parse(response.body);
          expect(response.status).toBe(200);
          expect(body.message).toBe("Routed via Torus");
          done();
        } catch (err) {
          done(err);
        }
      });
    });
  });

  it("should catch ECONNREFUSED and return 502 Bad Gateway when backend is dead", (done) => {
    // Configure Torus to point to a dead port (e.g., 9999)
    const deadBackend = new BackendServer("127.0.0.1", 9999);
    const pool = new BackendPool(new RoundRobinStrategy());
    pool.addServer(deadBackend);

    const router = new Router();
    router.addRoute("/api", pool);

    proxyServer = new ProxyServer(router, tlsOptions);
    proxyServer.server.listen(proxyPort + 1, async () => {
      try {
        const response = await fetchInsecure(
          `https://127.0.0.1:${proxyPort + 1}/api/data`,
          { Authorization: validJwt },
        );

        // Torus MUST not crash. It must return 502.
        expect(response.status).toBe(502);
        expect(response.body).toBe("502 Bad Gateway");

        done();
      } catch (err) {
        done(err);
      }
    });
  });

  it("should successfully proxy a Layer 4 WebSocket upgrade and stream bidirectional TCP data", (done) => {
    // 1. Boot the Dummy Backend to act as a raw WebSocket server
    backendServer = http.createServer();

    // Track every raw socket that touches this dummy server
    backendServer.on("connection", (socket) => {
      activeSockets.add(socket);
      socket.on("close", () => activeSockets.delete(socket));
    });

    backendServer.on("upgrade", (req, socket, head) => {
      // Approve the protocol upgrade
      socket.write(
        "HTTP/1.1 101 Switching Protocols\r\n" +
          "Upgrade: websocket\r\n" +
          "Connection: Upgrade\r\n" +
          "\r\n",
      );

      // Echo any raw bytes back to the client
      socket.on("data", (chunk) => {
        if (chunk.toString() === "PING") socket.write("PONG");
      });
    });

    backendServer.listen(0, () => {
      backendPort = (backendServer.address() as any).port;

      // 2. Configure Torus Routing
      const backend = new BackendServer("127.0.0.1", backendPort);
      const pool = new BackendPool(new RoundRobinStrategy());
      pool.addServer(backend);

      const router = new Router();
      router.addRoute("/ws", pool);

      // 3. Boot Torus Proxy
      proxyServer = new ProxyServer(router, tlsOptions);
      proxyServer.server.listen(proxyPort + 2, () => {
        // Use port + 2 to avoid collisions

        // 4. Fire a raw TLS Socket at Torus (Bypassing HTTP entirely)
        const client = tls.connect(
          proxyPort + 2,
          "127.0.0.1",
          { rejectUnauthorized: false },
          () => {
            // Manually construct the HTTP Upgrade payload
            client.write(
              "GET /ws HTTP/1.1\r\n" +
                "Host: 127.0.0.1\r\n" +
                "Connection: Upgrade\r\n" +
                "Upgrade: websocket\r\n" +
                `Authorization: ${validJwt}\r\n` +
                "\r\n",
            );
          },
        );

        let handshakeComplete = false;

        // 5. Assert the Bidirectional Pipe
        client.on("data", (chunk) => {
          const response = chunk.toString();

          if (!handshakeComplete) {
            // Assert Torus successfully proxied the 101 response from the backend
            expect(response).toContain("101 Switching Protocols");
            handshakeComplete = true;

            // Send raw Layer 4 TCP data through the proxy
            client.write("PING");
          } else {
            // Assert Torus successfully proxied the response back from the backend
            expect(response).toBe("PONG");

            // Clean up sockets
            client.destroy();
            done();
          }
        });

        client.on("error", (err) => done(err));
      });
    });
  });

  it("should aggressively reject an HTTP request with a forged JWT signature", (done) => {
    // 1. Boot the Dummy Backend
    backendServer = http.createServer((req, res) => {
      // If the authenticator fails, this code executes and the test fails.
      res.writeHead(200);
      res.end("CRITICAL FAILURE: Backend reached by unauthorized traffic");
    });

    backendServer.listen(0, () => {
      backendPort = (backendServer.address() as any).port;

      // 2. Configure Torus Routing
      const backend = new BackendServer("127.0.0.1", backendPort);
      const pool = new BackendPool(new RoundRobinStrategy());
      pool.addServer(backend);

      const router = new Router();
      router.addRoute("/api", pool);

      // 3. Boot Torus Proxy on port + 3 to avoid collisions
      proxyServer = new ProxyServer(router, tlsOptions);
      proxyServer.server.listen(proxyPort + 3, async () => {
        // 4. Mathematically forge a malicious token
        const header = Buffer.from(
          JSON.stringify({ alg: "HS256", typ: "JWT" }),
        ).toString("base64url");
        const maliciousPayload = Buffer.from(
          JSON.stringify({
            exp: Math.floor(Date.now() / 1000) + 3600,
            admin: true,
          }),
        ).toString("base64url");

        // Attacker signs it with their own random secret
        const forgedSignature = crypto
          .createHmac("sha256", "HACKER_SECRET_KEY")
          .update(`${header}.${maliciousPayload}`)
          .digest("base64url");
        const forgedJwt = `Bearer ${header}.${maliciousPayload}.${forgedSignature}`;

        // 5. Fire the attack
        try {
          const response = await fetchInsecure(
            `https://127.0.0.1:${proxyPort + 3}/api/data`,
            { Authorization: forgedJwt },
          );

          const body = JSON.parse(response.body);

          // 6. Assert the authenticator did its job
          expect(response.status).toBe(401);
          expect(body.error).toBe("Unauthorized");

          done();
        } catch (err) {
          done(err);
        }
      });
    });
  });
});
