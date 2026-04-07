import { BackendServer } from "./backend.js";
import { BackendPool } from "./pool.js";

export class Router {
  // Map of all the routes
  private routes: Map<string, BackendPool> = new Map();

  addRoute(prefix: string, pool: BackendPool): void {
    this.routes.set(prefix, pool);
  }

  // Route a request to the appropriate backend server
  routeRequest(url: string): BackendServer | null {
    let bestMatchPrefix = "";
    let selectedPool: BackendPool | null = null;

    for (const [prefix, pool] of this.routes.entries()) {
      if (url.startsWith(prefix) && prefix.length > bestMatchPrefix.length) {
        bestMatchPrefix = prefix;
        selectedPool = pool;
      }
    }

    // If no routes matches the incoming route url, return null
    if (!selectedPool) {
      return null;
    }

    return selectedPool.getServerForRequest();
  }
}
