# Torus Proxy Architecture Specification

Torus Proxy is a Layer 7 Reverse Proxy and Edge API Gateway built natively on Node.js. It is designed for high throughput, absolute memory safety, and dynamic configuration.

## 1. System Topology (The Cluster)
Node.js is single-threaded. Torus circumvents this by utilizing the `node:cluster` module.
* **The Master Process:** Binds to the host port, monitors the file system, and handles the `fs.watch` event loop. It does not route traffic.
* **The Worker Processes:** The Master spawns a worker for every physical CPU core. The OS TCP load balancer distributes incoming connections evenly across the Workers.

## 2. Zero-Downtime Hot Reloading (IPC)
Configuration updates (adding/removing backend servers) do not require a restart.
* The Master watches `torus.yaml`. Upon a validated change, it broadcasts an Inter-Process Communication (IPC) JSON payload to all Workers.
* Workers intercept the message, gracefully dismantle their existing `HealthChecker` instances, and inject the new `Router` into the active HTTP server memory space.
* **Result:** Zero dropped TCP sockets during topology updates.

## 3. The Security Edge
Torus acts as the absolute boundary between the hostile public internet and the trusted internal network.
* **TLS Termination:** Workers intercept incoming HTTPS traffic, perform the cryptographic handshake, decrypt the payload, and forward plain-text HTTP to the backend microservices, saving backend CPU cycles.
* **DDoS Shield (Rate Limiting):** Torus utilizes an atomic Token Bucket algorithm. To prevent race conditions across the 4 isolated Worker processes, the proxy executes a centralized Lua script directly inside a Redis database, ensuring microsecond-accurate global rate limiting.

## 4. The Routing Engine
Torus utilizes a **Longest-Prefix Match** algorithm to resolve URIs to upstream clusters. Routing tables are parsed dynamically from the `torus.yaml` specification.

## 5. Active Health Checking
Passive health checking (waiting for a user to hit a dead server) is unacceptable. Torus employs active probing.
* Every Worker runs an isolated `node:net` socket probe against the backend pool.
* If a server fails to acknowledge a TCP handshake within 2000ms, it is aggressively pruned from the Round Robin rotation.

## 6. Observability
* **Logging:** Emits strictly structured NDJSON via Pino for immediate ingestion by ELK or Datadog.
* **Telemetry:** Exposes a `/metrics` endpoint, providing real-time Prometheus gauges and histograms tracking Event Loop Lag, V8 Heap Size, and HTTP Latency.