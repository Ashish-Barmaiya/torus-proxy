import { spawn, type ChildProcess } from "node:child_process";
import * as path from "node:path";
import { jest, describe, it, expect, afterEach } from "@jest/globals";
import { fileURLToPath } from "node:url";
import * as fs from "node:fs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Jest timeout because booting clusters and waiting for timeouts takes time
jest.setTimeout(15000);

describe("Torus Cluster Lifecycle Architecture", () => {
  let masterProcess: ChildProcess;

  // Failsafe: Prevent zombie processes if a test fails
  afterEach(() => {
    if (masterProcess && masterProcess.exitCode === null) {
      masterProcess.kill("SIGKILL");
    }
  });

  it("should self-heal a violently killed worker, then gracefully shutdown on SIGINT", () => {
    return new Promise<void>((resolve, reject) => {
      const entryPoint = path.resolve(__dirname, "../../dist/index.js");
      const envPath = path.resolve(process.cwd(), ".env");

      const nodeArgs = fs.existsSync(envPath)
        ? ["--env-file=.env", entryPoint]
        : [entryPoint];

      // 1. Boot the Master process
      masterProcess = spawn("node", nodeArgs, {
        stdio: ["pipe", "pipe", "pipe", "ipc"],
      });

      let masterPid = masterProcess.pid;
      let targetWorkerPid: number | null = null;
      let workerResurrected = false;
      let shutdownInitiated = false;

      // 2. The Log Parser State Machine
      masterProcess.stdout!.on("data", (data) => {
        const output = data.toString();
        // console.log(output); // Raw logs - Uncomment to debug

        // STEP 1: Find a living worker PID
        if (
          !targetWorkerPid &&
          output.includes("listening for Secure HTTPS traffic")
        ) {
          // Extract PID from the string
          const match = output.match(/Worker\s+(\d+)\s+listening/);
          if (match) {
            targetWorkerPid = parseInt(match[1], 10);
            // Execute the assassination: Send SIGKILL (-9) to bypass graceful shutdown
            process.kill(targetWorkerPid, "SIGKILL");
          }
        }

        // STEP 2: Verify the Master caught the crash and resurrected it
        if (
          targetWorkerPid &&
          output.includes("crashed. Forking a replacement") &&
          !workerResurrected
        ) {
          workerResurrected = true; // Lock to prevent double firing

          // Give cluster 500ms to stabilize before dropping the axe
          setTimeout(() => {
            masterProcess.send({ type: "TEST_SHUTDOWN" });
          }, 500);
        }

        // STEP 3: Verify the Graceful Shutdown sequence started
        if (output.includes("Master received termination signal")) {
          shutdownInitiated = true;
        }
      });

      // 3. The Final Verdict
      masterProcess.on("close", (code) => {
        try {
          expect(workerResurrected).toBe(true);
          expect(shutdownInitiated).toBe(true);
          expect(code).toBe(0); // Master must exit with 0 (Clean)
          resolve();
        } catch (error) {
          reject(error);
        }
      });

      masterProcess.stderr?.on("data", (data) => {
        console.error(`STDERR: ${data.toString()}`);
      });
    });
  });
});
