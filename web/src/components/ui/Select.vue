<template>
  <Listbox :model-value="modelValue" :disabled="disabled || loading" @update:model-value="onUpdate">
    <div class="relative">
      <ListboxButton :class="buttonClasses" :aria-busy="loading || undefined">
        <span class="truncate text-sm">{{ selectedLabel }}</span>
        <span class="ui-monospace text-xs text-ui-muted">v</span>
      </ListboxButton>

      <transition
        enter-active-class="transition duration-120 ease-out"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition duration-90 ease-in"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-95"
      >
        <ListboxOptions class="ui-overlay-panel absolute z-30 mt-1 max-h-60 w-full overflow-auto p-1">
          <ListboxOption
            v-for="option in options"
            :key="String(option.value)"
            :value="option.value"
            as="template"
            v-slot="{ active, selected }"
          >
            <li
              :class="[
                'ui-focus-ring ui-pressable cursor-pointer rounded-button px-2 py-2 text-sm',
                active ? 'bg-ui-hover' : '',
                selected ? 'text-primary-700 dark:text-primary-500' : 'text-ui-fg',
              ]"
            >
              {{ option.label }}
            </li>
          </ListboxOption>
        </ListboxOptions>
      </transition>
    </div>
  </Listbox>
</template>

<script setup lang="ts">
import { Listbox, ListboxButton, ListboxOption, ListboxOptions } from '@headlessui/vue'
import { computed } from 'vue'
import { cn } from '@/utils/cn'

export interface SelectOption {
  label: string
  value: string
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    options: SelectOption[]
    disabled?: boolean
    loading?: boolean
  }>(),
  {
    disabled: false,
    loading: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

function onUpdate(value: string): void {
  emit('update:modelValue', value)
}

const selectedLabel = computed(() => {
  const target = props.options.find((item) => item.value === props.modelValue)
  return target?.label ?? props.modelValue
})

const buttonClasses = computed(() =>
  cn(
    'ui-control ui-focus-ring ui-pressable flex w-full items-center justify-between text-left',
    props.disabled && 'ui-disabled',
    props.loading && 'ui-loading',
  ),
)
</script>
