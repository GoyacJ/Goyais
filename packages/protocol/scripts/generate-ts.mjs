import { compile } from "json-schema-to-typescript";
import fs from "node:fs/promises";
import path from "node:path";

const root = path.resolve(process.cwd());
const schemaDir = path.join(root, "schemas", "v2");
const outFile = path.join(root, "generated", "ts", "protocol.d.ts");

const schemaFiles = [
  "goyais-error.schema.json",
  "execution-create.request.schema.json",
  "execution-create.response.schema.json",
  "confirmation-decision.request.schema.json",
  "confirmation-decision.response.schema.json"
];

const blocks = [];
for (const file of schemaFiles) {
  const json = JSON.parse(await fs.readFile(path.join(schemaDir, file), "utf8"));
  const typeName = file.replace(/\..+$/, "").replace(/[-.]/g, "_");
  const out = await compile(json, typeName, { additionalProperties: false });
  blocks.push(out);
}

await fs.mkdir(path.dirname(outFile), { recursive: true });
const envelopeBlock = `export type EventType =
  | "plan"
  | "tool_call"
  | "tool_result"
  | "patch"
  | "error"
  | "done"
  | "text_delta"
  | "heartbeat"
  | "confirmation_request"
  | "confirmation_decision"
  | "cancelled";

export interface EventEnvelope {
  protocol_version: "2.0.0";
  trace_id: string;
  event_id: string;
  execution_id: string;
  seq: number;
  ts: string;
  type: EventType;
  payload: Record<string, unknown>;
}
`;

await fs.writeFile(outFile, `${blocks.join("\n\n")}\n\n${envelopeBlock}`, "utf8");
console.log(`generated ${outFile}`);
