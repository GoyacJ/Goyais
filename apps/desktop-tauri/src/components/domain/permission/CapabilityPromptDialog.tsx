import { Globe, ShieldAlert, Terminal, TriangleAlert } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { classifyToolRisk } from "@/lib/risk";
import { useExecutionStore } from "@/stores/executionStore";
import type { PendingConfirmation } from "@/stores/executionStore";

interface CapabilityPromptDialogProps {
  item?: PendingConfirmation;
  open: boolean;
  onClose: () => void;
  onDecision: (mode: "once" | "always" | "deny") => void;
}

export function CapabilityPromptDialog({ item, open, onClose, onDecision }: CapabilityPromptDialogProps) {
  const { t } = useTranslation();
  const risk = useMemo(() => (item ? classifyToolRisk(item.toolName, item.args) : null), [item]);
  const lastPlan = useExecutionStore((state) => state.lastPlan);

  const isPlanApproval = item?.toolName === "plan_approval";

  if (isPlanApproval) {
    return (
      <Dialog open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
        <DialogContent
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              onDecision("deny");
            }
            if (event.key === "Enter") {
              event.preventDefault();
              onDecision("once");
            }
          }}
        >
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-info" />
              {t("conversation.planApprovalTitle")}
            </DialogTitle>
            <DialogDescription>{t("conversation.planApprovalDescription")}</DialogDescription>
          </DialogHeader>

          {lastPlan ? (
            <div className="space-y-3">
              {lastPlan.summary ? (
                <div>
                  <p className="mb-1 text-xs font-medium text-foreground">{t("conversation.planSummary")}</p>
                  <p className="rounded-control border border-border-subtle bg-background/60 p-2 text-small text-muted-foreground">
                    {lastPlan.summary}
                  </p>
                </div>
              ) : null}
              {lastPlan.steps && lastPlan.steps.length > 0 ? (
                <div>
                  <p className="mb-1 text-xs font-medium text-foreground">{t("conversation.planSteps")}</p>
                  <ol className="list-decimal space-y-1 pl-4 text-small text-muted-foreground">
                    {lastPlan.steps.map((step, idx) => (
                      <li key={idx}>{step}</li>
                    ))}
                  </ol>
                </div>
              ) : null}
            </div>
          ) : null}

          <DialogFooter>
            <Button variant="outline" onClick={() => onDecision("deny")}>
              {t("conversation.planReject")}
            </Button>
            <Button onClick={() => onDecision("once")}>{t("conversation.planApprove")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
      <DialogContent
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            onDecision("deny");
          }
          if (event.key === "Enter") {
            event.preventDefault();
            onDecision("once");
          }
        }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldAlert className="h-4 w-4 text-warning" />
            {t("permission.dialog.title")}
          </DialogTitle>
          <DialogDescription>
            {item ? t("permission.dialog.description", { toolName: item.toolName }) : t("permission.dialog.none")}
          </DialogDescription>
        </DialogHeader>

        {item && risk ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-small">
              <Badge variant="warning">{t("permission.riskLabel", { risk: risk.primary })}</Badge>
              <span className="text-muted-foreground">{t("permission.callId", { id: item.callId })}</span>
            </div>
            <p className="rounded-control border border-warning/40 bg-warning/10 p-2 text-small text-warning">
              {t("permission.dialog.reviewHint")}
            </p>

            <details open className="rounded-control border border-border-subtle bg-background/60 p-2">
              <summary className="cursor-pointer text-small text-foreground">{t("permission.dialog.details")}</summary>
              <div className="mt-2 space-y-2 text-small text-muted-foreground">
                {risk.details.command ? (
                  <div>
                    <div className="mb-1 flex items-center gap-1 text-foreground">
                      <Terminal className="h-3.5 w-3.5" />
                      {t("permission.dialog.command")}
                    </div>
                    <pre className="rounded-control border border-border-subtle bg-background p-2 text-code">{risk.details.command}</pre>
                  </div>
                ) : null}

                {risk.details.cwd ? <div>{t("permission.dialog.cwd", { cwd: risk.details.cwd })}</div> : null}

                {risk.details.paths.length > 0 ? (
                  <div>
                    <div className="mb-1 flex items-center gap-1 text-foreground">
                      <TriangleAlert className="h-3.5 w-3.5" />
                      {t("permission.dialog.paths")}
                    </div>
                    <ul className="list-disc space-y-1 pl-4">
                      {risk.details.paths.map((path) => (
                        <li key={path}>{path}</li>
                      ))}
                    </ul>
                    {risk.details.pathOutsideWorkspace ? (
                      <p className="mt-1 rounded-control border border-destructive/40 bg-destructive/10 px-2 py-1 text-destructive">
                        {t("permission.dialog.outsideWorkspace")}
                      </p>
                    ) : null}
                  </div>
                ) : null}

                {risk.details.domains.length > 0 ? (
                  <div>
                    <div className="mb-1 flex items-center gap-1 text-foreground">
                      <Globe className="h-3.5 w-3.5" />
                      {t("permission.dialog.domains")}
                    </div>
                    <ul className="list-disc space-y-1 pl-4">
                      {risk.details.domains.map((domain) => (
                        <li key={domain}>{domain}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </div>
            </details>
          </div>
        ) : null}

        <DialogFooter>
          <Button variant="outline" onClick={() => onDecision("deny")}>
            {t("permission.dialog.deny")}
          </Button>
          <Button variant="secondary" onClick={() => onDecision("always")}>
            {t("permission.dialog.always")}
          </Button>
          <Button onClick={() => onDecision("once")}>{t("permission.dialog.once")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
