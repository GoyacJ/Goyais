import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { EmptyState } from "@/components/domain/feedback/EmptyState";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { normalizeEventEnvelope } from "@/lib/events";
import type { EventEnvelope, EventType } from "@/types/generated";

import { TimelineEventCard } from "./TimelineEventCard";
import { TimelineFilters } from "./TimelineFilters";

interface TimelinePanelProps {
  events: EventEnvelope[];
  selectedEventId?: string;
  onSelectEvent: (eventId: string) => void;
  onSelectToolCall?: (callId: string) => void;
}

function defaultFilters(): Record<EventType, boolean> {
  return {
    plan: true,
    tool_call: true,
    tool_result: true,
    patch: true,
    error: true,
    done: true
  };
}

export function TimelinePanel({ events, selectedEventId, onSelectEvent, onSelectToolCall }: TimelinePanelProps) {
  const { t } = useTranslation();
  const [filters, setFilters] = useState<Record<EventType, boolean>>(defaultFilters);
  const [autoFollow, setAutoFollow] = useState(true);
  const [collapsedMap, setCollapsedMap] = useState<Record<string, boolean>>({});
  const viewportRef = useRef<HTMLDivElement | null>(null);

  const normalized = useMemo(() => events.map(normalizeEventEnvelope), [events]);
  const filteredEvents = useMemo(() => normalized.filter((event) => filters[event.type]), [normalized, filters]);

  useEffect(() => {
    if (!autoFollow || !viewportRef.current) return;
    viewportRef.current.scrollTop = viewportRef.current.scrollHeight;
  }, [filteredEvents, autoFollow]);

  return (
    <Card className="h-full min-h-0">
      <CardHeader className="pb-2">
        <CardTitle>{t("timeline.title")}</CardTitle>
      </CardHeader>
      <CardContent className="flex h-[calc(100%-4.5rem)] min-h-0 flex-col gap-2">
        <TimelineFilters
          filters={filters}
          autoFollow={autoFollow}
          onToggleFilter={(eventType) => setFilters((state) => ({ ...state, [eventType]: !state[eventType] }))}
          onToggleAutoFollow={() => setAutoFollow((state) => !state)}
        />
        <div ref={viewportRef} className="min-h-0 flex-1 overflow-auto rounded-control border border-border-subtle bg-background/40 p-2 scrollbar-subtle">
          {filteredEvents.length === 0 ? (
            <EmptyState title={t("timeline.noEventsTitle")} description={t("timeline.noEventsDescription")} />
          ) : (
            <div className="space-y-2">
              {filteredEvents.map((event) => (
                <TimelineEventCard
                  key={event.id}
                  event={event}
                  selected={selectedEventId === event.id}
                  collapsed={collapsedMap[event.id] ?? true}
                  onToggleCollapsed={() =>
                    setCollapsedMap((state) => ({
                      ...state,
                      [event.id]: !state[event.id]
                    }))
                  }
                  onSelect={() => {
                    onSelectEvent(event.id);
                    if (event.callId && onSelectToolCall) {
                      onSelectToolCall(event.callId);
                    }
                  }}
                  onRetry={event.type === "error" ? () => undefined : undefined}
                />
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
