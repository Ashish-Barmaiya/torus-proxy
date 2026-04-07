import crypto from "node:crypto";
import { logger } from "../utils/logger.js";

export class JwtAuthenticator {
  private secret: string;

  constructor(secret: string) {
    if (!secret) {
      throw new Error("JWT Secret is required to boot JWT Authenticator.");
    }
    this.secret = secret;
  }

  /**
   * Cryptographically verifies a JWT from an Authorization header.
   * Returns true if valid, false if invalid, forged, or expired.
   */
  public verify(authHeader?: string): boolean {
    if (!authHeader || !authHeader.startsWith("Bearer ")) {
      logger.warn(
        "JWT signature verification failed. Possible forgery attempt.",
      );
      return false;
    }

    const token = authHeader.split(" ")[1];
    if (!token) {
      logger.warn(
        "JWT signature verification failed. Possible forgery attempt.",
      );
      return false;
    }

    const parts = token.split(".");
    if (parts.length !== 3) {
      logger.warn(
        "JWT signature verification failed. Possible forgery attempt.",
      );
      return false;
    }

    const header = parts[0];
    const payload = parts[1];
    const signature = parts[2];

    if (!header || !payload || !signature) {
      logger.warn(
        "JWT signature verification failed. Possible forgery attempt.",
      );
      return false;
    }

    // 1. Cryptographic Signature Verification - Re-hash the header and payload using our secret key.
    const expectedSignature = crypto
      .createHmac("sha256", this.secret)
      .update(`${header}.${payload}`)
      .digest("base64url");

    // If the attacker tampered with the payload, the hashes will fail to match.
    if (signature !== expectedSignature) {
      logger.warn(
        "JWT signature verification failed. Possible forgery attempt.",
      );
      return false;
    }

    // 2. Lifecycle Verification (Expiration)
    try {
      const decodedPayload = JSON.parse(
        Buffer.from(payload, "base64url").toString("utf8"),
      );

      if (decodedPayload.exp) {
        const now = Math.floor(Date.now() / 1000);
        if (now > decodedPayload.exp) {
          logger.warn("JWT token expired.");
          return false;
        }
      }
    } catch (err) {
      logger.error("Failed to parse JWT payload");
      return false;
    }

    return true;
  }
}
