import { BackendServer } from "./backend.js";
import type { ILoadBalancingStrategy } from "./strategy.js";

export class BackendPool {
  private _servers: BackendServer[] = [];
  private _strategy: ILoadBalancingStrategy;

  constructor(strategy: ILoadBalancingStrategy) {
    this._strategy = strategy;
  }

  addServer(server: BackendServer): void {
    this._servers.push(server);
  }

  getServers(): BackendServer[] {
    return this._servers;
  }

  getServerForRequest(): BackendServer | null {
    // Filter healthy servers
    const healthyServers = this._servers.filter((s) => s.isHealthy);
    if (healthyServers.length === 0) return null;

    // Delegate selection to the injected strategy
    return this._strategy.getNext(healthyServers);
  }
}
