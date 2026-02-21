import "./styles/globals.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router-dom";

import { AppProviders } from "./app/providers";
import { initializeI18n } from "./i18n";
import { router } from "./router";
import { useSettingsStore } from "./stores/settingsStore";

async function bootstrap() {
  await useSettingsStore.getState().hydrate();
  await initializeI18n(useSettingsStore.getState().locale);

  createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>
    </StrictMode>
  );
}

void bootstrap();
