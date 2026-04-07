import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { buildRouterFromConfig } from "../parser.js";

describe("YAML Config Parser", () => {
  let tmpDir: string;

  // Before the tests run, create a temporary directory on the host OS
  beforeAll(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "torus-test-"));
  });

  // After the tests finish, wipe the directory to keep the disk clean
  afterAll(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  // Helper function to write a YAML string to disk
  const writeConfig = (name: string, content: string) => {
    const filePath = path.join(tmpDir, name);
    fs.writeFileSync(filePath, content);
    return filePath;
  };

  it("should successfully parse a valid production configuration", () => {
    const validYaml = `
server:
  port: 8080
routes:
  - path: /api
    upstream: api_backend
upstreams:
  - name: api_backend
    servers:
      - host: 127.0.0.1
        port: 3001
      - host: 127.0.0.1
        port: 3002
    `;
    const configPath = writeConfig("valid.yaml", validYaml);

    // Assert the parser accepts the correct schema without throwing
    expect(() => buildRouterFromConfig(configPath)).not.toThrow();
  });

  it("should throw a fatal error if the routes block is missing", () => {
    const invalidYaml = `
server:
  port: 8080
upstreams:
  - name: api_backend
    servers:
      - host: 127.0.0.1
        port: 3001
    `;
    const configPath = writeConfig("missing-routes.yaml", invalidYaml);

    // The parser should physically block this from loading into memory
    expect(() => buildRouterFromConfig(configPath)).toThrow();
  });

  it("should throw a fatal error if the upstreams block is missing", () => {
    const invalidYaml = `
server:
  port: 8080
routes:
  - path: /api
    upstream: api_backend
    `;
    const configPath = writeConfig("missing-upstreams.yaml", invalidYaml);

    expect(() => buildRouterFromConfig(configPath)).toThrow();
  });
});
