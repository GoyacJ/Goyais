import { compile } from "json-schema-to-typescript";
import fs from "node:fs/promises";
import path from "node:path";

const root = path.resolve(process.cwd());
const schemaDir = path.join(root, "schemas", "v1");
const outFile = path.join(root, "generated", "ts", "protocol.d.ts");

const schemaFiles = [
  "run-create.request.schema.json",
  "run-create.response.schema.json",
  "tool-confirmation.request.schema.json",
  "tool-confirmation.response.schema.json"
];

const blocks = [];
for (const file of schemaFiles) {
  const json = JSON.parse(await fs.readFile(path.join(schemaDir, file), "utf8"));
  const typeName = file.replace(/\..+$/, "").replace(/[-.]/g, "_");
  const out = await compile(json, typeName, { additionalProperties: false });
  blocks.push(out);
}

await fs.mkdir(path.dirname(outFile), { recursive: true });
const envelopeBlock = `export type EventType = "plan" | "tool_call" | "tool_result" | "patch" | "error" | "done";

export interface EventEnvelope {
  protocol_version: "1.0.0";
  event_id: string;
  run_id: string;
  seq: number;
  ts: string;
  type: EventType;
  payload: Record<string, unknown>;
}
`;

await fs.writeFile(outFile, `${blocks.join("\n\n")}\n\n${envelopeBlock}`, "utf8");
console.log(`generated ${outFile}`);
