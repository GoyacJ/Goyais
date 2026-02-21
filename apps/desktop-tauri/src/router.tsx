import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "./App";
import { PermissionGate } from "./components/domain/workspace/PermissionGate";
import { ConversationPage } from "./pages/ConversationPage";
import { SettingsPage } from "./pages/SettingsPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      {
        index: true,
        element: (
          <PermissionGate routePath="/">
            <ConversationPage />
          </PermissionGate>
        )
      },
      {
        path: "settings/:section?",
        element: (
          <PermissionGate routePath="/settings">
            <SettingsPage />
          </PermissionGate>
        )
      }
    ]
  }
]);
