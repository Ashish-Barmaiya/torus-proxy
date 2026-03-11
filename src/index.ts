import cluster from "node:cluster";
import * as os from "node:os";
import { Router } from "./routing/router.js";
import { BackendPool } from "./routing/pool.js";
import { BackendServer } from "./routing/backend.js";
import { RoundRobinStrategy } from "./routing/roundRobin.js";
import { ProxyServer } from "./proxy/server.js";

const PORT = 8080; // Using port 8080 for local dev

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

  // If a worker crashes instantly replace it.
  cluster.on("exit", (worker, code, signal) => {
    console.error(
      `[Master] Worker ${worker.process.pid} died. Booting a replacement...`,
    );
    cluster.fork();
  });
} else {
  // --- THE WORKER PROCESS ---
  // 1. Instantiate the Strategy
  const strategy = new RoundRobinStrategy();

  // 2. Instantiate the Pool and inject the Strategy
  const apiPool = new BackendPool(strategy);

  // 3. Create some dummy backend servers (will replace this with YAML parsing later)
  apiPool.addServer(new BackendServer("127.0.0.1", 3001));
  apiPool.addServer(new BackendServer("127.0.0.1", 3002));
  apiPool.addServer(new BackendServer("127.0.0.1", 3003));

  // 4. Instantiate the Router and map the URL prefix to the Pool
  const router = new Router();
  router.addRoute("/", apiPool); // Catch-all route for testing

  // 5. Instantiate the Server and bind the Router
  const proxy = new ProxyServer(router);

  // 6. Bind to the port
  proxy.listen(PORT, () => {
    console.log(
      `[Worker ${process.pid}] Listening for TCP/HTTP traffic on port ${PORT}`,
    );
  });
}
