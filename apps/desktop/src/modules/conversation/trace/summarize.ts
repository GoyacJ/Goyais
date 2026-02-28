import type { OperationIntentKind } from "@/modules/conversation/trace/types";

const CONTROL_CHARACTERS = /[\u0000-\u001f\u007f]/g;
const THINK_TAGS = /<\/?think>/gi;
const SENSITIVE_KEY_PATTERN = /(token|secret|password|api[_-]?key|authorization|cookie)/i;
const REASONING_PLACEHOLDER_PATTERN = /^(model[_\s-]?call|assistant[_\s-]?output|thinking|processing|other)$/i;

const OPERATION_INTENT_CANDIDATES: ReadonlyArray<{ key: string; kind: OperationIntentKind }> = [
  { key: "command", kind: "command" },
  { key: "path", kind: "path" },
  { key: "filePath", kind: "path" },
  { key: "url", kind: "url" },
  { key: "q", kind: "query" },
  { key: "query", kind: "query" }
];

const IGNORED_SCALAR_KEYS: ReadonlySet<string> = new Set([
  "name",
  "call_id",
  "risk_level",
  "source",
  "ok",
  "stage",
  "turn",
  "run_state",
  "action",
  "usage"
]);

export function asString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

export function asRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

export function truncateText(value: string, maxLength: number): string {
  const normalized = value.trim();
  if (normalized === "") {
    return "";
  }
  const chars = Array.from(normalized);
  if (chars.length <= maxLength) {
    return normalized;
  }
  return `${chars.slice(0, Math.max(0, maxLength - 1)).join("")}…`;
}

export function extractReasoningSentence(delta: string, maxLength = 88): string {
  const cleaned = cleanText(delta);
  if (cleaned === "") {
    return "";
  }
  const sentence = firstSentence(cleaned);
  if (sentence === "") {
    return "";
  }
  const truncated = truncateText(sentence, maxLength);
  if (isReasoningPlaceholder(truncated)) {
    return "";
  }
  return truncated;
}

export type OperationIntent = {
  kind: OperationIntentKind;
  value: string;
};

export function extractOperationIntent(payload: Record<string, unknown>): OperationIntent {
  const input = asRecord(payload.input);
  const source = input ?? payload;

  const directIntent = resolveOperationIntentFromRecord(source);
  if (directIntent.kind !== "none") {
    return directIntent;
  }

  if (input) {
    const topLevelIntent = resolveOperationIntentFromRecord(payload);
    if (topLevelIntent.kind !== "none") {
      return topLevelIntent;
    }
  }
  return { kind: "none", value: "" };
}

export function extractOperationSummary(payload: Record<string, unknown>): string {
  const intent = extractOperationIntent(payload);
  if (intent.kind === "none") {
    return "";
  }
  if (intent.kind === "scalar") {
    return intent.value;
  }
  const label = intent.kind === "path"
    ? "path"
    : intent.kind === "query"
      ? "query"
      : intent.kind;
  return truncateText(`${label}: ${intent.value}`, 120);
}

export function extractResultSummary(payload: Record<string, unknown>, isSuccess: boolean | null): string {
  if (isSuccess === false) {
    const err = cleanText(asString(payload.error));
    if (err !== "") {
      return truncateText(firstSentence(err) || err, 120);
    }
  }

  const output = payload.output;
  const outputSummary = readableValue(output);
  if (outputSummary !== "") {
    return truncateText(firstSentence(outputSummary) || outputSummary, 120);
  }

  if (isSuccess === false) {
    const fallback = firstReadableScalar(payload);
    return truncateText(firstSentence(fallback) || fallback, 120);
  }

  return "";
}

export function redactSensitivePayload(payload: Record<string, unknown>): Record<string, unknown> {
  return redactUnknown(payload) as Record<string, unknown>;
}

export function toCompactJSON(value: unknown, maxLength = 1500): string {
  try {
    return truncateText(JSON.stringify(value, null, 2), maxLength);
  } catch {
    return "";
  }
}

function redactUnknown(value: unknown, keyHint = ""): unknown {
  if (SENSITIVE_KEY_PATTERN.test(keyHint)) {
    return "***";
  }
  if (Array.isArray(value)) {
    return value.map((item) => redactUnknown(item));
  }
  const record = asRecord(value);
  if (!record) {
    return value;
  }
  const cloned: Record<string, unknown> = {};
  for (const [key, item] of Object.entries(record)) {
    cloned[key] = redactUnknown(item, key);
  }
  return cloned;
}

function readableValue(value: unknown): string {
  const scalar = readableScalar(value);
  if (scalar !== "") {
    return scalar;
  }
  const record = asRecord(value);
  if (record) {
    return toCompactJSON(record, 220);
  }
  if (Array.isArray(value)) {
    return toCompactJSON(value, 220);
  }
  return "";
}

function readableScalar(value: unknown): string {
  if (typeof value === "string") {
    return cleanText(value);
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

function firstReadableScalar(record: Record<string, unknown>): string {
  for (const [key, value] of Object.entries(record)) {
    if (IGNORED_SCALAR_KEYS.has(key)) {
      continue;
    }
    const scalar = readableScalar(value);
    if (scalar !== "") {
      return `${key}: ${scalar}`;
    }
  }
  return "";
}

function resolveOperationIntentFromRecord(record: Record<string, unknown>): OperationIntent {
  for (const candidate of OPERATION_INTENT_CANDIDATES) {
    const value = readableScalar(record[candidate.key]);
    if (value !== "") {
      return {
        kind: candidate.kind,
        value: truncateText(value, 120)
      };
    }
  }

  const scalar = firstReadableScalar(record);
  if (scalar !== "") {
    return {
      kind: "scalar",
      value: truncateText(scalar, 120)
    };
  }

  return { kind: "none", value: "" };
}

function cleanText(value: string): string {
  return value
    .replace(THINK_TAGS, " ")
    .replace(CONTROL_CHARACTERS, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function firstSentence(value: string): string {
  const normalized = value.trim();
  if (normalized === "") {
    return "";
  }
  const sentenceEndings = ["。", "！", "？", ".", "!", "?", "；", ";", "\n"];
  const chars = Array.from(normalized);
  const endIndex = chars.findIndex((char) => sentenceEndings.includes(char));
  if (endIndex < 0) {
    return normalized;
  }
  return chars.slice(0, endIndex + 1).join("").trim();
}

function isReasoningPlaceholder(value: string): boolean {
  const normalized = value.trim().replace(/[。.!！?？;；:：]+$/g, "");
  if (normalized === "") {
    return true;
  }
  return REASONING_PLACEHOLDER_PATTERN.test(normalized);
}
