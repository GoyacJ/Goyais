import type { EventEnvelope } from "../types/generated";

export function EventTimeline({ events }: { events: EventEnvelope[] }) {
  return (
    <section className="panel">
      <h3>Timeline</h3>
      <ul className="timeline">
        {events.map((event) => (
          <li key={event.event_id}>
            <div className="timeline-header">
              <span>#{event.seq}</span>
              <strong>{event.type}</strong>
              <small>{event.ts}</small>
            </div>
            <pre>{JSON.stringify(event.payload, null, 2)}</pre>
          </li>
        ))}
      </ul>
    </section>
  );
}
