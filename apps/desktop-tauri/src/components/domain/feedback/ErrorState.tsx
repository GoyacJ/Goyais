import { TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ title, message, onRetry }: ErrorStateProps) {
  const { t } = useTranslation();

  return (
    <div className="rounded-panel border border-destructive/40 bg-destructive/10 p-3">
      <div className="mb-2 flex items-center gap-2 text-destructive">
        <TriangleAlert className="h-4 w-4" />
        <p className="text-body font-medium">{title || t("feedback.errorTitle")}</p>
      </div>
      <p className="text-small text-muted-foreground">{message}</p>
      <div className="mt-3 flex items-center gap-2">
        <Button size="sm" variant="destructive" onClick={() => void navigator.clipboard.writeText(message)}>
          {t("feedback.copyDiagnostics")}
        </Button>
        {onRetry ? (
          <Button size="sm" variant="outline" onClick={onRetry}>
            {t("feedback.retry")}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
