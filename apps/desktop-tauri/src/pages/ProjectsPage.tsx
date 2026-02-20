import { FormEvent, useEffect, useState } from "react";

import { createProject, listProjects } from "../api/runtimeClient";

export function ProjectsPage() {
  const [name, setName] = useState("Demo Project");
  const [workspacePath, setWorkspacePath] = useState("/Users/goya/Repo/Git/Goyais");
  const [projects, setProjects] = useState<Array<Record<string, string>>>([]);

  const refresh = async () => {
    const payload = await listProjects();
    setProjects(payload.projects);
  };

  useEffect(() => {
    void refresh();
  }, []);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await createProject({ name, workspace_path: workspacePath });
    await refresh();
  };

  return (
    <section className="panel">
      <h2>Projects</h2>
      <form onSubmit={onSubmit} className="form-grid">
        <label>
          Name
          <input value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label>
          Workspace Path
          <input value={workspacePath} onChange={(e) => setWorkspacePath(e.target.value)} />
        </label>
        <button type="submit">Create Project</button>
      </form>
      <ul>
        {projects.map((project) => (
          <li key={project.project_id}>
            <strong>{project.name}</strong> - {project.workspace_path}
          </li>
        ))}
      </ul>
    </section>
  );
}
