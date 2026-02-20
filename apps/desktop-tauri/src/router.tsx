import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "./App";
import { ModelConfigsPage } from "./pages/ModelConfigsPage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { ReplayPage } from "./pages/TimelineReplayPage";
import { RunPage } from "./pages/RunPage";
import { SettingsPage } from "./pages/SettingsPage";

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
