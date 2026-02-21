import { randomBytes, scryptSync, timingSafeEqual } from "node:crypto";

const KEY_LEN = 64;

export function hashPassword(password: string): string {
  const salt = randomBytes(16);
  const derived = scryptSync(password, salt, KEY_LEN);
  return `scrypt$${salt.toString("base64")}$${derived.toString("base64")}`;
}

export function verifyPassword(password: string, encodedHash: string): boolean {
  const parts = encodedHash.split("$");
  if (parts.length !== 3 || parts[0] !== "scrypt") {
    return false;
  }

  const salt = Buffer.from(parts[1], "base64");
  const expected = Buffer.from(parts[2], "base64");
  const actual = scryptSync(password, salt, expected.length);
  return timingSafeEqual(actual, expected);
}
