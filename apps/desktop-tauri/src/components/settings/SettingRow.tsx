import type { ReactNode } from "react";

import { cn } from "@/lib/cn";

interface SettingRowProps {
  title: string;
  description?: string;
  control: ReactNode;
  status?: "idle" | "saving" | "error";
  statusLabel?: string;
  compact?: boolean;
}

export function SettingRow({ title, description, control, status = "idle", statusLabel, compact = false }: SettingRowProps) {
  return (
    <div className={cn("flex items-start justify-between gap-4 px-4 py-3", compact ? "" : "min-h-[76px]") }>
      <div className="min-w-0 flex-1">
        <p className="text-body font-medium text-foreground">{title}</p>
        {description ? <p className="mt-0.5 text-small text-muted-foreground">{description}</p> : null}
      </div>
      <div className="flex min-w-[220px] flex-col items-end gap-1">
        <div className="w-full max-w-[280px]">{control}</div>
        {status !== "idle" || statusLabel ? (
          <p
            className={cn(
              "text-xs",
              status === "error"
                ? "text-destructive"
                : status === "saving"
                  ? "text-warning"
                  : "text-muted-foreground"
            )}
          >
            {statusLabel}
          </p>
        ) : null}
      </div>
    </div>
  );
}
