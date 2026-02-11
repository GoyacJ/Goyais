/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

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
