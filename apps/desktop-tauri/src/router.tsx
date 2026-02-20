import { createBrowserRouter, Navigate } from "react-router-dom";

import { AppShell } from "./App";
import { PermissionGate } from "./components/domain/workspace/PermissionGate";
import { ModelConfigsPage } from "./pages/ModelConfigsPage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { RunPage } from "./pages/RunPage";
import { SettingsPage } from "./pages/SettingsPage";
import { ReplayPage } from "./pages/TimelineReplayPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      {
        index: true,
        element: <Navigate to="/run" replace />
      },
      {
        path: "run",
        element: (
          <PermissionGate routePath="/run">
            <RunPage />
          </PermissionGate>
        )
      },
      {
        path: "projects",
        element: (
          <PermissionGate routePath="/projects">
            <ProjectsPage />
          </PermissionGate>
        )
      },
      {
        path: "models",
        element: (
          <PermissionGate routePath="/models">
            <ModelConfigsPage />
          </PermissionGate>
        )
      },
      {
        path: "replay",
        element: (
          <PermissionGate routePath="/replay">
            <ReplayPage />
          </PermissionGate>
        )
      },
      {
        path: "settings",
        element: (
          <PermissionGate routePath="/settings">
            <SettingsPage />
          </PermissionGate>
        )
      }
    ]
  }
]);
