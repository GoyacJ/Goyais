<template>
  <div class="rounded-xl border border-ui-border bg-ui-panel px-3 py-2 shadow-sm">
    <Handle id="in" type="target" :position="Position.Left" />
    <div class="flex items-center justify-between gap-2">
      <div class="text-sm font-semibold text-ui-text">{{ label }}</div>
      <span class="rounded-md border border-ui-border px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-ui-muted">
        {{ nodeType }}
      </span>
    </div>
    <div class="mt-1 flex items-center gap-2 text-[11px] text-ui-muted">
      <span class="rounded border border-ui-border px-1.5 py-0.5">in: {{ inputType }}</span>
      <span class="rounded border border-ui-border px-1.5 py-0.5">out: {{ outputType }}</span>
    </div>
    <div v-if="runtime" class="canvas-node-runtime mt-2 rounded border border-ui-border px-2 py-1 text-[10px]" :class="runtimeToneClass">
      <div class="flex items-center justify-between gap-1">
        <span class="uppercase tracking-wide">{{ runtime.status }}</span>
        <span v-if="runtimeDuration">{{ runtimeDuration }}</span>
      </div>
      <div class="mt-1 flex items-center justify-between gap-1">
        <span>artifacts</span>
        <span>{{ runtime.artifactCount }}</span>
      </div>
      <div v-if="runtime.errorCode" class="mt-1 text-ui-danger">{{ runtime.errorCode }}</div>
    </div>
    <Handle id="out" type="source" :position="Position.Right" />
  </div>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */
import { canvasStepRuntimeByKeyKey } from '@/components/canvas/runtime'
import { computed, inject } from 'vue'
import { Handle, Position, type NodeProps } from '@vue-flow/core'

type NodeData = {
  label?: string
  inputType?: string
  outputType?: string
  nodeType?: string
}

const props = defineProps<NodeProps<NodeData>>()
const runtimeByKey = inject(canvasStepRuntimeByKeyKey, null)

const label = computed(() => props.data?.label ?? props.id)
const inputType = computed(() => props.data?.inputType ?? 'any')
const outputType = computed(() => props.data?.outputType ?? 'any')
const nodeType = computed(() => props.data?.nodeType ?? props.type ?? 'typed')
const runtime = computed(() => runtimeByKey?.value[props.id] ?? null)
const runtimeDuration = computed(() => {
  if (!runtime.value || typeof runtime.value.durationMs !== 'number') {
    return ''
  }
  return `${runtime.value.durationMs}ms`
})
const runtimeToneClass = computed(() => {
  const status = runtime.value?.status ?? ''
  if (status === 'running' || status === 'pending') {
    return 'text-ui-primary'
  }
  if (status === 'failed' || status === 'canceled') {
    return 'text-ui-danger'
  }
  return 'text-ui-text'
})
</script>
