import { createApp } from "vue";

import App from "./App.vue";
import { router } from "./router";
import { initializeTheme } from "./shared/stores/themeStore";
import "./styles/tokens.css";
import "./styles/base.css";

initializeTheme();
createApp(App).use(router).mount("#app");
