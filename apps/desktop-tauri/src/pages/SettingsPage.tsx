import { Loader2, Plus, RefreshCcw, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";

import {
  type DataModelConfig,
  getModelConfigsClient,
  type UpdateModelConfigInput
} from "@/api/dataSource";
import { SyncNowButton } from "@/components/SyncNowButton";
import { SettingRow } from "@/components/settings/SettingRow";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import type { SupportedLocale } from "@/i18n/types";
import { cn } from "@/lib/cn";
import {
  normalizeRuntimeUrl,
  type ThemeMode,
  useSettingsStore
} from "@/stores/settingsStore";
import {
  selectCurrentPermissions,
  selectCurrentProfile,
  selectCurrentWorkspaceKind,
  useWorkspaceStore
} from "@/stores/workspaceStore";
import {
  PROVIDER_METADATA,
  PROVIDER_ORDER,
  providerLabel,
  type ModelCatalogResponse,
  type ProviderKey
} from "@/types/modelCatalog";

type SettingsSection = "general" | "runtime" | "models";
type RowSaveState = "idle" | "saving" | "error";

type ModelDraft = {
  provider: ProviderKey;
  model: string;
  base_url: string;
  temperature: string;
  max_tokens: string;
  secret_ref: string;
  api_key: string;
};

function parseSection(section: string | undefined): SettingsSection {
  if (section === "runtime" || section === "models") {
    return section;
  }
  return "general";
}

function parseOptionalInt(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const parsed = Number(trimmed);
  if (!Number.isInteger(parsed)) {
    return null;
  }
  return parsed;
}

function buildFallbackCatalog(provider: ProviderKey, warning?: string): ModelCatalogResponse {
  return {
    provider,
    items: [...PROVIDER_METADATA[provider].fallbackModels],
    fetched_at: new Date().toISOString(),
    fallback_used: true,
    warning
  };
}

function modelDraftFromConfig(item: DataModelConfig): ModelDraft {
  return {
    provider: item.provider,
    model: item.model,
    base_url: item.base_url ?? PROVIDER_METADATA[item.provider].defaultBaseUrl,
    temperature: String(item.temperature ?? 0),
    max_tokens: item.max_tokens === null ? "" : String(item.max_tokens),
    secret_ref: item.secret_ref,
    api_key: ""
  };
}

function canManageModelConfigs(workspaceKind: "local" | "remote", permissions: string[]): boolean {
  if (workspaceKind === "local") {
    return true;
  }
  return permissions.includes("modelconfig:manage");
}

function validateModelDraft(
  draft: ModelDraft,
  options: { requireSecretRef: boolean }
): { payload: UpdateModelConfigInput; error?: string } {
  const model = draft.model.trim();
  if (!model) {
    return { payload: {}, error: "Model is required" };
  }

  const normalizedBaseUrl = draft.base_url.trim();
  if (draft.provider === "custom" && !normalizedBaseUrl) {
    return { payload: {}, error: "Custom provider requires Base URL" };
  }

  if (normalizedBaseUrl) {
    try {
      normalizeRuntimeUrl(normalizedBaseUrl);
    } catch {
      return { payload: {}, error: "Base URL must be a valid http/https URL" };
    }
  }

  const temperatureValue = Number(draft.temperature.trim());
  if (!Number.isFinite(temperatureValue) || temperatureValue < 0 || temperatureValue > 2) {
    return { payload: {}, error: "Temperature must be between 0 and 2" };
  }

  const maxTokens = parseOptionalInt(draft.max_tokens);
  if (draft.max_tokens.trim() && (maxTokens === null || maxTokens <= 0)) {
    return { payload: {}, error: "Max tokens must be a positive integer" };
  }

  const secretRef = draft.secret_ref.trim();
  if (options.requireSecretRef && !secretRef) {
    return { payload: {}, error: "Secret ref is required" };
  }

  const payload: UpdateModelConfigInput = {
    provider: draft.provider,
    model,
    base_url: normalizedBaseUrl || null,
    temperature: temperatureValue,
    max_tokens: maxTokens
  };

  if (options.requireSecretRef) {
    payload.secret_ref = secretRef;
  }

  const apiKey = draft.api_key.trim();
  if (apiKey) {
    payload.api_key = apiKey;
  }

  return { payload };
}

export function SettingsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { addToast } = useToast();
  const params = useParams<{ section?: string }>();
  const activeSection = parseSection(params.section);

  const locale = useSettingsStore((state) => state.locale);
  const theme = useSettingsStore((state) => state.theme);
  const runtimeUrl = useSettingsStore((state) => state.runtimeUrl);
  const defaultModelConfigId = useSettingsStore((state) => state.defaultModelConfigId);
  const setLocale = useSettingsStore((state) => state.setLocale);
  const setTheme = useSettingsStore((state) => state.setTheme);
  const setRuntimeUrl = useSettingsStore((state) => state.setRuntimeUrl);
  const setDefaultModelConfigId = useSettingsStore((state) => state.setDefaultModelConfigId);

  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const modelConfigsClient = useMemo(() => getModelConfigsClient(currentProfile), [currentProfile]);
  const writable = canManageModelConfigs(workspaceKind, permissions);

  const [runtimeDraft, setRuntimeDraft] = useState(runtimeUrl);
  const [runtimeState, setRuntimeState] = useState<RowSaveState>("idle");
  const [runtimeHint, setRuntimeHint] = useState<string | undefined>(undefined);

  const [loadingModelConfigs, setLoadingModelConfigs] = useState(false);
  const [modelConfigs, setModelConfigs] = useState<DataModelConfig[]>([]);
  const [catalogByConfigId, setCatalogByConfigId] = useState<Record<string, ModelCatalogResponse>>({});
  const [draftsById, setDraftsById] = useState<Record<string, ModelDraft>>({});
  const [rowStateById, setRowStateById] = useState<Record<string, RowSaveState>>({});
  const [rowHintById, setRowHintById] = useState<Record<string, string | undefined>>({});

  const saveTimersRef = useRef<Record<string, number>>({});
  const draftsRef = useRef<Record<string, ModelDraft>>({});
  const modelConfigsRef = useRef<DataModelConfig[]>([]);
  const rowStateRef = useRef<Record<string, RowSaveState>>({});
  const catalogCacheRef = useRef<Record<string, { expiresAt: number; payload: ModelCatalogResponse }>>({});

  useEffect(() => {
    if (params.section && params.section !== "general" && params.section !== "runtime" && params.section !== "models") {
      navigate("/settings/general", { replace: true });
    }
  }, [navigate, params.section]);

  useEffect(() => {
    setRuntimeDraft(runtimeUrl);
  }, [runtimeUrl]);

  useEffect(() => {
    draftsRef.current = draftsById;
  }, [draftsById]);

  useEffect(() => {
    modelConfigsRef.current = modelConfigs;
  }, [modelConfigs]);

  useEffect(() => {
    rowStateRef.current = rowStateById;
  }, [rowStateById]);

  const refreshModelCatalog = useCallback(
    async (
      modelConfigId: string,
      provider: ProviderKey,
      baseUrl?: string | null,
      options?: { force?: boolean }
    ) => {
      const cacheKey = `${provider}|${baseUrl ?? ""}|${modelConfigId}`;
      const now = Date.now();
      const cached = catalogCacheRef.current[cacheKey];
      if (!options?.force && cached && cached.expiresAt > now) {
        setCatalogByConfigId((current) => ({
          ...current,
          [modelConfigId]: cached.payload
        }));
        return;
      }

      try {
        const payload = await modelConfigsClient.listModels(modelConfigId);
        catalogCacheRef.current[cacheKey] = {
          expiresAt: now + 30 * 60 * 1000,
          payload
        };
        setCatalogByConfigId((current) => ({
          ...current,
          [modelConfigId]: payload
        }));
      } catch {
        const fallback = buildFallbackCatalog(provider, "Use snapshot catalog");
        catalogCacheRef.current[cacheKey] = {
          expiresAt: now + 30 * 60 * 1000,
          payload: fallback
        };
        setCatalogByConfigId((current) => ({
          ...current,
          [modelConfigId]: fallback
        }));
      }
    },
    [modelConfigsClient]
  );

  const refreshModelConfigs = useCallback(async () => {
    setLoadingModelConfigs(true);
    try {
      const list = await modelConfigsClient.list();
      setModelConfigs(list);
      if (!defaultModelConfigId && list[0]) {
        setDefaultModelConfigId(list[0].model_config_id);
      }

      setDraftsById((current) => {
        const next: Record<string, ModelDraft> = {};
        for (const item of list) {
          const state = rowStateRef.current[item.model_config_id];
          if (state === "error" && current[item.model_config_id]) {
            next[item.model_config_id] = current[item.model_config_id];
          } else {
            next[item.model_config_id] = modelDraftFromConfig(item);
          }
        }
        draftsRef.current = next;
        return next;
      });

      for (const item of list) {
        void refreshModelCatalog(item.model_config_id, item.provider, item.base_url);
      }
    } catch (error) {
      addToast({
        title: t("models.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setLoadingModelConfigs(false);
    }
  }, [
    addToast,
    defaultModelConfigId,
    modelConfigsClient,
    refreshModelCatalog,
    setDefaultModelConfigId,
    t
  ]);

  useEffect(() => {
    void refreshModelConfigs();
  }, [refreshModelConfigs]);

  useEffect(() => {
    return () => {
      const timers = saveTimersRef.current;
      for (const timer of Object.values(timers)) {
        window.clearTimeout(timer);
      }
      saveTimersRef.current = {};
    };
  }, []);

  const commitRuntimeUrl = useCallback(() => {
    setRuntimeState("saving");
    setRuntimeHint(t("settings.saveState.saving"));
    try {
      setRuntimeUrl(runtimeDraft);
      setRuntimeState("idle");
      setRuntimeHint(undefined);
    } catch (error) {
      setRuntimeDraft(runtimeUrl);
      setRuntimeState("error");
      setRuntimeHint((error as Error).message);
      addToast({
        title: t("settings.runtimeUrlInvalid"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  }, [addToast, runtimeDraft, runtimeUrl, setRuntimeUrl, t]);

  const persistModelConfig = useCallback(
    async (modelConfigId: string) => {
      if (!writable) {
        return;
      }

      const draft = draftsRef.current[modelConfigId];
      const currentConfig = modelConfigsRef.current.find((item) => item.model_config_id === modelConfigId);
      if (!draft || !currentConfig) {
        return;
      }

      const validation = validateModelDraft(draft, {
        requireSecretRef: modelConfigsClient.kind === "local"
      });
      if (validation.error) {
        setRowStateById((current) => ({
          ...current,
          [modelConfigId]: "error"
        }));
        setRowHintById((current) => ({
          ...current,
          [modelConfigId]: validation.error
        }));
        return;
      }

      setRowStateById((current) => ({
        ...current,
        [modelConfigId]: "saving"
      }));
      setRowHintById((current) => ({
        ...current,
        [modelConfigId]: t("settings.saveState.saving")
      }));

      try {
        await modelConfigsClient.update(modelConfigId, validation.payload);
        setRowStateById((current) => ({
          ...current,
          [modelConfigId]: "idle"
        }));
        setRowHintById((current) => ({
          ...current,
          [modelConfigId]: undefined
        }));
        void refreshModelConfigs();
      } catch (error) {
        setRowStateById((current) => ({
          ...current,
          [modelConfigId]: "error"
        }));
        setRowHintById((current) => ({
          ...current,
          [modelConfigId]: (error as Error).message
        }));
        addToast({
          title: t("models.updateFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [addToast, modelConfigsClient, refreshModelConfigs, t, writable]
  );

  const scheduleModelConfigSave = useCallback(
    (modelConfigId: string) => {
      if (!writable) {
        return;
      }
      const timers = saveTimersRef.current;
      const existing = timers[modelConfigId];
      if (existing) {
        window.clearTimeout(existing);
      }
      timers[modelConfigId] = window.setTimeout(() => {
        delete timers[modelConfigId];
        void persistModelConfig(modelConfigId);
      }, 600);
    },
    [persistModelConfig, writable]
  );

  const flushModelConfigSave = useCallback(
    (modelConfigId: string) => {
      const timers = saveTimersRef.current;
      const existing = timers[modelConfigId];
      if (existing) {
        window.clearTimeout(existing);
        delete timers[modelConfigId];
      }
      void persistModelConfig(modelConfigId);
    },
    [persistModelConfig]
  );

  const patchDraft = useCallback(
    (modelConfigId: string, patch: Partial<ModelDraft>, flush = false) => {
      setDraftsById((current) => {
        const existing = current[modelConfigId];
        if (!existing) {
          return current;
        }

        const next = {
          ...current,
          [modelConfigId]: {
            ...existing,
            ...patch
          }
        };
        draftsRef.current = next;
        return next;
      });

      setRowStateById((current) => ({
        ...current,
        [modelConfigId]: "idle"
      }));
      setRowHintById((current) => ({
        ...current,
        [modelConfigId]: undefined
      }));

      if (flush) {
        flushModelConfigSave(modelConfigId);
      } else {
        scheduleModelConfigSave(modelConfigId);
      }
    },
    [flushModelConfigSave, scheduleModelConfigSave]
  );

  const onAddModelConfig = useCallback(async () => {
    if (!writable) {
      return;
    }

    const provider: ProviderKey = "openai";
    const defaultModel = PROVIDER_METADATA[provider].fallbackModels[0]?.model_id ?? "gpt-5-mini";
    const defaultBaseUrl = PROVIDER_METADATA[provider].defaultBaseUrl;

    try {
      if (modelConfigsClient.kind === "remote") {
        const apiKey = window.prompt(t("models.apiKeyPrompt"), "")?.trim() ?? "";
        if (!apiKey) {
          addToast({
            title: t("models.saveFailed"),
            description: t("models.apiKeyRequired"),
            variant: "error"
          });
          return;
        }

        await modelConfigsClient.create({
          provider,
          model: defaultModel,
          base_url: defaultBaseUrl,
          temperature: 0,
          max_tokens: null,
          api_key: apiKey
        });
      } else {
        await modelConfigsClient.create({
          provider,
          model: defaultModel,
          base_url: defaultBaseUrl,
          temperature: 0,
          max_tokens: null,
          secret_ref: `keychain:${provider}:default`
        });
      }

      await refreshModelConfigs();
      addToast({
        title: t("models.saveSuccess"),
        description: `${provider}:${defaultModel}`,
        variant: "success"
      });
    } catch (error) {
      addToast({
        title: t("models.saveFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  }, [addToast, modelConfigsClient, refreshModelConfigs, t, writable]);

  const onDeleteModel = useCallback(
    async (modelConfigId: string) => {
      if (!writable || !modelConfigsClient.supportsDelete) {
        return;
      }

      try {
        await modelConfigsClient.delete(modelConfigId);
        if (defaultModelConfigId === modelConfigId) {
          setDefaultModelConfigId(undefined);
        }
        await refreshModelConfigs();
        addToast({
          title: t("models.deleteSuccess"),
          variant: "success"
        });
      } catch (error) {
        addToast({
          title: t("models.deleteFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [addToast, defaultModelConfigId, modelConfigsClient, refreshModelConfigs, setDefaultModelConfigId, t, writable]
  );

  const sectionMeta = useMemo(
    () => ({
      label: t(`settings.sections.${activeSection}`),
      description: t(`settings.sections.${activeSection}Description`)
    }),
    [activeSection, t]
  );

  return (
    <div className="mx-auto w-full max-w-5xl p-page">
      <div className="mb-6">
        <h1 className="text-title font-semibold text-foreground">{sectionMeta.label}</h1>
        <p className="text-small text-muted-foreground">{sectionMeta.description}</p>
      </div>

      {activeSection === "general" ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("settings.sections.general")}</CardTitle>
            <CardDescription>{t("settings.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="divide-y divide-border-subtle rounded-panel border border-border-subtle bg-background/40">
              <SettingRow
                title={t("settings.languageLabel")}
                description={t("settings.languageHelp")}
                control={
                  <select
                    className="h-10 w-full rounded-control border border-border bg-background px-2 text-body text-foreground"
                    value={locale}
                    onChange={(event) => void setLocale(event.target.value as SupportedLocale)}
                  >
                    {(["zh-CN", "en-US"] satisfies SupportedLocale[]).map((item) => (
                      <option key={item} value={item}>
                        {t(`app.locale.${item}`)}
                      </option>
                    ))}
                  </select>
                }
              />
              <SettingRow
                title={t("settings.themeLabel")}
                description={t("settings.themeDescription")}
                control={
                  <select
                    className="h-10 w-full rounded-control border border-border bg-background px-2 text-body text-foreground"
                    value={theme}
                    onChange={(event) => setTheme(event.target.value as ThemeMode)}
                  >
                    {(["dark", "light"] as ThemeMode[]).map((item) => (
                      <option key={item} value={item}>
                        {t(`settings.theme.${item}`)}
                      </option>
                    ))}
                  </select>
                }
              />
            </div>
          </CardContent>
        </Card>
      ) : null}

      {activeSection === "runtime" ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("settings.sections.runtime")}</CardTitle>
            <CardDescription>{t("settings.sections.runtimeDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="divide-y divide-border-subtle rounded-panel border border-border-subtle bg-background/40">
              <SettingRow
                title={t("settings.runtimeUrlLabel")}
                description={t("settings.runtimeUrlHelp")}
                status={runtimeState}
                statusLabel={runtimeHint}
                control={
                  <Input
                    value={runtimeDraft}
                    onChange={(event) => setRuntimeDraft(event.target.value)}
                    onBlur={commitRuntimeUrl}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") {
                        event.preventDefault();
                        commitRuntimeUrl();
                      }
                    }}
                  />
                }
              />
              <SettingRow
                title={t("settings.syncSection")}
                description={t("settings.syncDescription")}
                control={<SyncNowButton />}
              />
            </div>
          </CardContent>
        </Card>
      ) : null}

      {activeSection === "models" ? (
        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-4">
            <div>
              <CardTitle>{t("models.title")}</CardTitle>
              <CardDescription>{t("settings.modelSectionDescription")}</CardDescription>
            </div>
            <Button type="button" onClick={() => void onAddModelConfig()} disabled={!writable}>
              <Plus className="mr-2 h-4 w-4" />
              <span>{t("models.add")}</span>
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-panel border border-border-subtle bg-background/40">
              <SettingRow
                title={t("settings.defaultModel")}
                description={t("settings.defaultModelDescription")}
                control={
                  <select
                    className="h-10 w-full rounded-control border border-border bg-background px-2 text-body text-foreground"
                    value={defaultModelConfigId ?? ""}
                    onChange={(event) => setDefaultModelConfigId(event.target.value || undefined)}
                  >
                    <option value="">{t("settings.noDefaultModel")}</option>
                    {modelConfigs.map((item) => (
                      <option key={item.model_config_id} value={item.model_config_id}>
                        {providerLabel(item.provider)}: {item.model}
                      </option>
                    ))}
                  </select>
                }
              />
            </div>

            {loadingModelConfigs ? (
              <div className="flex items-center gap-2 text-small text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>{t("models.loading")}</span>
              </div>
            ) : null}

            {modelConfigs.map((item) => {
              const draft = draftsById[item.model_config_id] ?? modelDraftFromConfig(item);
              const rowState = rowStateById[item.model_config_id] ?? "idle";
              const rowHint = rowHintById[item.model_config_id];
              const catalog = catalogByConfigId[item.model_config_id] ?? buildFallbackCatalog(draft.provider);

              const modelOptions = [...catalog.items].sort((left, right) => {
                if (left.is_latest === right.is_latest) {
                  return left.model_id.localeCompare(right.model_id);
                }
                return left.is_latest ? -1 : 1;
              });

              return (
                <div key={item.model_config_id} className="overflow-hidden rounded-panel border border-border-subtle bg-background/40">
                  <div className="flex items-center justify-between border-b border-border-subtle px-4 py-2">
                    <p className="text-small text-muted-foreground">{t("models.modelConfigId", { id: item.model_config_id })}</p>
                    <Button
                      type="button"
                      size="sm"
                      variant="destructive"
                      disabled={!writable || !modelConfigsClient.supportsDelete}
                      onClick={() => void onDeleteModel(item.model_config_id)}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      <span>{t("models.delete")}</span>
                    </Button>
                  </div>

                  <div className="divide-y divide-border-subtle">
                    <SettingRow
                      compact
                      title={t("models.provider")}
                      description={t("models.providerDescription")}
                      status={rowState}
                      statusLabel={rowHint}
                      control={
                        <select
                          className="h-10 w-full rounded-control border border-border bg-background px-2 text-body text-foreground"
                          value={draft.provider}
                          onChange={(event) => {
                            const provider = event.target.value as ProviderKey;
                            const fallbackModel = PROVIDER_METADATA[provider].fallbackModels[0]?.model_id ?? draft.model;
                            patchDraft(item.model_config_id, {
                              provider,
                              model: fallbackModel,
                              base_url: PROVIDER_METADATA[provider].defaultBaseUrl || draft.base_url
                            });
                          }}
                          onBlur={() => flushModelConfigSave(item.model_config_id)}
                        >
                          {PROVIDER_ORDER.map((provider) => (
                            <option key={provider} value={provider}>
                              {providerLabel(provider)}
                            </option>
                          ))}
                        </select>
                      }
                    />

                    <SettingRow
                      compact
                      title={t("models.model")}
                      description={
                        catalog.fallback_used
                          ? t("models.catalogFallback")
                          : t("models.catalogLive")
                      }
                      control={
                        <div className="flex gap-2">
                          <select
                            className="h-10 min-w-0 flex-1 rounded-control border border-border bg-background px-2 text-body text-foreground"
                            value={draft.model}
                            onChange={(event) => patchDraft(item.model_config_id, { model: event.target.value })}
                            onBlur={() => flushModelConfigSave(item.model_config_id)}
                          >
                            {modelOptions.map((model) => (
                              <option key={model.model_id} value={model.model_id}>
                                {model.is_latest ? `${model.display_name} (${t("models.latest")})` : model.display_name}
                              </option>
                            ))}
                          </select>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() =>
                              void refreshModelCatalog(
                                item.model_config_id,
                                draft.provider,
                                draft.base_url,
                                { force: true }
                              )
                            }
                          >
                            <RefreshCcw className="h-4 w-4" />
                          </Button>
                        </div>
                      }
                    />

                    <SettingRow
                      compact
                      title={t("models.baseUrl")}
                      description={t("models.baseUrlDescription")}
                      control={
                        <Input
                          value={draft.base_url}
                          onChange={(event) => patchDraft(item.model_config_id, { base_url: event.target.value })}
                          onBlur={() => flushModelConfigSave(item.model_config_id)}
                        />
                      }
                    />

                    <SettingRow
                      compact
                      title={t("models.temperature")}
                      description={t("models.temperatureDescription")}
                      control={
                        <Input
                          value={draft.temperature}
                          onChange={(event) => patchDraft(item.model_config_id, { temperature: event.target.value })}
                          onBlur={() => flushModelConfigSave(item.model_config_id)}
                        />
                      }
                    />

                    <SettingRow
                      compact
                      title={t("models.maxTokens")}
                      description={t("models.maxTokensDescription")}
                      control={
                        <Input
                          value={draft.max_tokens}
                          onChange={(event) => patchDraft(item.model_config_id, { max_tokens: event.target.value })}
                          onBlur={() => flushModelConfigSave(item.model_config_id)}
                        />
                      }
                    />

                    {modelConfigsClient.kind === "local" ? (
                      <SettingRow
                        compact
                        title={t("models.secretRef")}
                        description={t("models.secretRefDescription")}
                        control={
                          <Input
                            value={draft.secret_ref}
                            onChange={(event) => patchDraft(item.model_config_id, { secret_ref: event.target.value })}
                            onBlur={() => flushModelConfigSave(item.model_config_id)}
                          />
                        }
                      />
                    ) : (
                      <SettingRow
                        compact
                        title={t("models.secretRef")}
                        description={t("models.secretRefDescription")}
                        control={
                          <div className="h-10 rounded-control border border-border-subtle bg-muted/50 px-3 text-small text-muted-foreground">
                            <span className="inline-flex h-full items-center truncate">{item.secret_ref}</span>
                          </div>
                        }
                      />
                    )}

                    {modelConfigsClient.kind === "remote" ? (
                      <SettingRow
                        compact
                        title={t("models.apiKey")}
                        description={t("models.apiKeyDescription")}
                        control={
                          <Input
                            type="password"
                            value={draft.api_key}
                            placeholder={t("models.apiKeyPlaceholder")}
                            onChange={(event) => patchDraft(item.model_config_id, { api_key: event.target.value })}
                            onBlur={() => flushModelConfigSave(item.model_config_id)}
                          />
                        }
                      />
                    ) : null}
                  </div>
                </div>
              );
            })}

            {!loadingModelConfigs && modelConfigs.length === 0 ? (
              <p className="text-small text-muted-foreground">{t("models.empty")}</p>
            ) : null}
          </CardContent>
        </Card>
      ) : null}

      <p className={cn("mt-4 text-small text-muted-foreground", activeSection === "models" ? "" : "hidden")}>
        {t("models.remoteHint")}
      </p>
    </div>
  );
}
