<template>
  <div
    v-if="open"
    class="modal-mask fixed inset-0 z-[1000] grid place-items-center bg-[var(--semantic-overlay)]"
    @click.self="onBackdropClick"
  >
    <section
      ref="dialogRef"
      class="modal grid w-[min(720px,90vw)] gap-[var(--global-space-12)] border border-[var(--component-modal-border)] rounded-[var(--component-modal-radius)] bg-[var(--component-modal-bg)] p-[var(--global-space-16)]"
      :class="panelClass"
      role="dialog"
      aria-modal="true"
      :aria-label="ariaLabel"
      :aria-labelledby="titleId"
      tabindex="-1"
      @keydown="onDialogKeydown"
    >
      <header :id="titleId" class="title text-[var(--component-modal-title-fg)]">
        <slot name="title" />
      </header>
      <div class="body text-[var(--component-modal-body-fg)]">
        <slot />
      </div>
      <footer class="footer flex justify-end gap-[var(--global-space-8)]">
        <slot name="footer" />
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    open: boolean;
    ariaLabel?: string;
    closeOnEsc?: boolean;
    closeOnBackdrop?: boolean;
    initialFocusSelector?: string;
    panelClass?: string;
  }>(),
  {
    ariaLabel: undefined,
    closeOnEsc: true,
    closeOnBackdrop: false,
    initialFocusSelector: undefined,
    panelClass: ""
  }
);

const emit = defineEmits<{
  (event: "close"): void;
}>();

const dialogRef = ref<HTMLElement | null>(null);
const titleId = `base-modal-title-${Math.random().toString(36).slice(2, 10)}`;
let restoreTarget: HTMLElement | null = null;

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])"
].join(",");

watch(
  () => props.open,
  async (open) => {
    if (open) {
      restoreTarget = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      await nextTick();
      focusInitialElement();
      return;
    }

    await nextTick();
    restoreFocus();
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  restoreFocus();
});

function onBackdropClick(): void {
  if (props.closeOnBackdrop) {
    emit("close");
  }
}

function onDialogKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape" && props.closeOnEsc) {
    event.preventDefault();
    emit("close");
    return;
  }

  if (event.key === "Tab") {
    trapFocus(event);
  }
}

function focusInitialElement(): void {
  const dialog = dialogRef.value;
  if (!dialog) {
    return;
  }

  if (props.initialFocusSelector) {
    const target = dialog.querySelector<HTMLElement>(props.initialFocusSelector);
    if (target) {
      target.focus();
      return;
    }
  }

  const focusables = getFocusableElements(dialog);
  const firstFocusable = focusables[0];
  if (firstFocusable) {
    firstFocusable.focus();
    return;
  }

  dialog.focus();
}

function trapFocus(event: KeyboardEvent): void {
  const dialog = dialogRef.value;
  if (!dialog) {
    return;
  }

  const focusables = getFocusableElements(dialog);
  if (focusables.length === 0) {
    event.preventDefault();
    dialog.focus();
    return;
  }

  const first = focusables[0];
  const last = focusables[focusables.length - 1];
  const active = document.activeElement instanceof HTMLElement ? document.activeElement : null;

  if (event.shiftKey) {
    if (active === first || active === dialog || !dialog.contains(active)) {
      event.preventDefault();
      last?.focus();
    }
    return;
  }

  if (active === last || !dialog.contains(active)) {
    event.preventDefault();
    first?.focus();
  }
}

function getFocusableElements(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter((item) => {
    if (item.getAttribute("aria-hidden") === "true") {
      return false;
    }
    if (item.hasAttribute("hidden")) {
      return false;
    }
    return true;
  });
}

function restoreFocus(): void {
  if (!restoreTarget) {
    return;
  }

  if (restoreTarget.isConnected) {
    restoreTarget.focus();
  }

  restoreTarget = null;
}
</script>
