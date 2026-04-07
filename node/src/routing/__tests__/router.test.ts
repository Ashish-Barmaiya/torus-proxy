import { Router } from "../router.js";
import { BackendPool } from "../pool.js";
import { BackendServer } from "../backend.js";
import { RoundRobinStrategy } from "../roundRobin.js";

describe("Router - Longest Prefix Match Routing", () => {
  let router: Router;
  let rootPool: BackendPool;
  let apiPool: BackendPool;
  let v1Pool: BackendPool;

  beforeEach(() => {
    router = new Router();

    rootPool = new BackendPool(new RoundRobinStrategy());
    rootPool.addServer(new BackendServer("127.0.0.1", 8000));

    apiPool = new BackendPool(new RoundRobinStrategy());
    apiPool.addServer(new BackendServer("127.0.0.1", 8001));

    v1Pool = new BackendPool(new RoundRobinStrategy());
    v1Pool.addServer(new BackendServer("127.0.0.1", 8002));

    // Registering overlapping routes to test the matching logic
    router.addRoute("/", rootPool);
    router.addRoute("/api", apiPool);
    router.addRoute("/api/v1", v1Pool);
  });

  it("should route exact deep paths to the longest matching prefix", () => {
    // /api/v1/users should hit v1Pool (Port 8002) because '/api/v1' is a longer match than '/api'
    const server = router.routeRequest("/api/v1/users");
    expect(server?.port).toBe(8002);
  });

  it("should fall back to a shorter prefix if the deep prefix does not match completely", () => {
    // /api/v2/users does not match '/api/v1', so it must fall back to '/api' (Port 8001)
    const server = router.routeRequest("/api/v2/users");
    expect(server?.port).toBe(8001);
  });

  it("should route to the root catch-all if no specific prefix matches", () => {
    // /images/logo.png matches neither '/api' nor '/api/v1'. Must hit '/' (Port 8000)
    const server = router.routeRequest("/images/logo.png");
    expect(server?.port).toBe(8000);
  });

  it("should return null if no route matches and no catch-all exists", () => {
    // Create an empty router with no '/' catch-all route
    const emptyRouter = new Router();
    emptyRouter.addRoute("/secure", apiPool);

    const server = emptyRouter.routeRequest("/public/data");
    expect(server).toBeNull();
  });
});
