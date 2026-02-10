<template>
  <div class="ui-surface inline-flex gap-1 p-1" role="tablist" :aria-label="ariaLabel">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.id"
      :aria-disabled="tab.disabled || undefined"
      :class="[
        'ui-control ui-focus-ring ui-pressable min-w-20 border-transparent px-3 text-sm',
        modelValue === tab.id ? 'ui-tab-active' : '',
        tab.disabled ? 'ui-disabled' : '',
      ]"
      :disabled="tab.disabled"
      @click="emit('update:modelValue', tab.id)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
export interface TabItem {
  id: string
  label: string
  disabled?: boolean
}

withDefaults(
  defineProps<{
    tabs: TabItem[]
    modelValue: string
    ariaLabel?: string
  }>(),
  {
    ariaLabel: 'tabs',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()
</script>
