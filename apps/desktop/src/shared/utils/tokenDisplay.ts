export function formatTokenCompact(value: unknown): string {
  const normalized = toNonNegativeTokenNumber(value);
  if (normalized < 1000) {
    return String(Math.trunc(normalized));
  }
  if (normalized < 1_000_000) {
    return withUnit(normalized / 1000, "K");
  }
  return withUnit(normalized / 1_000_000, "M");
}

export function formatTokenUsageWithThreshold(total: unknown, threshold: unknown): string {
  const totalText = formatTokenCompact(total);
  const thresholdNumber = toPositiveTokenNumber(threshold);
  const thresholdText = thresholdNumber === null ? "∞" : formatTokenCompact(thresholdNumber);
  return `${totalText} / ${thresholdText}`;
}

function toNonNegativeTokenNumber(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return 0;
  }
  return value;
}

function toPositiveTokenNumber(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return null;
  }
  return value;
}

function withUnit(value: number, unit: "K" | "M"): string {
  const rounded = Math.round(value * 10) / 10;
  const normalized = Number.isInteger(rounded) ? String(Math.trunc(rounded)) : rounded.toFixed(1);
  return `${normalized}${unit}`;
}
