import { EmptyState } from "@/components/domain/feedback/EmptyState";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export interface ConversationTurn {
  executionId: string;
  status: string;
  createdAt: string;
  userText: string;
  assistantText: string;
}

interface ConversationTranscriptPanelProps {
  title: string;
  emptyTitle: string;
  emptyDescription: string;
  turns: ConversationTurn[];
}

export function ConversationTranscriptPanel({
  title,
  emptyTitle,
  emptyDescription,
  turns
}: ConversationTranscriptPanelProps) {
  return (
    <Card className="min-h-0">
      <CardHeader className="pb-2">
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-4.5rem)] min-h-0 overflow-auto scrollbar-subtle">
        {turns.length === 0 ? (
          <EmptyState title={emptyTitle} description={emptyDescription} />
        ) : (
          <div className="space-y-4">
            {turns.map((turn) => (
              <article key={turn.executionId} className="space-y-2">
                <div className="flex justify-end">
                  <div className="max-w-[85%] rounded-panel border border-border-subtle bg-accent/15 px-3 py-2 text-small text-foreground">
                    <p className="whitespace-pre-wrap break-words">{turn.userText}</p>
                  </div>
                </div>
                <div className="flex justify-start">
                  <div className="max-w-[85%] rounded-panel border border-border-subtle bg-background/60 px-3 py-2 text-small text-foreground">
                    <p className="mb-1 text-xs text-muted-foreground">
                      {turn.executionId} · {turn.status}
                    </p>
                    <p className="whitespace-pre-wrap break-words">{turn.assistantText}</p>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
