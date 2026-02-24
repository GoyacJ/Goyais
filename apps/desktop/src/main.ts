import { createApp } from "vue";

import App from "./App.vue";
import { initializeGeneralSettings } from "./modules/workspace/store/generalSettingsStore";
import { router } from "./router";
import { initializeTheme } from "./shared/stores/themeStore";
import "./styles/tokens.css";
import "./styles/theme-profiles.css";
import "./styles/base.css";

initializeTheme();
void initializeGeneralSettings();
createApp(App).use(router).mount("#app");
