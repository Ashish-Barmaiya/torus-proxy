import { createClient, type RedisClientType } from "redis";
import { logger } from "../utils/logger.js";

export class RedisRateLimiter {
  private client: RedisClientType;

  // --- Lua script ---
  // No two workers can deduct a token at the exact same microsecond.
  private readonly LUA_SCRIPT = `
    local key = KEYS[1]
    local capacity = tonumber(ARGV[1])
    local refillRatePerSecond = tonumber(ARGV[2])
    local now = tonumber(ARGV[3])
    
    local bucket = redis.call("HMGET", key, "tokens", "last_refill")
    local tokens = tonumber(bucket[1])
    local last_refill = tonumber(bucket[2])
    
    -- If bucket doesn't exist, initialize it
    if not tokens then
      tokens = capacity
      last_refill = now
    else
      -- Calculate how many tokens to add based on elapsed time
      local time_passed = math.max(0, now - last_refill)
      local refill_amount = math.floor(time_passed * refillRatePerSecond)
      tokens = math.min(capacity, tokens + refill_amount)
      if refill_amount > 0 then
        last_refill = now
      end
    end
    
    -- Check if we have enough tokens to allow the request
    if tokens >= 1 then
      tokens = tokens - 1
      redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
      -- Set TTL so we don't leak memory for stale IPs
      redis.call("EXPIRE", key, math.ceil(capacity / refillRatePerSecond) * 2) 
      return 1 -- ALLOW
    else
      return 0 -- REJECT
    end
  `;
  constructor(redisUrl: string = "redis://127.0.0.1:6379") {
    this.client = createClient({ url: redisUrl });

    this.client.on("error", (err) =>
      logger.error({ err }, "Redis Client Error"),
    );

    this.client.connect().then(() => {
      logger.info("Rate Limiter securely connected to Redis cluster");
    });
  }

  /**
   * Evaluates the Token Bucket for a given IP.
   * @param ip The client IP address
   * @param capacity Maximum burst size
   * @param refillRate Tokens added per second
   * @returns boolean - True if allowed, False if rate limited
   */
  public async consume(
    ip: string,
    capacity: number,
    refillRate: number,
  ): Promise<boolean> {
    try {
      const now = Math.floor(Date.now() / 1000); // Current time in seconds

      // Execute the atomic Lua script in Redis
      const result = await this.client.sendCommand([
        "EVAL",
        this.LUA_SCRIPT,
        "1", // Number of keys
        `rate_limit:${ip}`, // KEYS[1]
        capacity.toString(), // ARGV[1]
        refillRate.toString(), // ARGV[2]
        now.toString(), // ARGV[3]
      ]);

      return Number(result) === 1;
    } catch (error) {
      logger.error(
        { err: error, ip },
        "Rate limiter failed to communicate with Redis. Failing open.",
      );
      return true; // If Redis dies, we fail OPEN to prevent taking down the whole proxy.
      /* Choosing AVAILABILITY over Secondary Security */
    }
  }

  public async disconnect() {
    await this.client.quit();
  }
}
