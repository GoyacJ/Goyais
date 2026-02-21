export type ProviderKey =
  | "deepseek"
  | "minimax_cn"
  | "minimax_intl"
  | "zhipu"
  | "qwen"
  | "doubao"
  | "openai"
  | "anthropic"
  | "google"
  | "custom";

export type ModelCatalogSource = "live" | "snapshot";

export interface ModelCatalogItem {
  model_id: string;
  display_name: string;
  provider: ProviderKey;
  released_at?: string;
  is_latest: boolean;
  source: ModelCatalogSource;
  capabilities: string[];
}

export interface ModelCatalogResponse {
  provider: ProviderKey;
  items: ModelCatalogItem[];
  fetched_at: string;
  fallback_used: boolean;
  warning?: string;
}

export interface ProviderMetadata {
  key: ProviderKey;
  label: string;
  defaultBaseUrl: string;
  authHeader: string;
  openaiCompatible: boolean;
  docsUrl: string;
  fallbackModels: ModelCatalogItem[];
}

function fallback(provider: ProviderKey, models: Array<{ id: string; name: string; latest?: boolean }>): ModelCatalogItem[] {
  return models.map((item, index) => ({
    model_id: item.id,
    display_name: item.name,
    provider,
    is_latest: item.latest ?? index === 0,
    source: "snapshot",
    capabilities: ["chat"]
  }));
}

export const PROVIDER_METADATA: Record<ProviderKey, ProviderMetadata> = {
  deepseek: {
    key: "deepseek",
    label: "DeepSeek",
    defaultBaseUrl: "https://api.deepseek.com",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "https://api-docs.deepseek.com/",
    fallbackModels: fallback("deepseek", [
      { id: "deepseek-chat", name: "DeepSeek Chat", latest: true },
      { id: "deepseek-reasoner", name: "DeepSeek Reasoner" }
    ])
  },
  minimax_cn: {
    key: "minimax_cn",
    label: "MiniMax (国内)",
    defaultBaseUrl: "https://api.minimaxi.com/v1",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "https://platform.minimaxi.com/document",
    fallbackModels: fallback("minimax_cn", [
      { id: "MiniMax-M1", name: "MiniMax M1", latest: true },
      { id: "abab6.5-chat", name: "abab6.5 Chat" }
    ])
  },
  minimax_intl: {
    key: "minimax_intl",
    label: "MiniMax (国际)",
    defaultBaseUrl: "https://api.minimax.io/v1",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "https://platform.minimax.io/docs",
    fallbackModels: fallback("minimax_intl", [
      { id: "MiniMax-M1", name: "MiniMax M1", latest: true },
      { id: "abab6.5-chat", name: "abab6.5 Chat" }
    ])
  },
  zhipu: {
    key: "zhipu",
    label: "BigModel (智谱)",
    defaultBaseUrl: "https://open.bigmodel.cn/api/paas/v4",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "https://docs.bigmodel.cn/cn/guide/models",
    fallbackModels: fallback("zhipu", [
      { id: "glm-4-plus", name: "GLM-4-Plus", latest: true },
      { id: "glm-4-air", name: "GLM-4-Air" }
    ])
  },
  qwen: {
    key: "qwen",
    label: "Qwen",
    defaultBaseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "https://www.alibabacloud.com/help/en/model-studio/",
    fallbackModels: fallback("qwen", [
      { id: "qwen-plus-latest", name: "Qwen Plus Latest", latest: true },
      { id: "qwen-turbo-latest", name: "Qwen Turbo Latest" }
    ])
  },
  doubao: {
    key: "doubao",
    label: "豆包",
    defaultBaseUrl: "https://ark.cn-beijing.volces.com/api/v3",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "https://www.volcengine.com/docs",
    fallbackModels: fallback("doubao", [
      { id: "doubao-1.5-pro-32k", name: "Doubao 1.5 Pro 32k", latest: true },
      { id: "doubao-1.5-lite-32k", name: "Doubao 1.5 Lite 32k" }
    ])
  },
  openai: {
    key: "openai",
    label: "OpenAI",
    defaultBaseUrl: "https://api.openai.com/v1",
    authHeader: "Authorization: Bearer",
    openaiCompatible: false,
    docsUrl: "https://platform.openai.com/docs/models",
    fallbackModels: fallback("openai", [
      { id: "gpt-5", name: "GPT-5", latest: true },
      { id: "gpt-5-mini", name: "GPT-5 Mini" },
      { id: "gpt-4.1", name: "GPT-4.1" }
    ])
  },
  anthropic: {
    key: "anthropic",
    label: "Anthropic",
    defaultBaseUrl: "https://api.anthropic.com/v1",
    authHeader: "x-api-key",
    openaiCompatible: false,
    docsUrl: "https://docs.anthropic.com/en/docs/about-claude/models/all-models",
    fallbackModels: fallback("anthropic", [
      { id: "claude-sonnet-4-5", name: "Claude Sonnet 4.5", latest: true },
      { id: "claude-opus-4-1", name: "Claude Opus 4.1" }
    ])
  },
  google: {
    key: "google",
    label: "Google",
    defaultBaseUrl: "https://generativelanguage.googleapis.com/v1beta",
    authHeader: "x-goog-api-key",
    openaiCompatible: false,
    docsUrl: "https://ai.google.dev/gemini-api/docs/models",
    fallbackModels: fallback("google", [
      { id: "gemini-2.5-pro", name: "Gemini 2.5 Pro", latest: true },
      { id: "gemini-2.5-flash", name: "Gemini 2.5 Flash" }
    ])
  },
  custom: {
    key: "custom",
    label: "Custom",
    defaultBaseUrl: "",
    authHeader: "Authorization: Bearer",
    openaiCompatible: true,
    docsUrl: "",
    fallbackModels: []
  }
};

export const PROVIDER_ORDER: ProviderKey[] = [
  "deepseek",
  "minimax_cn",
  "minimax_intl",
  "zhipu",
  "qwen",
  "doubao",
  "openai",
  "anthropic",
  "google",
  "custom"
];

export function isProviderKey(value: string): value is ProviderKey {
  return value in PROVIDER_METADATA;
}

export function providerLabel(provider: ProviderKey): string {
  return PROVIDER_METADATA[provider].label;
}
