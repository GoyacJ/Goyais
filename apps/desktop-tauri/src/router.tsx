import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "./App";
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
      { index: true, element: <RunPage /> },
      { path: "projects", element: <ProjectsPage /> },
      { path: "models", element: <ModelConfigsPage /> },
      { path: "replay", element: <ReplayPage /> },
      { path: "settings", element: <SettingsPage /> }
    ]
  }
]);
