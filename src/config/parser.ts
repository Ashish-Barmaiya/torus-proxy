import * as fs from "node:fs";
import * as yaml from "yaml";
import { Router } from "../routing/router.js";
import { BackendPool } from "../routing/pool.js";
import { BackendServer } from "../routing/backend.js";
import { RoundRobinStrategy } from "../routing/roundRobin.js";

// Strict typing for the expected YAML structure
interface ConfigStructure {
  server: { port: number };
  routes: { path: string; upstream: string }[];
  upstreams: { name: string; servers: { host: string; port: number }[] }[];
}

export function buildRouterFromConfig(filePath: string): {
  router: Router;
  port: number;
} {
  // 1. Read and parse the file synchronously.
  // Using synchronous process to block the thread here as this only happens once at startup.
  if (!fs.existsSync(filePath)) {
    throw new Error(`[Fatal] Configuration file not found at ${filePath}`);
  }

  const fileContent = fs.readFileSync(filePath, "utf8");
  const config = yaml.parse(fileContent) as ConfigStructure;

  // Basic validation
  if (!config.upstreams || !config.routes) {
    throw new Error(
      "[Fatal] Invalid configuration: Missing routes or upstreams definition.",
    );
  }

  const router = new Router();
  const pools = new Map<string, BackendPool>();

  // 2. Build the Upstream Pools
  for (const upstream of config.upstreams) {
    const strategy = new RoundRobinStrategy();
    const pool = new BackendPool(strategy);

    for (const serverConfig of upstream.servers) {
      pool.addServer(new BackendServer(serverConfig.host, serverConfig.port));
    }

    // Store the pool temporarily in a Map so servers can be linked to routes later
    pools.set(upstream.name, pool);
  }

  // 3. Map the Routes to the Pools
  for (const route of config.routes) {
    const targetPool = pools.get(route.upstream);

    if (!targetPool) {
      // This fails the startup and do not let the proxy boot if the config file has a typo or any other mistakes linking to a non-existent pool.
      throw new Error(
        `[Fatal] Route ${route.path} points to an undefined upstream: ${route.upstream}`,
      );
    }

    router.addRoute(route.path, targetPool);
  }

  return { router, port: config.server?.port || 8080 };
}
