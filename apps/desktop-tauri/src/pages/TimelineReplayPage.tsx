import { FormEvent, useState } from "react";

import { listRuns } from "../api/runtimeClient";
import { EventTimeline } from "../components/EventTimeline";
import { useReplay } from "../hooks/useReplay";

export function ReplayPage() {
  const [sessionId, setSessionId] = useState("session-demo");
  const [runs, setRuns] = useState<Array<Record<string, string>>>([]);
  const [runId, setRunId] = useState<string>();

  const { events, loading } = useReplay(runId);

  const onLoadRuns = async (event: FormEvent) => {
    event.preventDefault();
    const payload = await listRuns(sessionId);
    setRuns(payload.runs);
  };

  return (
    <section className="panel">
      <h2>Replay</h2>
      <form onSubmit={onLoadRuns} className="form-grid">
        <label>
          Session ID
          <input value={sessionId} onChange={(e) => setSessionId(e.target.value)} />
        </label>
        <button type="submit">Load Runs</button>
      </form>

      <ul>
        {runs.map((run) => (
          <li key={run.run_id}>
            <button onClick={() => setRunId(run.run_id)}>{run.run_id}</button> ({run.status})
          </li>
        ))}
      </ul>

      {loading ? <p>Loading replay...</p> : <EventTimeline events={events} />}
    </section>
  );
}
