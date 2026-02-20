import fs from "node:fs";
import path from "node:path";

import { HubServerError } from "./errors";

function parseVersionDirectoryName(name: string): number | null {
  const match = /^v(\d+)$/.exec(name);
  if (!match) {
    return null;
  }
  return Number.parseInt(match[1], 10);
}

function candidateSchemaDirectories(): string[] {
  const fromCwd = path.resolve(process.cwd(), "../../packages/protocol/schemas");
  const fromModuleDir = path.resolve(__dirname, "../../../../packages/protocol/schemas");
  return [fromCwd, fromModuleDir];
}

function resolveProtocolVersionFile(): string | null {
  for (const schemasDir of candidateSchemaDirectories()) {
    if (!fs.existsSync(schemasDir)) {
      continue;
    }

    const versions = fs
      .readdirSync(schemasDir, { withFileTypes: true })
      .filter((entry) => entry.isDirectory())
      .map((entry) => ({
        name: entry.name,
        order: parseVersionDirectoryName(entry.name)
      }))
      .filter((entry): entry is { name: string; order: number } => typeof entry.order === "number")
      .sort((a, b) => b.order - a.order);

    for (const version of versions) {
      const protocolVersionPath = path.join(schemasDir, version.name, "protocol-version.json");
      if (fs.existsSync(protocolVersionPath)) {
        return protocolVersionPath;
      }
    }
  }

  return null;
}

export function loadProtocolVersionFromSchema(): string {
  const protocolVersionPath = resolveProtocolVersionFile();
  if (!protocolVersionPath) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Protocol version schema file was not found.",
      retryable: false,
      statusCode: 500,
      causeType: "protocol_version_missing"
    });
  }

  try {
    const parsed = JSON.parse(fs.readFileSync(protocolVersionPath, "utf8")) as {
      version?: unknown;
    };
    if (typeof parsed.version !== "string" || parsed.version.trim().length === 0) {
      throw new Error("version field missing");
    }
    return parsed.version.trim();
  } catch (error) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Protocol version schema file is invalid.",
      retryable: false,
      statusCode: 500,
      details: {
        file: protocolVersionPath
      },
      causeType: error instanceof Error ? error.name : "protocol_version_parse"
    });
  }
}
