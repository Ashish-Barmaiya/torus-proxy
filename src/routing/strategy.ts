import { BackendServer } from "./backend.js";

export interface ILoadBalancingStrategy {
  // Selects the next appropriate backend server from a list of healthy backend servers
  getNext(servers: BackendServer[]): BackendServer | null; // returns null if no servers are available
}
