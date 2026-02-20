import { useState } from "react";

import { runSyncNow } from "../api/syncClient";

export function SyncNowButton() {
  const [status, setStatus] = useState<string>("");

  const onClick = async () => {
    try {
      const result = await runSyncNow();
      setStatus(`pushed=${result.pushed} pulled=${result.pulled}`);
    } catch (error) {
      setStatus((error as Error).message);
    }
  };

  return (
    <div>
      <button onClick={onClick}>Sync now</button>
      {status && <p>{status}</p>}
    </div>
  );
}
