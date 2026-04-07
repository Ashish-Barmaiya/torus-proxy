import { jest } from "@jest/globals";

// 1. Mock the external dependencies before importing our code
jest.unstable_mockModule("redis", () => ({
  createClient: jest.fn(),
}));

jest.unstable_mockModule("../../utils/logger.js", () => ({
  logger: {
    info: jest.fn(),
    error: jest.fn(),
    warn: jest.fn(),
  },
}));

// 2. Import the module under test dynamically after mocking
const { RedisRateLimiter } = await import("../rateLimiter.js");
const { createClient } = await import("redis");

describe("RedisRateLimiter", () => {
  let mockSendCommand: jest.Mock<any>;
  let limiter: any;

  beforeEach(() => {
    mockSendCommand = jest.fn();

    // Create a fake Redis client
    (createClient as jest.Mock).mockReturnValue({
      on: jest.fn(),
      connect: jest.fn<() => Promise<void>>().mockResolvedValue(undefined),
      sendCommand: mockSendCommand,
      disconnect: jest.fn(),
    });

    limiter = new RedisRateLimiter();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("should return true (allow) when the Lua script returns 1", async () => {
    mockSendCommand.mockResolvedValue(1);
    const result = await limiter.consume("192.168.1.100", 10, 2);

    expect(result).toBe(true);
    expect(mockSendCommand).toHaveBeenCalledTimes(1);
  });

  it("should return false (block) when the Lua script returns 0", async () => {
    mockSendCommand.mockResolvedValue(0);
    const result = await limiter.consume("192.168.1.100", 10, 2);

    expect(result).toBe(false);
  });

  it("should fail OPEN and return true if the Redis connection dies", async () => {
    // Simulate a hard network failure or database crash
    mockSendCommand.mockRejectedValue(new Error("ECONNREFUSED: Redis is dead"));

    const result = await limiter.consume("192.168.1.100", 10, 2);

    // The proxy MUST survive and let the traffic through
    expect(result).toBe(true);
  });
});
