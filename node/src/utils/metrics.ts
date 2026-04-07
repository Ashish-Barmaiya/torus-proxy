import client from "prom-client";

export const register = new client.Registry();

// This adds default metrics automatically (like CPU, RAM, event loop delay)
client.collectDefaultMetrics({
  register,
});

// 1. Counter: Increments every time a request is made
export const requestCounter = new client.Counter({
  name: "torus_http_requests_total",
  help: "Total number of HTTP requests processed by the proxy",
  labelNames: ["method", "status"],
  registers: [register],
});

// 2. Histogram: How long do requests take?
export const requestDurationHistogram = new client.Histogram({
  name: "torus_http_request_duration_seconds",
  help: "Latency of HTTP requests in seconds",
  labelNames: ["method", "status"],
  registers: [register],
  buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
});

// 3. Guage: Active WebSockets
export const activeWebSockets = new client.Gauge({
  name: "torus_ws_active_connections",
  help: "Current number of active WebSocket connections",
  registers: [register],
});

// 4. Counter: Total WebSocket Connections
export const wsUpgradesTotal = new client.Counter({
  name: "torus_ws_upgrades_total",
  help: "Total number of WebSocket protocol upgrade requests",
  labelNames: ["status"], // 'success', 'rejected_rate_limit', 'rejected_not_found'
  registers: [register],
});
