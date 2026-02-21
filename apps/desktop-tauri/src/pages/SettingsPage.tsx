import { Loader2, Plus, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";

import {
  type DataModelConfig,
  getModelConfigsClient,
  type UpdateModelConfigInput
} from "@/api/dataSource";
import { McpPanel } from "@/components/settings/McpPanel";
import { SettingRow } from "@/components/settings/SettingRow";
import { SkillsPanel } from "@/components/settings/SkillsPanel";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import type { SupportedLocale } from "@/i18n/types";
import { cn } from "@/lib/cn";
import {
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
  type ModelCatalogItem,
  type ProviderKey,
  providerLabel
} from "@/types/modelCatalog";

type SettingsSection = "general" | "runtime" | "models" | "skills" | "mcp";
type RowSaveState = "idle" | "saving" | "error";

export const SETTINGS_MODEL_VISIBLE_FIELDS = [
  "provider",
  "model",
  "api_key",
  "base_url",
  "temperature",
  "max_tokens",
] as const;

export const SETTINGS_MODEL_LIST_COLUMNS = [
  "provider",
  "model",
  "base_url",
  "temperature",
  "max_tokens",
  "actions"
] as const;

type ModelDraft = {
  provider: ProviderKey;
  model: string;
  base_url: string;
  temperature: string;
  max_tokens: string;
  api_key: string;
};

type AddModelDraft = ModelDraft;
type EditModelDraft = ModelDraft;

function parseSection(section: string | undefined): SettingsSection {
  if (section === "runtime" || section === "models" || section === "skills" || section === "mcp") {
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

function normalizeHttpUrl(input: string): string {
  const normalized = input.trim();
  if (!normalized) {
    return "";
  }

  const parsed = new URL(normalized);
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("URL must use http or https");
  }

  return parsed.toString().replace(/\/+$/, "");
}

function modelDraftFromConfig(item: DataModelConfig): ModelDraft {
  return {
    provider: item.provider,
    model: item.model,
    base_url: item.base_url ?? PROVIDER_METADATA[item.provider].defaultBaseUrl,
    temperature: String(item.temperature ?? 0),
    max_tokens: item.max_tokens === null ? "" : String(item.max_tokens),
    api_key: ""
  };
}

function defaultModelDraft(provider: ProviderKey = "openai"): ModelDraft {
  return {
    provider,
    model: PROVIDER_METADATA[provider].fallbackModels[0]?.model_id ?? "",
    base_url: PROVIDER_METADATA[provider].defaultBaseUrl,
    temperature: "0",
    max_tokens: "",
    api_key: ""
  };
}

function canManageModelConfigs(workspaceKind: "local" | "remote", permissions: string[]): boolean {
  if (workspaceKind === "local") {
    return true;
  }
  return permissions.includes("modelconfig:manage");
}

function validateModelDraft(draft: ModelDraft): { payload: UpdateModelConfigInput; error?: string } {
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
      normalizeHttpUrl(normalizedBaseUrl);
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

  const payload: UpdateModelConfigInput = {
    provider: draft.provider,
    model,
    base_url: normalizedBaseUrl || null,
    temperature: temperatureValue,
    max_tokens: maxTokens
  };

  return { payload };
}

export function buildModelSuggestions(modelConfigs: DataModelConfig[], provider: ProviderKey): string[] {
  const providerOnly = [...new Set(
    modelConfigs
      .filter((item) => item.provider === provider)
      .map((item) => item.model.trim())
      .filter(Boolean)
  )].sort((left, right) => left.localeCompare(right));

  if (providerOnly.length > 0) {
    return providerOnly;
  }

  return [...new Set(
    modelConfigs
      .map((item) => item.model.trim())
      .filter(Boolean)
  )].sort((left, right) => left.localeCompare(right));
}

function normalizeModelId(value: string): string {
  return value.trim().toLowerCase();
}

export function isModelAvailableInCatalog(model: string, items: Pick<ModelCatalogItem, "model_id">[]): boolean {
  const configuredModelId = normalizeModelId(model);
  if (!configuredModelId) {
    return false;
  }
  return items.some((item) => normalizeModelId(item.model_id) === configuredModelId);
}

function formatOptionalNumber(value: number | null): string {
  if (value === null) {
    return "—";
  }
  return String(value);
}

function formatOptionalText(value: string | null): string {
  if (!value) {
    return "—";
  }
  return value;
}

interface ModelConfigDialogFieldsProps {
  draft: ModelDraft;
  setDraft: (updater: (current: ModelDraft) => ModelDraft) => void;
  modelSuggestions: string[];
  modelListId: string;
  hint?: string;
  state: RowSaveState;
  disabled: boolean;
}

function ModelConfigDialogFields({
  draft,
  setDraft,
  modelSuggestions,
  modelListId,
  hint,
  state,
  disabled
}: ModelConfigDialogFieldsProps) {
  const { t } = useTranslation();

  return (
    <>
      <div className="space-y-2.5">
        <div className="space-y-1">
          <p className="text-small text-muted-foreground">{t("models.provider")}</p>
          <select
            className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
            disabled={disabled}
            value={draft.provider}
            onChange={(event) => {
              const provider = event.target.value as ProviderKey;
              setDraft((current) => ({
                ...current,
                provider,
                model: PROVIDER_METADATA[provider].fallbackModels[0]?.model_id ?? current.model,
                base_url: PROVIDER_METADATA[provider].defaultBaseUrl
              }));
            }}
          >
            {PROVIDER_ORDER.map((provider) => (
              <option key={provider} value={provider}>
                {providerLabel(provider)}
              </option>
            ))}
          </select>
        </div>

        <div className="space-y-1">
          <p className="text-small text-muted-foreground">{t("models.model")}</p>
          <Input
            className="h-9"
            list={modelListId}
            disabled={disabled}
            value={draft.model}
            onChange={(event) => {
              const value = event.target.value;
              setDraft((current) => ({ ...current, model: value }));
            }}
          />
          <datalist id={modelListId}>
            {modelSuggestions.map((modelId) => (
              <option key={modelId} value={modelId} />
            ))}
          </datalist>
        </div>

        <div className="space-y-1">
          <p className="text-small text-muted-foreground">{t("models.apiKey")}</p>
          <Input
            type="password"
            className="h-9"
            disabled={disabled}
            value={draft.api_key}
            placeholder={t("models.apiKeyPlaceholder")}
            onChange={(event) => {
              const value = event.target.value;
              setDraft((current) => ({ ...current, api_key: value }));
            }}
          />
          <p className="text-xs text-muted-foreground">
            {t("models.apiKeyDescription")}
          </p>
        </div>

        <div className="space-y-1">
          <p className="text-small text-muted-foreground">{t("models.baseUrl")}</p>
          <Input
            className="h-9"
            disabled={disabled}
            value={draft.base_url}
            onChange={(event) => {
              const value = event.target.value;
              setDraft((current) => ({ ...current, base_url: value }));
            }}
          />
        </div>

        <div className="space-y-1">
          <p className="text-small text-muted-foreground">{t("models.temperature")}</p>
          <Input
            className="h-9"
            disabled={disabled}
            value={draft.temperature}
            onChange={(event) => {
              const value = event.target.value;
              setDraft((current) => ({ ...current, temperature: value }));
            }}
          />
        </div>

        <div className="space-y-1">
          <p className="text-small text-muted-foreground">{t("models.maxTokens")}</p>
          <Input
            className="h-9"
            disabled={disabled}
            value={draft.max_tokens}
            onChange={(event) => {
              const value = event.target.value;
              setDraft((current) => ({ ...current, max_tokens: value }));
            }}
          />
        </div>
      </div>

      {hint ? (
        <p className={cn("text-small", state === "error" ? "text-destructive" : "text-muted-foreground")}>
          {hint}
        </p>
      ) : null}
    </>
  );
}

export function SettingsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { addToast } = useToast();
  const params = useParams<{ section?: string }>();
  const activeSection = parseSection(params.section);

  const locale = useSettingsStore((state) => state.locale);
  const theme = useSettingsStore((state) => state.theme);
  const defaultModelConfigId = useSettingsStore((state) => state.defaultModelConfigId);
  const setLocale = useSettingsStore((state) => state.setLocale);
  const setTheme = useSettingsStore((state) => state.setTheme);
  const setDefaultModelConfigId = useSettingsStore((state) => state.setDefaultModelConfigId);

  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const modelConfigsClient = useMemo(() => getModelConfigsClient(currentProfile), [currentProfile]);
  const writable = canManageModelConfigs(workspaceKind, permissions);

  const [loadingModelConfigs, setLoadingModelConfigs] = useState(false);
  const [modelConfigs, setModelConfigs] = useState<DataModelConfig[]>([]);

  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [addDraft, setAddDraft] = useState<AddModelDraft>(() => defaultModelDraft());
  const [addState, setAddState] = useState<RowSaveState>("idle");
  const [addHint, setAddHint] = useState<string | undefined>(undefined);

  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingModelConfigId, setEditingModelConfigId] = useState<string | null>(null);
  const [editDraft, setEditDraft] = useState<EditModelDraft>(() => defaultModelDraft());
  const [editState, setEditState] = useState<RowSaveState>("idle");
  const [editHint, setEditHint] = useState<string | undefined>(undefined);
  const [checkingModelConfigId, setCheckingModelConfigId] = useState<string | null>(null);

  useEffect(() => {
    const valid = ["general", "runtime", "models", "skills", "mcp"];
    if (params.section && !valid.includes(params.section)) {
      navigate("/settings/general", { replace: true });
    }
  }, [navigate, params.section]);

  const refreshModelConfigs = useCallback(async () => {
    setLoadingModelConfigs(true);
    try {
      const list = await modelConfigsClient.list();
      setModelConfigs(list);
      if (!defaultModelConfigId && list[0]) {
        setDefaultModelConfigId(list[0].model_config_id);
      }
      if (defaultModelConfigId && !list.some((item) => item.model_config_id === defaultModelConfigId)) {
        setDefaultModelConfigId(undefined);
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
  }, [addToast, defaultModelConfigId, modelConfigsClient, setDefaultModelConfigId, t]);

  useEffect(() => {
    void refreshModelConfigs();
  }, [refreshModelConfigs]);

  const resetAddDraft = useCallback((provider: ProviderKey = "openai") => {
    setAddDraft(defaultModelDraft(provider));
    setAddState("idle");
    setAddHint(undefined);
  }, []);

  const resetEditDraft = useCallback((item?: DataModelConfig) => {
    setEditingModelConfigId(item?.model_config_id ?? null);
    setEditDraft(item ? modelDraftFromConfig(item) : defaultModelDraft());
    setEditState("idle");
    setEditHint(undefined);
  }, []);

  const onOpenAddModelDialog = useCallback(() => {
    resetAddDraft();
    setAddDialogOpen(true);
  }, [resetAddDraft]);

  const onOpenEditModelDialog = useCallback(
    (item: DataModelConfig) => {
      if (!writable) {
        return;
      }
      resetEditDraft(item);
      setEditDialogOpen(true);
    },
    [resetEditDraft, writable]
  );

  const onCreateModelConfig = useCallback(async () => {
    if (!writable) {
      return;
    }

    const validation = validateModelDraft(addDraft);
    if (validation.error) {
      setAddState("error");
      setAddHint(validation.error);
      return;
    }

    if (!addDraft.api_key.trim()) {
      setAddState("error");
      setAddHint(t("models.apiKeyRequired"));
      return;
    }

    setAddState("saving");
    setAddHint(t("settings.saveState.saving"));

    try {
      await modelConfigsClient.create({
        ...validation.payload,
        provider: addDraft.provider,
        model: addDraft.model.trim(),
        api_key: addDraft.api_key.trim()
      });

      await refreshModelConfigs();
      setAddDialogOpen(false);
      resetAddDraft();
      addToast({
        title: t("models.saveSuccess"),
        description: `${addDraft.provider}:${addDraft.model.trim()}`,
        variant: "success"
      });
    } catch (error) {
      const message = (error as Error).message;
      setAddState("error");
      setAddHint(message);
      addToast({
        title: t("models.saveFailed"),
        description: message,
        diagnostic: message,
        variant: "error"
      });
    }
  }, [addDraft, addToast, modelConfigsClient, refreshModelConfigs, resetAddDraft, t, writable]);

  const onSaveEditedModelConfig = useCallback(async () => {
    if (!writable || !editingModelConfigId) {
      return;
    }

    const validation = validateModelDraft(editDraft);
    if (validation.error) {
      setEditState("error");
      setEditHint(validation.error);
      return;
    }

    setEditState("saving");
    setEditHint(t("settings.saveState.saving"));

    try {
      await modelConfigsClient.update(editingModelConfigId, {
        ...validation.payload,
        api_key: editDraft.api_key.trim() || undefined
      });

      await refreshModelConfigs();
      setEditDialogOpen(false);
      resetEditDraft();
      addToast({
        title: t("models.updateSuccess"),
        description: `${editDraft.provider}:${editDraft.model.trim()}`,
        variant: "success"
      });
    } catch (error) {
      const message = (error as Error).message;
      setEditState("error");
      setEditHint(message);
      addToast({
        title: t("models.updateFailed"),
        description: message,
        diagnostic: message,
        variant: "error"
      });
    }
  }, [editDraft, editingModelConfigId, addToast, modelConfigsClient, refreshModelConfigs, resetEditDraft, t, writable]);

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

  const onConnectModel = useCallback(
    async (item: DataModelConfig) => {
      if (!modelConfigsClient.supportsModelCatalog || checkingModelConfigId) {
        return;
      }

      setCheckingModelConfigId(item.model_config_id);
      try {
        const catalog = await modelConfigsClient.listModels(item.model_config_id);
        const catalogItems = Array.isArray(catalog.items) ? catalog.items : [];
        const available = isModelAvailableInCatalog(item.model, catalogItems);
        if (available) {
          addToast({
            title: t("models.connectSuccess"),
            description: `${providerLabel(item.provider)}: ${item.model}`,
            variant: "success"
          });
          return;
        }

        addToast({
          title: t("models.connectUnavailable"),
          description: t("models.connectUnavailableDescription", { model: item.model }),
          variant: "warning"
        });
      } catch (error) {
        const message = (error as Error).message;
        addToast({
          title: t("models.connectFailed"),
          description: message,
          diagnostic: message,
          variant: "error"
        });
      } finally {
        setCheckingModelConfigId(null);
      }
    },
    [addToast, checkingModelConfigId, modelConfigsClient, t]
  );

  const sectionMeta = useMemo(
    () => ({
      label: t(`settings.sections.${activeSection}`),
      description: t(`settings.sections.${activeSection}Description`)
    }),
    [activeSection, t]
  );

  const workspaceScope = useMemo(() => {
    if (!currentProfile) {
      return t("workspace.unknown");
    }
    if (currentProfile.kind === "local") {
      return currentProfile.name;
    }
    const selectedWorkspaceId = currentProfile.remote?.selectedWorkspaceId;
    return selectedWorkspaceId ? `${currentProfile.name} · ${selectedWorkspaceId}` : currentProfile.name;
  }, [currentProfile, t]);
  const addModelSuggestions = useMemo(
    () => buildModelSuggestions(modelConfigs, addDraft.provider),
    [addDraft.provider, modelConfigs]
  );
  const editModelSuggestions = useMemo(
    () => buildModelSuggestions(modelConfigs, editDraft.provider),
    [editDraft.provider, modelConfigs]
  );

  return (
    <div className="mx-auto w-full max-w-4xl p-page">
      <div className="mb-4">
        <h1 className="text-title font-semibold text-foreground">{sectionMeta.label}</h1>
        <p className="text-small text-muted-foreground">{sectionMeta.description}</p>
      </div>

      {activeSection === "general" ? (
        <Card>
          <CardHeader className="pb-3">
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
                    className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
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
                    className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
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
          <CardHeader className="pb-3">
            <CardTitle>{t("settings.sections.runtime")}</CardTitle>
            <CardDescription>{t("settings.sections.runtimeDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="divide-y divide-border-subtle rounded-panel border border-border-subtle bg-background/40">
              <SettingRow
                title={t("settings.workspaceScope")}
                description={t("settings.workspaceScopeDescription")}
                control={
                  <p className="truncate text-small text-foreground" title={workspaceScope}>
                    {workspaceScope}
                  </p>
                }
              />
              <SettingRow
                title={t("settings.workspacePermission")}
                description={t("settings.workspacePermissionDescription")}
                control={
                  <p
                    className={cn(
                      "text-small font-medium",
                      writable ? "text-success" : "text-muted-foreground"
                    )}
                  >
                    {writable ? t("settings.permissionWritable") : t("settings.permissionReadonly")}
                  </p>
                }
              />
            </div>
          </CardContent>
        </Card>
      ) : null}

      {activeSection === "models" ? (
        <>
          <Card>
            <CardHeader className="flex flex-row items-start justify-between gap-3 pb-3">
              <div>
                <CardTitle>{t("models.title")}</CardTitle>
                <CardDescription>{t("settings.modelSectionDescription")}</CardDescription>
              </div>
              <Button type="button" size="sm" onClick={onOpenAddModelDialog} disabled={!writable}>
                <Plus className="mr-2 h-4 w-4" />
                <span>{t("models.add")}</span>
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="rounded-panel border border-border-subtle bg-background/40">
                <SettingRow
                  title={t("settings.workspaceScope")}
                  description={t("settings.workspaceScopeDescription")}
                  control={
                    <p className="truncate text-small text-foreground" title={workspaceScope}>
                      {workspaceScope}
                    </p>
                  }
                />
                <SettingRow
                  title={t("settings.workspacePermission")}
                  description={t("settings.workspacePermissionDescription")}
                  control={
                    <p
                      className={cn(
                        "text-small font-medium",
                        writable ? "text-success" : "text-muted-foreground"
                      )}
                    >
                      {writable ? t("settings.permissionWritable") : t("settings.permissionReadonly")}
                    </p>
                  }
                />
              </div>

              <div className="rounded-panel border border-border-subtle bg-background/40">
                <SettingRow
                  title={t("settings.defaultModel")}
                  description={t("settings.defaultModelDescription")}
                  control={
                    <select
                      className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
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

              {!loadingModelConfigs && modelConfigs.length > 0 ? (
                <div className="overflow-auto rounded-panel border border-border-subtle bg-background/40 scrollbar-subtle">
                  <table className="w-full min-w-[680px] text-small text-foreground">
                    <thead className="bg-muted/55 text-xs text-muted-foreground">
                      <tr>
                        <th className="px-3 py-2 text-left font-medium">{t("models.provider")}</th>
                        <th className="px-3 py-2 text-left font-medium">{t("models.model")}</th>
                        <th className="px-3 py-2 text-left font-medium">{t("models.baseUrl")}</th>
                        <th className="px-3 py-2 text-left font-medium">{t("models.temperature")}</th>
                        <th className="px-3 py-2 text-left font-medium">{t("models.maxTokens")}</th>
                        <th className="px-3 py-2 text-right font-medium">{t("models.actions")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border-subtle">
                      {modelConfigs.map((item) => (
                        <tr key={item.model_config_id} className="align-middle">
                          <td className="px-3 py-2">{providerLabel(item.provider)}</td>
                          <td className="px-3 py-2">{item.model}</td>
                          <td className="max-w-[22rem] truncate px-3 py-2" title={formatOptionalText(item.base_url)}>
                            {formatOptionalText(item.base_url)}
                          </td>
                          <td className="px-3 py-2">{formatOptionalNumber(item.temperature)}</td>
                          <td className="px-3 py-2">{formatOptionalNumber(item.max_tokens)}</td>
                          <td className="px-3 py-2">
                            <div className="flex justify-end gap-2">
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                disabled={!writable}
                                onClick={() => onOpenEditModelDialog(item)}
                              >
                                {t("models.edit")}
                              </Button>
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                disabled={!modelConfigsClient.supportsModelCatalog || checkingModelConfigId !== null}
                                onClick={() => void onConnectModel(item)}
                              >
                                {checkingModelConfigId === item.model_config_id ? (
                                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                ) : null}
                                {t("models.connect")}
                              </Button>
                              <Button
                                type="button"
                                size="sm"
                                variant="destructive"
                                disabled={!writable || !modelConfigsClient.supportsDelete}
                                onClick={() => void onDeleteModel(item.model_config_id)}
                              >
                                <Trash2 className="mr-2 h-4 w-4" />
                                {t("models.delete")}
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : null}

              {!loadingModelConfigs && modelConfigs.length === 0 ? (
                <p className="text-small text-muted-foreground">{t("models.empty")}</p>
              ) : null}
            </CardContent>
          </Card>

          <Dialog
            open={addDialogOpen}
            onOpenChange={(open) => {
              setAddDialogOpen(open);
              if (!open) {
                resetAddDraft();
              }
            }}
          >
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>{t("models.add")}</DialogTitle>
                <DialogDescription>{t("models.addDescription")}</DialogDescription>
              </DialogHeader>

              <ModelConfigDialogFields
                draft={addDraft}
                setDraft={(updater) => {
                  setAddDraft((current) => updater(current));
                  setAddState("idle");
                  setAddHint(undefined);
                }}
                modelSuggestions={addModelSuggestions}
                modelListId="model-suggestions-add"
                hint={addHint}
                state={addState}
                disabled={!writable || addState === "saving"}
              />

              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setAddDialogOpen(false)}>
                  {t("workspace.cancel")}
                </Button>
                <Button type="button" onClick={() => void onCreateModelConfig()} disabled={!writable || addState === "saving"}>
                  {addState === "saving" ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                  {t("models.save")}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Dialog
            open={editDialogOpen}
            onOpenChange={(open) => {
              setEditDialogOpen(open);
              if (!open) {
                resetEditDraft();
              }
            }}
          >
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>{t("models.edit")}</DialogTitle>
                <DialogDescription>{t("models.editDescription")}</DialogDescription>
              </DialogHeader>

              <ModelConfigDialogFields
                draft={editDraft}
                setDraft={(updater) => {
                  setEditDraft((current) => updater(current));
                  setEditState("idle");
                  setEditHint(undefined);
                }}
                modelSuggestions={editModelSuggestions}
                modelListId="model-suggestions-edit"
                hint={editHint}
                state={editState}
                disabled={!writable || editState === "saving"}
              />

              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setEditDialogOpen(false)}>
                  {t("workspace.cancel")}
                </Button>
                <Button
                  type="button"
                  onClick={() => void onSaveEditedModelConfig()}
                  disabled={!writable || editState === "saving"}
                >
                  {editState === "saving" ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                  {t("models.save")}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </>
      ) : null}

      <p className={cn("mt-3 text-small text-muted-foreground", activeSection === "models" ? "" : "hidden")}>
        {t("models.remoteHint")}
      </p>

      {activeSection === "skills" ? <SkillsPanel /> : null}
      {activeSection === "mcp" ? <McpPanel /> : null}
    </div>
  );
}
