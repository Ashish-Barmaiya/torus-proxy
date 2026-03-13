import cluster from "node:cluster";
import * as os from "node:os";
import * as fs from "node:fs";
import * as path from "node:path";
import { ProxyServer } from "./proxy/server.js";
import { buildRouterFromConfig } from "./config/parser.js";
import { HealthChecker } from "./routing/health.js";
import { logger } from "./utils/logger.js";

if (cluster.isPrimary) {
  // --- THE MASTER PROCESS ---
  logger.info(
    {
      pid: process.pid,
    },
    "Master process is running",
  );

  // Count the machine's physical/logical CPU cores
  const numCPUs = os.cpus().length;
  logger.info(
    {
      cores: numCPUs,
    },
    "Booting worker processes to handle traffic",
  );

  // Fork a worker for every core
  for (let i = 0; i < numCPUs; i++) {
    cluster.fork();
  }

  // If a worker crashes instantly replace it
  cluster.on("exit", (worker, code, signal) => {
    logger.warn(
      {
        pid: worker.process.pid,
        exitCode: code,
        signal: signal,
      },
      "Worker died. Booting a replacement...",
    );
    cluster.fork();
  });
} else {
  // --- THE WORKER PROCESS ---
  try {
    // 1. Resolve the absolute path to the YAML file
    const configPath = path.resolve(process.cwd(), "torus.yaml");

    // 2. Build the routing state machine
    const { router, port, servers } = buildRouterFromConfig(configPath);

    // 3. Start health checks
    /**
     * Currently, the health checker runs for every worker process.
     * This is not the most efficient way to do it, but it works for now.
     *
     * TODO: Implement a global health checker that runs in the master process using Inter-Process Communication.
     */
    const healthChecker = new HealthChecker(servers);
    healthChecker.start();

    // 4. Load the Cryptographic Keys into Memory
    const tlsOptions = {
      key: fs.readFileSync(path.resolve(process.cwd(), "certs", "key.pem")),
      cert: fs.readFileSync(path.resolve(process.cwd(), "certs", "cert.pem")),
    };

    // 5. Inject cofigured Router into the raw http server
    const proxy = new ProxyServer(router, tlsOptions);

    // 6. Bind to the port
    proxy.listen(port, () => {
      logger.info(
        {
          pid: process.pid,
          port: port,
        },
        "Worker listening for Secure HTTPS traffic",
      );
    });
  } catch (error: any) {
    // If the YAML file is malformed, kill the process completely
    logger.error(
      {
        pid: process.pid,
        err: error,
      },
      "Worker boot failed",
    );
    process.exit(1);
  }
}
