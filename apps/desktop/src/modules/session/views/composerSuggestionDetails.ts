import type { ComposerSuggestion } from "@/shared/types/api";

const BUILTIN_COMMAND_DETAIL_KEYS: Record<string, string> = {
  help: "session.composer.suggestion.command.help",
  agents: "session.composer.suggestion.command.agents",
  "algorithmic-art": "session.composer.suggestion.command.algorithmicArt",
  antfu: "session.composer.suggestion.command.antfu",
  bug: "session.composer.suggestion.command.bug",
  "brand-guidelines": "session.composer.suggestion.command.brandGuidelines",
  "build-things": "session.composer.suggestion.command.buildThings",
  "canvas-design": "session.composer.suggestion.command.canvasDesign",
  clear: "session.composer.suggestion.command.clear",
  compact: "session.composer.suggestion.command.compact",
  "compact-threshold": "session.composer.suggestion.command.compactThreshold",
  config: "session.composer.suggestion.command.config",
  cost: "session.composer.suggestion.command.cost",
  "ctx-viz": "session.composer.suggestion.command.ctxViz",
  doctor: "session.composer.suggestion.command.doctor",
  init: "session.composer.suggestion.command.init",
  listen: "session.composer.suggestion.command.listen",
  login: "session.composer.suggestion.command.login",
  logout: "session.composer.suggestion.command.logout",
  mcp: "session.composer.suggestion.command.mcp",
  "messages-debug": "session.composer.suggestion.command.messagesDebug",
  model: "session.composer.suggestion.command.model",
  modelstatus: "session.composer.suggestion.command.modelstatus",
  ms: "session.composer.suggestion.command.modelstatus",
  "model-status": "session.composer.suggestion.command.modelstatus",
  onboarding: "session.composer.suggestion.command.onboarding",
  "output-style": "session.composer.suggestion.command.outputStyle",
  plugin: "session.composer.suggestion.command.plugin",
  "pr-comments": "session.composer.suggestion.command.prComments",
  "refresh-commands": "session.composer.suggestion.command.refreshCommands",
  "release-notes": "session.composer.suggestion.command.releaseNotes",
  rename: "session.composer.suggestion.command.rename",
  resume: "session.composer.suggestion.command.resume",
  review: "session.composer.suggestion.command.review",
  statusline: "session.composer.suggestion.command.statusline",
  tag: "session.composer.suggestion.command.tag",
  todos: "session.composer.suggestion.command.todos",
  todo: "session.composer.suggestion.command.todos"
};

const SKILL_COMMAND_DETAIL_KEYS: Record<string, string> = {
  doc: "session.composer.suggestion.command.skill.doc",
  "doc-coauthoring": "session.composer.suggestion.command.skill.doc-coauthoring",
  docx: "session.composer.suggestion.command.skill.docx",
  figma: "session.composer.suggestion.command.skill.figma",
  "frontend-design": "session.composer.suggestion.command.skill.frontend-design",
  "gh-address-comments": "session.composer.suggestion.command.skill.gh-address-comments",
  "gh-fix-ci": "session.composer.suggestion.command.skill.gh-fix-ci",
  "internal-comms": "session.composer.suggestion.command.skill.internal-comms",
  "mcp-builder": "session.composer.suggestion.command.skill.mcp-builder",
  nuxt: "session.composer.suggestion.command.skill.nuxt",
  pdf: "session.composer.suggestion.command.skill.pdf",
  pinia: "session.composer.suggestion.command.skill.pinia",
  playwright: "session.composer.suggestion.command.skill.playwright",
  pnpm: "session.composer.suggestion.command.skill.pnpm",
  pptx: "session.composer.suggestion.command.skill.pptx",
  "skill-creator": "session.composer.suggestion.command.skill.skill-creator",
  "slack-gif-creator": "session.composer.suggestion.command.skill.slack-gif-creator",
  slidev: "session.composer.suggestion.command.skill.slidev",
  "theme-factory": "session.composer.suggestion.command.skill.theme-factory",
  tsdown: "session.composer.suggestion.command.skill.tsdown",
  turborepo: "session.composer.suggestion.command.skill.turborepo",
  unocss: "session.composer.suggestion.command.skill.unocss",
  vite: "session.composer.suggestion.command.skill.vite",
  vitepress: "session.composer.suggestion.command.skill.vitepress",
  vitest: "session.composer.suggestion.command.skill.vitest",
  vue: "session.composer.suggestion.command.skill.vue",
  "vue-best-practices": "session.composer.suggestion.command.skill.vue-best-practices",
  "vue-router-best-practices": "session.composer.suggestion.command.skill.vue-router-best-practices",
  "vue-testing-best-practices": "session.composer.suggestion.command.skill.vue-testing-best-practices",
  "vueuse-functions": "session.composer.suggestion.command.skill.vueuse-functions",
  "web-artifacts-builder": "session.composer.suggestion.command.skill.web-artifacts-builder",
  "web-design-guidelines": "session.composer.suggestion.command.skill.web-design-guidelines",
  "webapp-testing": "session.composer.suggestion.command.skill.webapp-testing",
  xlsx: "session.composer.suggestion.command.skill.xlsx"
};

export function localizeComposerSuggestionDetails(
  suggestions: ComposerSuggestion[],
  translate: (key: string) => string
): ComposerSuggestion[] {
  return suggestions.map((suggestion) => presentComposerSuggestion(localizeComposerSuggestionDetail(suggestion, translate)));
}

function localizeComposerSuggestionDetail(
  suggestion: ComposerSuggestion,
  translate: (key: string) => string
): ComposerSuggestion {
  if (suggestion.kind === "command") {
    const commandName = extractCommandName(suggestion);
    const fallback = normalizeText(suggestion.detail);
    const key = BUILTIN_COMMAND_DETAIL_KEYS[commandName] ?? SKILL_COMMAND_DETAIL_KEYS[commandName];
    if (!key) {
      return suggestion;
    }
    const localized = normalizeText(translate(key));
    if (localized === "" || localized === key || localized === fallback) {
      return suggestion;
    }
    return {
      ...suggestion,
      detail: localized
    };
  }

  if (suggestion.kind === "resource_type") {
    const type = extractResourceType(suggestion);
    const key = resolveResourceTypeDetailKey(type);
    if (!key) {
      return suggestion;
    }
    const localized = normalizeText(translate(key));
    const current = normalizeText(suggestion.detail);
    if (localized === "" || localized === key || localized === current) {
      return suggestion;
    }
    return {
      ...suggestion,
      detail: localized
    };
  }

  if (suggestion.kind === "resource" && extractResourceType(suggestion) === "file") {
    if (normalizeText(suggestion.detail) === "") {
      return suggestion;
    }
    return {
      ...suggestion,
      detail: ""
    };
  }

  return suggestion;
}

function presentComposerSuggestion(suggestion: ComposerSuggestion): ComposerSuggestion {
  if (suggestion.kind !== "resource") {
    return suggestion;
  }
  const resourceType = extractResourceType(suggestion);
  if (resourceType !== "model") {
    return suggestion;
  }
  const displayName = normalizeText(suggestion.detail);
  if (displayName === "") {
    return suggestion;
  }
  const canonicalLabel = normalizeText(suggestion.label);
  if (canonicalLabel === "") {
    return suggestion;
  }
  return {
    ...suggestion,
    label: `@model:${displayName}`,
    detail: canonicalLabel
  };
}

function resolveResourceTypeDetailKey(type: string): string {
  switch (type) {
    case "model":
      return "session.composer.suggestion.type.model";
    case "rule":
      return "session.composer.suggestion.type.rule";
    case "skill":
      return "session.composer.suggestion.type.skill";
    case "mcp":
      return "session.composer.suggestion.type.mcp";
    default:
      return "";
  }
}

function extractCommandName(suggestion: Pick<ComposerSuggestion, "label" | "insert_text">): string {
  return extractSlashName(suggestion.insert_text) || extractSlashName(suggestion.label);
}

function extractSlashName(value: string): string {
  const token = normalizeText(value);
  if (!token.startsWith("/")) {
    return "";
  }
  const command = token.slice(1).split(/\s+/, 1)[0] ?? "";
  return normalizeText(command).toLowerCase();
}

function extractResourceType(suggestion: Pick<ComposerSuggestion, "label" | "insert_text">): string {
  return extractMentionType(suggestion.insert_text) || extractMentionType(suggestion.label);
}

function extractMentionType(value: string): string {
  const token = normalizeText(value);
  if (!token.startsWith("@")) {
    return "";
  }
  const separator = token.indexOf(":");
  if (separator <= 1) {
    return "";
  }
  return token.slice(1, separator).trim().toLowerCase();
}

function normalizeText(value: string | undefined): string {
  return (value ?? "").trim();
}
