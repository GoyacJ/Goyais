import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Toast, ToastAction, ToastClose, ToastDescription, ToastProvider, ToastTitle, ToastViewport, useToast } from "@/components/ui/toast";

export function Toaster() {
  const { t } = useTranslation();
  const { toasts, removeToast } = useToast();

  return (
    <ToastProvider swipeDirection="right">
      {toasts.map((toast) => (
        <Toast
          key={toast.id}
          variant={toast.variant}
          open
          onOpenChange={(open) => {
            if (!open) removeToast(toast.id);
          }}
        >
          <div className="grid gap-1">
            <ToastTitle>{toast.title}</ToastTitle>
            {toast.description ? <ToastDescription>{toast.description}</ToastDescription> : null}
            {toast.diagnostic ? (
              <details className="rounded-control border border-border-subtle bg-background/60 p-1 text-small text-muted-foreground">
                <summary className="cursor-pointer">{t("toast.diagnostics")}</summary>
                <pre className="mt-1 max-h-28 overflow-auto whitespace-pre-wrap text-code">{toast.diagnostic}</pre>
                <Button
                  size="sm"
                  variant="outline"
                  className="mt-1"
                  onClick={() => void navigator.clipboard.writeText(toast.diagnostic ?? "")}
                >
                  {t("toast.copyDiagnostics")}
                </Button>
              </details>
            ) : null}
          </div>
          {toast.actionLabel ? (
            <ToastAction altText={toast.actionLabel} onClick={toast.onAction}>
              {toast.actionLabel}
            </ToastAction>
          ) : null}
          <ToastClose />
        </Toast>
      ))}
      <ToastViewport />
    </ToastProvider>
  );
}
