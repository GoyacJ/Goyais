<template>
  <TransitionRoot :show="open" as="template">
    <Dialog as="div" class="relative z-50" @close="emit('close')">
      <TransitionChild
        as="template"
        enter="transition-opacity duration-120"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="transition-opacity duration-90"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="ui-overlay-backdrop fixed inset-0" aria-hidden="true" />
      </TransitionChild>

      <div class="fixed inset-0 flex items-center justify-center p-4">
        <TransitionChild
          as="template"
          enter="transition duration-120"
          enter-from="scale-95 opacity-0"
          enter-to="scale-100 opacity-100"
          leave="transition duration-90"
          leave-from="scale-100 opacity-100"
          leave-to="scale-95 opacity-0"
        >
          <DialogPanel class="ui-overlay-panel w-full max-w-lg p-5">
            <DialogTitle class="text-lg font-semibold">{{ title }}</DialogTitle>
            <DialogDescription v-if="description" class="mt-2 text-sm text-ui-muted">
              {{ description }}
            </DialogDescription>

            <div class="mt-4">
              <slot />
            </div>

            <footer v-if="showFooter" class="mt-5 flex items-center justify-end gap-2">
              <Button variant="ghost" @click="emit('close')">{{ cancelLabel }}</Button>
              <Button :loading="confirmLoading" @click="emit('confirm')">{{ confirmLabel }}</Button>
            </footer>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import {
  Dialog,
  DialogDescription,
  DialogPanel,
  DialogTitle,
  TransitionChild,
  TransitionRoot,
} from '@headlessui/vue'
import Button from '@/components/ui/Button.vue'

withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    cancelLabel?: string
    confirmLabel?: string
    confirmLoading?: boolean
    showFooter?: boolean
  }>(),
  {
    description: '',
    cancelLabel: 'Cancel',
    confirmLabel: 'Confirm',
    confirmLoading: false,
    showFooter: true,
  },
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'confirm'): void
}>()
</script>
