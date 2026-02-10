<template>
  <Menu as="div" class="relative inline-block text-left">
    <MenuButton :class="buttonClasses">
      <slot name="trigger">
        <span>{{ label }}</span>
      </slot>
    </MenuButton>

    <transition
      enter-active-class="transition duration-120 ease-out"
      enter-from-class="scale-95 opacity-0"
      enter-to-class="scale-100 opacity-100"
      leave-active-class="transition duration-90 ease-in"
      leave-from-class="scale-100 opacity-100"
      leave-to-class="scale-95 opacity-0"
    >
      <MenuItems class="ui-overlay-panel absolute right-0 z-40 mt-1 w-56 origin-top-right p-1">
        <MenuItem
          v-for="item in items"
          :key="item.value"
          :disabled="item.disabled"
          as="template"
          v-slot="{ active, disabled: itemDisabled }"
        >
          <button
            type="button"
            class="ui-focus-ring ui-pressable flex w-full items-center justify-between rounded-button px-2 py-2 text-sm"
            :class="[
              active ? 'bg-ui-hover' : '',
              item.danger ? 'text-error' : 'text-ui-fg',
              itemDisabled ? 'ui-disabled' : '',
            ]"
            @click="emit('select', item.value)"
          >
            <span>{{ item.label }}</span>
            <span v-if="item.hint" class="ui-monospace text-[11px] text-ui-muted">{{ item.hint }}</span>
          </button>
        </MenuItem>
      </MenuItems>
    </transition>
  </Menu>
</template>

<script setup lang="ts">
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { computed } from 'vue'
import { cn } from '@/utils/cn'

export interface DropdownItem {
  label: string
  value: string
  hint?: string
  disabled?: boolean
  danger?: boolean
}

const props = withDefaults(
  defineProps<{
    label?: string
    items: DropdownItem[]
    disabled?: boolean
  }>(),
  {
    label: 'Actions',
    disabled: false,
  },
)

const emit = defineEmits<{
  (e: 'select', value: string): void
}>()

const buttonClasses = computed(() =>
  cn('ui-control ui-focus-ring ui-pressable inline-flex items-center gap-2 text-sm', props.disabled && 'ui-disabled'),
)
</script>
