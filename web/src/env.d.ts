/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_GOYAIS_API_BASE_URL?: string
  readonly VITE_GOYAIS_DEV_PROXY_TARGET?: string
  readonly VITE_GOYAIS_TENANT_ID?: string
  readonly VITE_GOYAIS_WORKSPACE_ID?: string
  readonly VITE_GOYAIS_USER_ID?: string
  readonly VITE_GOYAIS_ROLES?: string
  readonly VITE_GOYAIS_POLICY_VERSION?: string
  readonly VITE_GOYAIS_USE_MOCK?: string
  readonly VITE_GOYAIS_FEATURE_CANVAS_V2?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
