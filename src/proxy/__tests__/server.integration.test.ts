import { jest } from "@jest/globals";
import http from "node:http";
import https from "node:https";
import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import os from "node:os";

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

  // Helper to bypass strict TLS engine
  const fetchInsecure = (
    url: string,
  ): Promise<{ status: number; body: string }> => {
    return new Promise((resolve, reject) => {
      https
        .get(url, { rejectUnauthorized: false }, (res) => {
          let data = "";
          res.on("data", (chunk) => (data += chunk));
          res.on("end", () =>
            resolve({ status: res.statusCode || 500, body: data }),
          );
        })
        .on("error", reject);
    });
  };

  beforeAll(() => {
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
    // Then close the proxy
    const closeProxy = () => {
      if (proxyServer && proxyServer.server) {
        proxyServer.server.close(done);
        proxyServer = null;
      } else {
        done();
      }
    };

    // Close backend first
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
});
