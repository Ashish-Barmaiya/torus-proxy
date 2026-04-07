import { RoundRobinStrategy } from "../roundRobin.js";
import { BackendServer } from "../backend.js";

describe("RoundRobinStrategy - Core Logic", () => {
  let strategy: RoundRobinStrategy;
  let servers: BackendServer[];

  beforeEach(() => {
    strategy = new RoundRobinStrategy();
    servers = [
      new BackendServer("127.0.0.1", 8001),
      new BackendServer("127.0.0.1", 8002),
      new BackendServer("127.0.0.1", 8003),
    ];
  });

  it("should route sequentially across all healthy servers", () => {
    expect(strategy.getNext(servers)?.port).toBe(8001);
    expect(strategy.getNext(servers)?.port).toBe(8002);
    expect(strategy.getNext(servers)?.port).toBe(8003);
  });

  it("should loop back to the first server after reaching the end of the array", () => {
    strategy.getNext(servers); // 8001
    strategy.getNext(servers); // 8002
    strategy.getNext(servers); // 8003
    expect(strategy.getNext(servers)?.port).toBe(8001); // Loop back
  });

  it("should instantly skip dead servers and route to the next healthy node", () => {
    servers[1]!.markDead(); // Kill Port 8002

    expect(strategy.getNext(servers)?.port).toBe(8001);
    expect(strategy.getNext(servers)?.port).toBe(8003); // Skips 8002 completely
    expect(strategy.getNext(servers)?.port).toBe(8001); // Loops back
  });

  it("should return null if the pool is completely empty or all servers are dead", () => {
    servers[0]!.markDead();
    servers[1]!.markDead();
    servers[2]!.markDead();

    expect(strategy.getNext(servers)).toBeNull();
    expect(strategy.getNext([])).toBeNull();
  });
});
