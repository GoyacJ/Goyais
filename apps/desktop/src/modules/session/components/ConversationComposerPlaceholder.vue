<template>
  <section
    class="composer grid gap-[var(--component-space-sm)] border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] p-[var(--component-space-md)]"
    data-testid="conversation-composer"
  >
    <h3 class="m-0 text-[var(--component-font-size-title)]">Input Composer</h3>
    <textarea
      disabled
      placeholder="输入区占位"
      class="min-h-[96px] resize-y border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] p-[var(--component-space-sm)] text-[var(--component-text-main)] [font:inherit]"
    />
    <div class="actions flex gap-[var(--component-space-sm)]">
      <button
        type="button"
        disabled
        class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)]"
      >
        Send
      </button>
      <button
        type="button"
        disabled
        data-testid="conversation-stop-button"
        class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)]"
      >
        Stop
      </button>
    </div>
  </section>
</template>
