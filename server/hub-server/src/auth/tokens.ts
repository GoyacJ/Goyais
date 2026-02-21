import { createHash, randomBytes, randomUUID } from "node:crypto";

export interface IssuedToken {
  tokenId: string;
  token: string;
  tokenHash: string;
  expiresAt: string;
  createdAt: string;
}

export function hashToken(token: string): string {
  return createHash("sha256").update(token).digest("hex");
}

export function issueOpaqueToken(tokenTtlSeconds: number, now: Date = new Date()): IssuedToken {
  const token = randomBytes(32).toString("base64url");
  const createdAt = now.toISOString();
  const expiresAt = new Date(now.getTime() + tokenTtlSeconds * 1000).toISOString();

  return {
    tokenId: randomUUID(),
    token,
    tokenHash: hashToken(token),
    expiresAt,
    createdAt
  };
}
