const DEFAULT_NAME_PATTERN = /^(?:新对话(?: \d+)?|Conversation(?: \d+)?)$/i;

export function isDefaultConversationName(name: string): boolean {
  const normalized = normalizeWhitespace(name);
  return DEFAULT_NAME_PATTERN.test(normalized);
}

export function buildNameFromFirstMessage(content: string, maxChars = 10): string {
  const normalized = normalizeWhitespace(content);
  if (normalized === "" || maxChars <= 0) {
    return "";
  }
  return Array.from(normalized).slice(0, maxChars).join("");
}

function normalizeWhitespace(content: string): string {
  return content.trim().replace(/\s+/g, " ");
}
