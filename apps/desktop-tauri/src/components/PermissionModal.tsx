interface PermissionModalProps {
  toolName: string;
  args: Record<string, unknown>;
  onApprove: () => void;
  onDeny: () => void;
}

export function PermissionModal({ toolName, args, onApprove, onDeny }: PermissionModalProps) {
  return (
    <div className="modal-backdrop">
      <div className="modal">
        <h3>Permission Required</h3>
        <p>
          Tool <code>{toolName}</code> requests confirmation.
        </p>
        <pre>{JSON.stringify(args, null, 2)}</pre>
        <div className="modal-actions">
          <button onClick={onDeny}>Deny</button>
          <button onClick={onApprove}>Approve</button>
        </div>
      </div>
    </div>
  );
}
