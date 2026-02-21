import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { DataModelConfig, getModelConfigsClient } from "@/api/dataSource";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { PROVIDER_ORDER, providerLabel, type ProviderKey } from "@/types/modelCatalog";
import {
  selectCurrentPermissions,
  selectCurrentProfile,
  selectCurrentWorkspaceKind,
  useWorkspaceStore
} from "@/stores/workspaceStore";

function parseOptionalNumber(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
}

export function canManageModelConfigs(workspaceKind: "local" | "remote", permissions: string[]): boolean {
  if (workspaceKind === "local") {
    return true;
  }
  return permissions.includes("modelconfig:manage");
}

export function ModelConfigsPage() {
  const { t } = useTranslation();
  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const modelConfigsClient = useMemo(() => getModelConfigsClient(currentProfile), [currentProfile]);
  const writable = canManageModelConfigs(workspaceKind, permissions);

  const [provider, setProvider] = useState<ProviderKey>("openai");
  const [model, setModel] = useState("gpt-4.1-mini");
  const [baseUrl, setBaseUrl] = useState("");
  const [temperature, setTemperature] = useState("0");
  const [maxTokens, setMaxTokens] = useState("");
  const [secretRef, setSecretRef] = useState("keychain:openai:default");
  const [apiKey, setApiKey] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [modelConfigs, setModelConfigs] = useState<DataModelConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const { addToast } = useToast();

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      setModelConfigs(await modelConfigsClient.list());
    } catch (error) {
      addToast({
        title: t("models.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setLoading(false);
    }
  }, [addToast, modelConfigsClient, t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const resetForm = () => {
    setProvider("openai");
    setModel("gpt-4.1-mini");
    setBaseUrl("");
    setTemperature("0");
    setMaxTokens("");
    setSecretRef("keychain:openai:default");
    setApiKey("");
    setEditingId(null);
  };

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!writable) {
      return;
    }

    const temperatureValue = parseOptionalNumber(temperature);
    const maxTokensValue = parseOptionalNumber(maxTokens);

    try {
      if (modelConfigsClient.kind === "remote") {
        if (editingId) {
          await modelConfigsClient.update(editingId, {
            provider,
            model,
            base_url: baseUrl.trim() || null,
            temperature: temperatureValue,
            max_tokens: maxTokensValue,
            api_key: apiKey.trim() || undefined
          });
          addToast({
            title: t("models.updateSuccess"),
            description: `${provider}:${model}`,
            variant: "success"
          });
        } else {
          await modelConfigsClient.create({
            provider,
            model,
            base_url: baseUrl.trim() || null,
            temperature: temperatureValue,
            max_tokens: maxTokensValue,
            api_key: apiKey
          });
          addToast({
            title: t("models.saveSuccess"),
            description: `${provider}:${model}`,
            variant: "success"
          });
        }
      } else {
        await modelConfigsClient.create({
          provider,
          model,
          base_url: baseUrl.trim() || null,
          temperature: temperatureValue,
          max_tokens: maxTokensValue,
          secret_ref: secretRef
        });
        addToast({
          title: t("models.saveSuccess"),
          description: `${provider}:${model}`,
          variant: "success"
        });
      }

      resetForm();
      await refresh();
    } catch (error) {
      addToast({
        title: editingId ? t("models.updateFailed") : t("models.saveFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  const onEdit = (item: DataModelConfig) => {
    if (!writable || modelConfigsClient.kind !== "remote") {
      return;
    }

    setEditingId(item.model_config_id);
    setProvider(item.provider);
    setModel(item.model);
    setBaseUrl(item.base_url ?? "");
    setTemperature(item.temperature === null ? "" : String(item.temperature));
    setMaxTokens(item.max_tokens === null ? "" : String(item.max_tokens));
    setApiKey("");
  };

  const onDelete = async (item: DataModelConfig) => {
    if (!writable || !modelConfigsClient.supportsDelete) {
      return;
    }

    try {
      await modelConfigsClient.delete(item.model_config_id);
      addToast({
        title: t("models.deleteSuccess"),
        variant: "success"
      });
      await refresh();
    } catch (error) {
      addToast({
        title: t("models.deleteFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  return (
    <div className="grid gap-panel lg:grid-cols-[22rem_minmax(0,1fr)]">
      <Card>
        <CardHeader>
          <CardTitle>{t("models.title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-form" onSubmit={onSubmit}>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.provider")}
              <select
                value={provider}
                onChange={(event) => setProvider(event.target.value as ProviderKey)}
                className="h-10 rounded-control border border-border bg-background px-2"
              >
                {PROVIDER_ORDER.map((key) => (
                  <option key={key} value={key}>
                    {providerLabel(key)}
                  </option>
                ))}
              </select>
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.model")}
              <Input value={model} onChange={(event) => setModel(event.target.value)} />
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.baseUrl")}
              <Input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} />
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.temperature")}
              <Input value={temperature} onChange={(event) => setTemperature(event.target.value)} />
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.maxTokens")}
              <Input value={maxTokens} onChange={(event) => setMaxTokens(event.target.value)} />
            </label>
            {modelConfigsClient.kind === "remote" ? (
              <label className="grid gap-1 text-small text-muted-foreground">
                {t("models.apiKey")}
                <Input type="password" value={apiKey} onChange={(event) => setApiKey(event.target.value)} />
              </label>
            ) : (
              <label className="grid gap-1 text-small text-muted-foreground">
                {t("models.secretRef")}
                <Input value={secretRef} onChange={(event) => setSecretRef(event.target.value)} />
              </label>
            )}

            <Button className="w-full" type="submit" disabled={!writable || loading}>
              {editingId ? t("models.update") : t("models.save")}
            </Button>
            {editingId ? (
              <Button className="w-full" type="button" variant="outline" onClick={resetForm}>
                {t("models.cancelEdit")}
              </Button>
            ) : null}
          </form>
          {modelConfigsClient.kind === "remote" ? (
            <p className="mt-3 text-small text-muted-foreground">{t("models.remoteHint")}</p>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("models.listTitle")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {loading ? <p className="text-small text-muted-foreground">{t("models.loading")}</p> : null}
          {modelConfigs.map((item) => (
            <div key={item.model_config_id} className="rounded-control border border-border-subtle bg-background/60 p-2">
              <p className="text-body text-foreground">
                {item.provider}:{item.model}
              </p>
              <p className="text-small text-muted-foreground">
                {t("models.secretRefLabel", {
                  secretRef: item.secret_ref
                })}
              </p>
              {modelConfigsClient.kind === "remote" && writable ? (
                <div className="mt-2 flex gap-2">
                  <Button size="sm" variant="outline" onClick={() => onEdit(item)}>
                    {t("models.edit")}
                  </Button>
                  <Button size="sm" variant="destructive" onClick={() => void onDelete(item)}>
                    {t("models.delete")}
                  </Button>
                </div>
              ) : null}
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
