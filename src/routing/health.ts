// src/routing/health.ts
import * as net from "node:net";
import { BackendServer } from "./backend.js";

export class HealthChecker {
  private servers: Set<BackendServer>;
  private intervalMs: number;

  constructor(servers: BackendServer[], intervalMs: number = 10000) {
    // Runs every 10 seconds
    this.servers = new Set(servers);
    this.intervalMs = intervalMs;
  }

  public start(): void {
    console.log(
      `[HealthChecker] Starting active monitoring for ${this.servers.size} unique servers...`,
    );

    setInterval(() => {
      this.servers.forEach((server) => this.ping(server));
    }, this.intervalMs);
  }

  private ping(server: BackendServer): void {
    const socket = new net.Socket();
    socket.setTimeout(2000); // If a server takes longer than 2 seconds to respond consider it dead.

    socket.once("connect", () => {
      if (!server.isHealthy) {
        console.log(
          `[Health] ${server.host}:${server.port} recovered. Marking ALIVE.`,
        );
        server.markAlive();
      }
      socket.destroy();
    });

    socket.once("timeout", () => {
      if (server.isHealthy) {
        console.error(
          `[Health] ${server.host}:${server.port} TIMED OUT. Marking DEAD.`,
        );
        server.markDead();
      }
      socket.destroy();
    });

    socket.once("error", (err) => {
      if (server.isHealthy) {
        console.error(
          `[Health] ${server.host}:${server.port} FAILED (${err.message}). Marking DEAD.`,
        );
        server.markDead();
      }
      socket.destroy();
    });

    socket.connect(server.port, server.host);
  }
}
