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
