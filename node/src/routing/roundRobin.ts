import { BackendServer } from "./backend.js";
import type { ILoadBalancingStrategy } from "./strategy.js";

export class RoundRobinStrategy implements ILoadBalancingStrategy {
  private currentIndex: number = 0;

  getNext(servers: BackendServer[]): BackendServer | null {
    if (servers.length === 0) {
      return null;
    }

    // Loop exactly server.length times
    for (let i = 0; i < servers.length; i++) {
      const server = servers[this.currentIndex];
      this.currentIndex = (this.currentIndex + 1) % servers.length;

      if (server && server.isHealthy) {
        return server;
      }
    }

    return null; // if loop ended and entire pool is unhealthy
  }
}
