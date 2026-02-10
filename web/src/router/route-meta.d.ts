import 'vue-router'
import type { LayoutMode } from '@/design-system/types'

declare module 'vue-router' {
  interface RouteMeta {
    layoutDefault?: LayoutMode
    windowed?: boolean
    windowManifestKey?: string
  }
}

export {}
