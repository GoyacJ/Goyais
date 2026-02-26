import type { ComposerSuggestion } from "@/shared/types/api";

const BUILTIN_COMMAND_DETAIL_KEYS: Record<string, string> = {
  help: "conversation.composer.suggestion.command.help",
  agents: "conversation.composer.suggestion.command.agents",
  "algorithmic-art": "conversation.composer.suggestion.command.algorithmicArt",
  antfu: "conversation.composer.suggestion.command.antfu",
  bug: "conversation.composer.suggestion.command.bug",
  "brand-guidelines": "conversation.composer.suggestion.command.brandGuidelines",
  "build-things": "conversation.composer.suggestion.command.buildThings",
  "canvas-design": "conversation.composer.suggestion.command.canvasDesign",
  clear: "conversation.composer.suggestion.command.clear",
  compact: "conversation.composer.suggestion.command.compact",
  "compact-threshold": "conversation.composer.suggestion.command.compactThreshold",
  config: "conversation.composer.suggestion.command.config",
  cost: "conversation.composer.suggestion.command.cost",
  "ctx-viz": "conversation.composer.suggestion.command.ctxViz",
  doctor: "conversation.composer.suggestion.command.doctor",
  init: "conversation.composer.suggestion.command.init",
  listen: "conversation.composer.suggestion.command.listen",
  login: "conversation.composer.suggestion.command.login",
  logout: "conversation.composer.suggestion.command.logout",
  mcp: "conversation.composer.suggestion.command.mcp",
  "messages-debug": "conversation.composer.suggestion.command.messagesDebug",
  model: "conversation.composer.suggestion.command.model",
  modelstatus: "conversation.composer.suggestion.command.modelstatus",
  ms: "conversation.composer.suggestion.command.modelstatus",
  "model-status": "conversation.composer.suggestion.command.modelstatus",
  onboarding: "conversation.composer.suggestion.command.onboarding",
  "output-style": "conversation.composer.suggestion.command.outputStyle",
  plugin: "conversation.composer.suggestion.command.plugin",
  "pr-comments": "conversation.composer.suggestion.command.prComments",
  "refresh-commands": "conversation.composer.suggestion.command.refreshCommands",
  "release-notes": "conversation.composer.suggestion.command.releaseNotes",
  rename: "conversation.composer.suggestion.command.rename",
  resume: "conversation.composer.suggestion.command.resume",
  review: "conversation.composer.suggestion.command.review",
  statusline: "conversation.composer.suggestion.command.statusline",
  tag: "conversation.composer.suggestion.command.tag",
  todos: "conversation.composer.suggestion.command.todos",
  todo: "conversation.composer.suggestion.command.todos"
};

const SKILL_COMMAND_DETAIL_KEYS: Record<string, string> = {
  doc: "conversation.composer.suggestion.command.skill.doc",
  "doc-coauthoring": "conversation.composer.suggestion.command.skill.doc-coauthoring",
  docx: "conversation.composer.suggestion.command.skill.docx",
  figma: "conversation.composer.suggestion.command.skill.figma",
  "frontend-design": "conversation.composer.suggestion.command.skill.frontend-design",
  "gh-address-comments": "conversation.composer.suggestion.command.skill.gh-address-comments",
  "gh-fix-ci": "conversation.composer.suggestion.command.skill.gh-fix-ci",
  "internal-comms": "conversation.composer.suggestion.command.skill.internal-comms",
  "mcp-builder": "conversation.composer.suggestion.command.skill.mcp-builder",
  nuxt: "conversation.composer.suggestion.command.skill.nuxt",
  pdf: "conversation.composer.suggestion.command.skill.pdf",
  pinia: "conversation.composer.suggestion.command.skill.pinia",
  playwright: "conversation.composer.suggestion.command.skill.playwright",
  pnpm: "conversation.composer.suggestion.command.skill.pnpm",
  pptx: "conversation.composer.suggestion.command.skill.pptx",
  "skill-creator": "conversation.composer.suggestion.command.skill.skill-creator",
  "slack-gif-creator": "conversation.composer.suggestion.command.skill.slack-gif-creator",
  slidev: "conversation.composer.suggestion.command.skill.slidev",
  "theme-factory": "conversation.composer.suggestion.command.skill.theme-factory",
  tsdown: "conversation.composer.suggestion.command.skill.tsdown",
  turborepo: "conversation.composer.suggestion.command.skill.turborepo",
  unocss: "conversation.composer.suggestion.command.skill.unocss",
  vite: "conversation.composer.suggestion.command.skill.vite",
  vitepress: "conversation.composer.suggestion.command.skill.vitepress",
  vitest: "conversation.composer.suggestion.command.skill.vitest",
  vue: "conversation.composer.suggestion.command.skill.vue",
  "vue-best-practices": "conversation.composer.suggestion.command.skill.vue-best-practices",
  "vue-router-best-practices": "conversation.composer.suggestion.command.skill.vue-router-best-practices",
  "vue-testing-best-practices": "conversation.composer.suggestion.command.skill.vue-testing-best-practices",
  "vueuse-functions": "conversation.composer.suggestion.command.skill.vueuse-functions",
  "web-artifacts-builder": "conversation.composer.suggestion.command.skill.web-artifacts-builder",
  "web-design-guidelines": "conversation.composer.suggestion.command.skill.web-design-guidelines",
  "webapp-testing": "conversation.composer.suggestion.command.skill.webapp-testing",
  xlsx: "conversation.composer.suggestion.command.skill.xlsx"
};

export function localizeComposerSuggestionDetails(
  suggestions: ComposerSuggestion[],
  translate: (key: string) => string
): ComposerSuggestion[] {
  return suggestions.map((suggestion) => localizeComposerSuggestionDetail(suggestion, translate));
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

function resolveResourceTypeDetailKey(type: string): string {
  switch (type) {
    case "model":
      return "conversation.composer.suggestion.type.model";
    case "rule":
      return "conversation.composer.suggestion.type.rule";
    case "skill":
      return "conversation.composer.suggestion.type.skill";
    case "mcp":
      return "conversation.composer.suggestion.type.mcp";
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
