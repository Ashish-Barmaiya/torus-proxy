import * as net from "node:net";
import { BackendServer } from "./backend.js";
import { logger } from "../utils/logger.js";

export class HealthChecker {
  private servers: Set<BackendServer>;
  private intervalMs: number;

  constructor(servers: BackendServer[], intervalMs: number = 10000) {
    // Runs every 10 seconds
    this.servers = new Set(servers);
    this.intervalMs = intervalMs;
  }

  public start(): void {
    logger.info(
      {
        serverCount: this.servers.size,
      },
      "Starting active monitoring for unique servers",
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
        logger.info(
          {
            host: server.host,
            port: server.port,
          },
          "Server recovered. Marking ALIVE.",
        );
        server.markAlive();
      }
      socket.destroy();
    });

    socket.once("timeout", () => {
      if (server.isHealthy) {
        logger.warn(
          {
            host: server.host,
            port: server.port,
          },
          "Health check timed out. Marking DEAD.",
        );
        server.markDead();
      }
      socket.destroy();
    });

    socket.once("error", (err) => {
      if (server.isHealthy) {
        logger.warn(
          {
            host: server.host,
            port: server.port,
            err: err,
          },
          "Health check failed. Marking DEAD.",
        );
        server.markDead();
      }
      socket.destroy();
    });

    socket.connect(server.port, server.host);
  }
}
