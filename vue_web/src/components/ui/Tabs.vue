<template>
  <div class="ui-surface inline-flex gap-1 p-1" role="tablist" :aria-label="ariaLabel" :aria-orientation="orientation">
    <button
      v-for="(tab, index) in tabs"
      :key="tab.id"
      :ref="(el) => setTabRef(el, index)"
      type="button"
      role="tab"
      :id="tabDomId(tab.id)"
      :aria-selected="modelValue === tab.id"
      :aria-controls="tab.panelId || undefined"
      :aria-disabled="tab.disabled || undefined"
      :tabindex="resolveTabIndex(tab)"
      :class="[
        'ui-control ui-focus-ring ui-pressable min-w-20 border-transparent px-3 text-sm',
        modelValue === tab.id ? 'ui-tab-active' : '',
        tab.disabled ? 'ui-disabled' : '',
      ]"
      :disabled="tab.disabled"
      @click="emit('update:modelValue', tab.id)"
      @keydown="onKeydown($event, index)"
    >
      {{ tab.label }}
    </button>
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
import { nextTick, ref, type ComponentPublicInstance } from 'vue'

export interface TabItem {
  id: string
  label: string
  disabled?: boolean
  panelId?: string
}

const props = withDefaults(
  defineProps<{
    tabs: TabItem[]
    modelValue: string
    ariaLabel?: string
    idBase?: string
    orientation?: 'horizontal' | 'vertical'
  }>(),
  {
    ariaLabel: 'tabs',
    idBase: 'ui-tabs',
    orientation: 'horizontal',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const tabRefs = ref<Array<HTMLButtonElement | null>>([])

function setTabRef(element: Element | ComponentPublicInstance | null, index: number): void {
  if (!element) {
    tabRefs.value[index] = null
    return
  }

  if (element instanceof HTMLButtonElement) {
    tabRefs.value[index] = element
    return
  }

  const maybeElement = '$el' in element ? element.$el : null
  tabRefs.value[index] = maybeElement instanceof HTMLButtonElement ? maybeElement : null
}

function tabDomId(id: string): string {
  return `${props.idBase}-${id}`
}

function resolveTabIndex(tab: TabItem): number {
  if (tab.disabled) {
    return -1
  }

  return props.modelValue === tab.id ? 0 : -1
}

function firstEnabledIndex(): number {
  return props.tabs.findIndex((item) => !item.disabled)
}

function lastEnabledIndex(): number {
  for (let index = props.tabs.length - 1; index >= 0; index -= 1) {
    if (!props.tabs[index]?.disabled) {
      return index
    }
  }
  return -1
}

function nextEnabledIndex(startIndex: number, direction: 1 | -1): number {
  if (props.tabs.length === 0) {
    return -1
  }

  let index = startIndex
  for (let attempt = 0; attempt < props.tabs.length; attempt += 1) {
    index = (index + direction + props.tabs.length) % props.tabs.length
    if (!props.tabs[index]?.disabled) {
      return index
    }
  }

  return -1
}

function focusAndSelect(index: number): void {
  const target = props.tabs[index]
  if (!target || target.disabled) {
    return
  }

  emit('update:modelValue', target.id)
  nextTick(() => {
    tabRefs.value[index]?.focus()
  })
}

function onKeydown(event: KeyboardEvent, index: number): void {
  const current = props.tabs[index]
  if (!current || current.disabled) {
    return
  }

  const horizontal = props.orientation === 'horizontal'
  if (event.key === 'Home') {
    event.preventDefault()
    focusAndSelect(firstEnabledIndex())
    return
  }

  if (event.key === 'End') {
    event.preventDefault()
    focusAndSelect(lastEnabledIndex())
    return
  }

  if (horizontal && event.key === 'ArrowRight') {
    event.preventDefault()
    focusAndSelect(nextEnabledIndex(index, 1))
    return
  }

  if (horizontal && event.key === 'ArrowLeft') {
    event.preventDefault()
    focusAndSelect(nextEnabledIndex(index, -1))
    return
  }

  if (!horizontal && event.key === 'ArrowDown') {
    event.preventDefault()
    focusAndSelect(nextEnabledIndex(index, 1))
    return
  }

  if (!horizontal && event.key === 'ArrowUp') {
    event.preventDefault()
    focusAndSelect(nextEnabledIndex(index, -1))
  }
}
</script>
