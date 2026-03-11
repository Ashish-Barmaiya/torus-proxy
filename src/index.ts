import cluster from "node:cluster";
import * as os from "node:os";
import * as path from "node:path";
import { ProxyServer } from "./proxy/server.js";
import { buildRouterFromConfig } from "./config/parser.js";

if (cluster.isPrimary) {
  // --- THE MASTER PROCESS ---
  console.log(`[Master] Process ${process.pid} is running`);

  // Count the machine's physical/logical CPU cores
  const numCPUs = os.cpus().length;
  console.log(
    `[Master] Booting ${numCPUs} worker processes to handle traffic...`,
  );

  // Fork a worker for every core
  for (let i = 0; i < numCPUs; i++) {
    cluster.fork();
  }

  // If a worker crashes instantly replace it
  cluster.on("exit", (worker, code, signal) => {
    console.error(
      `[Master] Worker ${worker.process.pid} died. Booting a replacement...`,
    );
    cluster.fork();
  });
} else {
  // --- THE WORKER PROCESS ---
  try {
    // 1. Resolve the absolute path to the YAML file
    const configPath = path.resolve(process.cwd(), "torus.yaml");

    // 2. Build the routing state machine
    const { router, port } = buildRouterFromConfig(configPath);

    // 3. Inject cofigured Router into the raw http server
    const proxy = new ProxyServer(router);

    // 4. Bind to the port
    proxy.listen(port, () => {
      console.log(
        `[Worker ${process.pid}] Listening for TCP/HTTP traffic on port ${port}`,
      );
    });
  } catch (error: any) {
    // If the YAML file is malformed, kill the process completely
    console.error(`[Worker ${process.pid}] Boot failed: ${error.message}`);
    process.exit(1);
  }
}
