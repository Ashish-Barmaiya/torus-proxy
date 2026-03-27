import cluster from "node:cluster";
import * as os from "node:os";
import * as fs from "node:fs";
import * as path from "node:path";
import { ProxyServer } from "./proxy/server.js";
import { buildRouterFromConfig } from "./config/parser.js";
import { HealthChecker } from "./routing/health.js";
import { logger } from "./utils/logger.js";
import { Lifecycle } from "./utils/lifecycle.js";

const configPath = path.resolve(process.cwd(), "torus.yaml");

if (cluster.isPrimary) {
  /**
   * -------------------
   * THE MASTER PROCESS
   * -------------------
   */

  // --- Graceful Shutdown ---
  let isShuttingDown = false;
  // 1. The Master Interceptor
  const initiateClusterTeardown = (signal: string) => {
    if (isShuttingDown) return;
    isShuttingDown = true;
    logger.info(
      { signal },
      "Master received termination signal. Broadcasting to workers...",
    );

    // Send the execution order to all living workers via IPC
    for (const id in cluster.workers) {
      cluster.workers[id]?.send({ type: "SHUTDOWN" });
    }
  };

  process.on("SIGTERM", () => initiateClusterTeardown("SIGTERM"));
  process.on("SIGINT", () => initiateClusterTeardown("SIGINT")); // For Ctrl+C in terminal

  // Testing Backdoor for Windows OS limitations
  process.on("message", (msg: any) => {
    if (msg && msg.type === "TEST_SHUTDOWN") {
      initiateClusterTeardown("TEST_SHUTDOWN");
    }
  });

  // 2. Halt Resurrection
  cluster.on("exit", (worker, code, signal) => {
    if (isShuttingDown) {
      logger.info(`Worker ${worker.process.pid} died cleanly during shutdown.`);

      // If the last worker just died, the Master can finally exit
      if (Object.keys(cluster.workers || {}).length === 0) {
        logger.info("All workers dead. Master exiting with code 0.");
        process.exit(0);
      }
    } else {
      // Normal operation: Worker crashed, boot a new one
      logger.warn(
        `Worker ${worker.process.pid} crashed. Forking a replacement...`,
      );
      cluster.fork();
    }
  });

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

  // --- THE WATCHER (Zero-Downtime Hot Reload) ---
  let debounceTimer: NodeJS.Timeout;

  fs.watch(configPath, (eventType) => {
    if (eventType === "change") {
      clearTimeout(debounceTimer);

      // Debounce the file system events by 500ms
      debounceTimer = setTimeout(() => {
        try {
          // DRY RUN: Parse the config in the Master first. If this throws an error, the catch blocks stop the reload.
          buildRouterFromConfig(configPath);
          const activeWorkers = Object.values(cluster.workers || {}); // If undefined, fallback to empty object
          logger.info(
            { workerCount: activeWorkers.length },
            "Configuration change detected and validated. Broadcasting hot-reload signal.",
          );

          // Broadcast the signal to all living workers via IPC
          activeWorkers.forEach((worker) => {
            worker?.send({ type: "RELOAD_CONFIG" });
          });
        } catch (error) {
          logger.error(
            {
              err: error,
            },
            "Hot-reload aborted: Invalid torus.yaml configuration.",
          );
        }
      }, 500);
    }
  });
} else {
  /**-------------------
   * THE WORKER PROCESS
   * -------------------
   */
  try {
    // 3. The Worker Trigger
    process.on("message", (msg: any) => {
      if (msg && msg.type === "SHUTDOWN") {
        Lifecycle.executeTeardown("IPC_SHUTDOWN");
      }
    });

    // Fallback: If the worker receives a direct OS signal somehow
    process.on("SIGTERM", () => Lifecycle.executeTeardown("SIGTERM"));
    process.on("SIGINT", () => Lifecycle.executeTeardown("SIGINT"));
    // 1. Build the routing state machine
    const { router, port, servers } = buildRouterFromConfig(configPath);

    // 2. Start health checks
    /**
     * Currently, the health checker runs for every worker process.
     * This is not the most efficient way to do it, but it works for now.
     *
     * TODO: Implement a global health checker that runs in the master process using Inter-Process Communication.
     */
    let healthChecker = new HealthChecker(servers);
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
        `Worker ${process.pid} listening for Secure HTTPS traffic`,
      );
    });

    // 7. --- IPC LISTENER (Hot Swap Logic) ---
    process.on("message", (msg: any) => {
      if (msg.type === "RELOAD_CONFIG") {
        logger.info(
          { pid: process.pid, msg: msg },
          "IPC Bridge: Message received from Master",
        );
        if (msg && msg.type === "RELOAD_CONFIG") {
          logger.info({ pid: process.pid }, "Worker swapping state...");
          try {
            // 1. Parse the new configuration
            const newConfig = buildRouterFromConfig(configPath);

            // 2. Safely terminate the old health checker loop
            healthChecker.stop();

            // 3. Boot the new health checker
            healthChecker = new HealthChecker(newConfig.servers);
            healthChecker.start();

            // 4. Hot-swap the router in the active TCP server
            proxy.updateRouter(newConfig.router);

            logger.info(
              { pid: process.pid },
              "Worker successfully hot-reloaded routing state without dropping connections.",
            );
          } catch (err: any) {
            logger.error(
              { err, pid: process.pid },
              "Worker failed to hot-reload state.",
            );
          }
        }
      }
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
