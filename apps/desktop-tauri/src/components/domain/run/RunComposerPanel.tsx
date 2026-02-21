import { type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";

interface ModelOption {
  model_config_id: string;
  provider: string;
  model: string;
}

interface RunComposerPanelProps {
  input: string;
  running: boolean;
  modelConfigId?: string;
  modelOptions: ModelOption[];
  onInputChange: (value: string) => void;
  onModelConfigIdChange: (value: string) => void;
  onSubmit: (event: FormEvent) => void;
}

export function RunComposerPanel({
  input,
  running,
  modelConfigId,
  modelOptions,
  onInputChange,
  onModelConfigIdChange,
  onSubmit
}: RunComposerPanelProps) {
  const { t } = useTranslation();

  return (
    <Card>
      <CardContent className="space-y-3 p-3">
        <form className="space-y-3" onSubmit={onSubmit}>
          <Textarea
            rows={4}
            placeholder={t("conversation.inputPlaceholder")}
            value={input}
            onChange={(event) => onInputChange(event.target.value)}
          />

          <div className="flex items-center gap-2">
            <label className="min-w-0 flex-1">
              <select
                className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
                value={modelConfigId ?? ""}
                onChange={(event) => onModelConfigIdChange(event.target.value)}
              >
                {modelOptions.map((item) => (
                  <option key={item.model_config_id} value={item.model_config_id}>
                    {item.provider}:{item.model}
                  </option>
                ))}
              </select>
            </label>
            <Button type="submit" disabled={running || !input.trim()}>
              {running ? t("conversation.sending") : t("conversation.send")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
