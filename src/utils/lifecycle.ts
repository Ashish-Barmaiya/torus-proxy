import { logger } from "./logger.js";

type TeardownTask = () => Promise<void> | void;

class LifecycleManager {
  private tasks: TeardownTask[] = [];
  private isShuttingDown = false;
  private readonly SHUTDOWN_TIMEOUT_MS = 10000; // The Guillotine timer (10 sec)

  // Register a teardown task
  public onShutdown(task: TeardownTask) {
    this.tasks.push();
  }

  /**
   * Executes all registered teardown tasks
   * If they take longer than SHUTDOWN_TIMEOUT_MS, it forces an exit
   */
  public async executeTeardown(signal: string) {
    // Prevent double execution if multiple signals fire
    if (this.isShuttingDown) {
      logger.warn(
        { signal },
        "Shutdown already in progress. Ignoring duplicate signal.",
      );
      return;
    }

    this.isShuttingDown = true;
    logger.info({ signal }, "Initiating graceful shutdown sequence...");

    // 1. The Drain Promise
    const drainPromise = Promise.all(
      this.tasks.map(async (task) => {
        try {
          await task;
        } catch (err) {
          logger.error({ err }, "A teardown task failed during shutdown.");
        }
      }),
    );

    // 2. The Guillotine
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => {
        reject(
          new Error(`Shutdown timed out after ${this.SHUTDOWN_TIMEOUT_MS}ms`),
        );
      }, this.SHUTDOWN_TIMEOUT_MS).unref(); // unref() prevents this timer from keeping the event loop alive
    });

    // 3. The Execution
    try {
      await Promise.race([drainPromise, timeoutPromise]);
      logger.info("All systems cleanly drained. Exiting process gracefully.");
      process.exit(0);
    } catch (error: any) {
      logger.error(
        { err: error.message },
        "Graceful shutdown aborted. Forcing exit.",
      );
      process.exit(1);
    }
  }
}

export const Lifecycle = new LifecycleManager();
