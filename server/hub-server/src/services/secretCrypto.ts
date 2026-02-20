import { createCipheriv, randomBytes } from "node:crypto";

import { HubServerError } from "../errors";

const ENCRYPTION_VERSION = "enc:v1";
const ENCRYPTION_ALGO = "aes-256-gcm";

function loadKeyFromBase64(rawKey: string): Buffer {
  const trimmed = rawKey.trim();
  if (!trimmed) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Hub secret key is not configured.",
      retryable: false,
      statusCode: 500,
      details: {
        config_key: "GOYAIS_HUB_SECRET_KEY"
      },
      causeType: "hub_secret_key_missing"
    });
  }

  let key: Buffer;
  try {
    key = Buffer.from(trimmed, "base64");
  } catch {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Hub secret key is invalid.",
      retryable: false,
      statusCode: 500,
      details: {
        config_key: "GOYAIS_HUB_SECRET_KEY",
        expected: "base64-encoded 32-byte key"
      },
      causeType: "hub_secret_key_parse"
    });
  }

  if (key.length !== 32) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Hub secret key is invalid.",
      retryable: false,
      statusCode: 500,
      details: {
        config_key: "GOYAIS_HUB_SECRET_KEY",
        expected: "base64-encoded 32-byte key",
        actual_bytes: key.length
      },
      causeType: "hub_secret_key_size"
    });
  }

  return key;
}

export function encryptApiKey(apiKey: string, rawKey: string): string {
  const key = loadKeyFromBase64(rawKey);
  const iv = randomBytes(12);
  const cipher = createCipheriv(ENCRYPTION_ALGO, key, iv);
  const encrypted = Buffer.concat([cipher.update(apiKey, "utf8"), cipher.final()]);
  const authTag = cipher.getAuthTag();

  return `${ENCRYPTION_VERSION}:${iv.toString("base64")}:${authTag.toString("base64")}:${encrypted.toString("base64")}`;
}
