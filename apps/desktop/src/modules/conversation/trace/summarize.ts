const CONTROL_CHARACTERS = /[\u0000-\u001f\u007f]/g;
const THINK_TAGS = /<\/?think>/gi;
const SENSITIVE_KEY_PATTERN = /(token|secret|password|api[_-]?key|authorization|cookie)/i;

const OPERATION_CANDIDATE_KEYS = ["command", "path", "filePath", "url", "q", "query"] as const;

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

export function extractReasoningSentence(delta: string, fallback: string, maxLength = 88): string {
  const cleaned = cleanText(delta);
  if (cleaned === "") {
    return fallback;
  }
  const sentence = firstSentence(cleaned);
  if (sentence === "") {
    return fallback;
  }
  return truncateText(sentence, maxLength);
}

export function extractOperationSummary(payload: Record<string, unknown>): string {
  const input = asRecord(payload.input);
  const source = input ?? payload;

  for (const key of OPERATION_CANDIDATE_KEYS) {
    const value = readableScalar(source[key]);
    if (value !== "") {
      return truncateText(`${key}: ${value}`, 120);
    }
  }

  const scalar = firstReadableScalar(source);
  if (scalar !== "") {
    return truncateText(scalar, 120);
  }

  // Fallback to top-level when input object has no readable scalar.
  if (input) {
    const topLevel = firstReadableScalar(payload);
    if (topLevel !== "") {
      return truncateText(topLevel, 120);
    }
  }
  return "";
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
