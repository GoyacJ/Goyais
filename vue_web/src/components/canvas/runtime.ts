/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import type { ComputedRef, InjectionKey } from 'vue'

export type CanvasStepRuntime = {
  status: string
  durationMs?: number
  logRef?: string
  artifactCount: number
  errorCode?: string
}

export const canvasStepRuntimeByKeyKey: InjectionKey<ComputedRef<Record<string, CanvasStepRuntime>>> = Symbol(
  'canvas-step-runtime-by-key',
)
