import type { CapabilityRisk, RiskDetails } from "@/types/ui";

function toRecord(input: unknown): Record<string, unknown> {
  return input && typeof input === "object" ? (input as Record<string, unknown>) : {};
}

function readString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function readStringArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === "string");
  }
  if (typeof value === "string") {
    return [value];
  }
  return [];
}

function pathOutsideWorkspace(path: string): boolean {
  return path.startsWith("/") || path.startsWith("..") || path.includes("../") || path.includes("..\\");
}

function detectRisk(toolName: string, details: RiskDetails): CapabilityRisk {
  if (toolName.includes("run_command")) return "exec";
  if (toolName.includes("write") || toolName.includes("patch")) return "write";
  if (toolName.includes("delete") || toolName.includes("remove")) return "delete";
  if (toolName.includes("network") || toolName.includes("http") || details.domains.length > 0) return "network";

  const cmd = details.command ?? "";
  if (/curl|wget|scp|rsync/i.test(cmd)) return details.domains.length > 0 ? "exfil" : "network";

  return "none";
}

export function classifyToolRisk(toolName: string, args: unknown): {
  primary: CapabilityRisk;
  hasRisk: boolean;
  details: RiskDetails;
} {
  const source = toRecord(args);
  const paths = [
    ...readStringArray(source.path),
    ...readStringArray(source.paths),
    ...readStringArray(source.file),
    ...readStringArray(source.files)
  ];
  const domains = [
    ...readStringArray(source.domain),
    ...readStringArray(source.domains),
    ...readStringArray(source.url),
    ...readStringArray(source.urls)
  ];

  const details: RiskDetails = {
    command: readString(source.cmd),
    cwd: readString(source.cwd),
    paths,
    domains,
    pathOutsideWorkspace: paths.some(pathOutsideWorkspace)
  };

  const primary = detectRisk(toolName, details);

  return {
    primary,
    hasRisk: primary !== "none",
    details
  };
}
