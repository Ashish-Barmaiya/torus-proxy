// src/routing/roundRobin.ts
import { BackendServer } from "./backend.js";
import type { ILoadBalancingStrategy } from "./strategy.js";

export class RoundRobinStrategy implements ILoadBalancingStrategy {
  private _currentIndex: number = 0;

  getNext(servers: BackendServer[]): BackendServer | null {
    if (servers.length === 0) {
      return null;
    }

    // This loops the array by using the modulo operator
    const server = servers[this._currentIndex % servers.length];

    // Increment the counter for the next incoming request
    this._currentIndex++;

    if (this._currentIndex >= Number.MAX_SAFE_INTEGER) {
      this._currentIndex = 0;
    }

    return server ?? null;
  }
}
