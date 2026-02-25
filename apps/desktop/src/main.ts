import { createApp } from "vue";

import App from "./App.vue";
import { initializeGeneralSettings } from "./modules/workspace/store/generalSettingsStore";
import { router } from "./router";
import { pinia } from "./shared/stores/pinia";
import { initializeTheme } from "./shared/stores/themeStore";
import "virtual:uno.css";
import "./styles/tokens.css";
import "./styles/theme-profiles.css";
import "./styles/base.css";

initializeTheme();
void initializeGeneralSettings();
createApp(App).use(pinia).use(router).mount("#app");
