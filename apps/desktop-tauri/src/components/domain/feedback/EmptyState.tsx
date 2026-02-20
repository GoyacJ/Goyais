import { Inbox } from "lucide-react";

import { Button } from "@/components/ui/button";

interface EmptyStateProps {
  title: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({ title, description, actionLabel, onAction }: EmptyStateProps) {
  return (
    <div className="flex h-full min-h-[10rem] flex-col items-center justify-center gap-2 rounded-panel border border-dashed border-border-subtle bg-background/40 p-4 text-center">
      <Inbox className="h-5 w-5 text-muted-foreground" />
      <p className="text-body font-medium text-foreground">{title}</p>
      {description ? <p className="max-w-sm text-small text-muted-foreground">{description}</p> : null}
      {actionLabel ? (
        <Button size="sm" variant="outline" onClick={onAction}>
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
}
