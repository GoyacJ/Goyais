import { createCipheriv, createDecipheriv, randomBytes } from "node:crypto";

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

export function decryptApiKey(valueEncrypted: string, rawKey: string): string {
  const key = loadKeyFromBase64(rawKey);
  const prefix = `${ENCRYPTION_VERSION}:`;
  if (!valueEncrypted.startsWith(prefix)) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Encrypted secret payload version is not supported.",
      retryable: false,
      statusCode: 500,
      details: {
        expected: ENCRYPTION_VERSION,
        actual: valueEncrypted.split(":").slice(0, 2).join(":")
      },
      causeType: "secret_payload_version"
    });
  }

  const parts = valueEncrypted.slice(prefix.length).split(":");
  if (parts.length !== 3) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Encrypted secret payload is invalid.",
      retryable: false,
      statusCode: 500,
      causeType: "secret_payload_parts"
    });
  }

  const [ivBase64, authTagBase64, cipherBase64] = parts;

  try {
    const iv = Buffer.from(ivBase64, "base64");
    const authTag = Buffer.from(authTagBase64, "base64");
    const cipherText = Buffer.from(cipherBase64, "base64");

    const decipher = createDecipheriv(ENCRYPTION_ALGO, key, iv);
    decipher.setAuthTag(authTag);
    const plain = Buffer.concat([decipher.update(cipherText), decipher.final()]);
    return plain.toString("utf8");
  } catch {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Encrypted secret payload is invalid.",
      retryable: false,
      statusCode: 500,
      causeType: "secret_payload_decrypt"
    });
  }
}
