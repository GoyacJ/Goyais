import { Link, Outlet } from "react-router-dom";

export function AppShell() {
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>Goyais</h1>
        <nav>
          <Link to="/">Run</Link>
          <Link to="/projects">Projects</Link>
          <Link to="/models">Model Configs</Link>
          <Link to="/replay">Replay</Link>
          <Link to="/settings">Settings</Link>
        </nav>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
