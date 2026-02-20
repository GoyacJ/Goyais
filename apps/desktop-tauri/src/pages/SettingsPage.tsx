import { SyncNowButton } from "../components/SyncNowButton";

export function SettingsPage() {
  return (
    <section className="panel">
      <h2>Settings</h2>
      <p>Single-user sync</p>
      <SyncNowButton />
    </section>
  );
}
