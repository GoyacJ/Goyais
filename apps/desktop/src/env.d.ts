/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_HUB_BASE_URL?: string;
  readonly VITE_APP_VERSION?: string;
  readonly VITE_RUNTIME_TARGET?: "desktop" | "mobile" | "web";
  readonly VITE_REQUIRE_HTTPS_HUB?: string;
  readonly VITE_ALLOW_INSECURE_HUB?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module "*.vue" {
  import type { DefineComponent } from "vue";

  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>;
  export default component;
}
