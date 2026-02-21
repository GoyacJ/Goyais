import { usePermissionStore } from "../stores/permissionStore";

export function PermissionCenter() {
  const decisions = usePermissionStore((state) => state.decisions);

  return (
    <section className="panel">
      <h3>Permission Center</h3>
      <ul>
        {decisions.map((item) => (
          <li key={`${item.executionId}:${item.callId}`}>
            <code>{item.callId}</code> - {item.approved ? "approved" : "denied"} @ {item.decidedAt}
          </li>
        ))}
      </ul>
    </section>
  );
}
