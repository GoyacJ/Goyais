import { FormEvent, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { createModelConfig, listModelConfigs } from "@/api/runtimeClient";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";

export function ModelConfigsPage() {
  const { t } = useTranslation();
  const [provider, setProvider] = useState("openai");
  const [model, setModel] = useState("gpt-4.1-mini");
  const [secretRef, setSecretRef] = useState("keychain:openai:default");
  const [modelConfigs, setModelConfigs] = useState<Array<Record<string, string>>>([]);
  const { addToast } = useToast();

  const refresh = useCallback(async () => {
    try {
      const payload = await listModelConfigs();
      setModelConfigs(payload.model_configs);
    } catch (error) {
      addToast({
        title: t("models.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  }, [addToast, t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      await createModelConfig({ provider, model, secret_ref: secretRef });
      addToast({
        title: t("models.saveSuccess"),
        description: `${provider}:${model}`,
        variant: "success"
      });
      await refresh();
    } catch (error) {
      addToast({
        title: t("models.saveFailed"),
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
                onChange={(event) => setProvider(event.target.value)}
                className="h-10 rounded-control border border-border bg-background px-2"
              >
                <option value="openai">openai</option>
                <option value="anthropic">anthropic</option>
              </select>
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.model")}
              <Input value={model} onChange={(event) => setModel(event.target.value)} />
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("models.secretRef")}
              <Input value={secretRef} onChange={(event) => setSecretRef(event.target.value)} />
            </label>
            <Button className="w-full" type="submit">
              {t("models.save")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("models.listTitle")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {modelConfigs.map((item) => (
            <div key={item.model_config_id} className="rounded-control border border-border-subtle bg-background/60 p-2">
              <p className="text-body text-foreground">
                {item.provider}:{item.model}
              </p>
              <p className="text-small text-muted-foreground">{item.secret_ref}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
