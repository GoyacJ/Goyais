<template>
  <section class="composer" data-testid="conversation-composer">
    <h3>Input Composer</h3>
    <textarea disabled placeholder="输入区占位" />
    <div class="actions">
      <button type="button" disabled>Send</button>
      <button type="button" disabled data-testid="conversation-stop-button">Stop</button>
    </div>
  </section>
</template>

<style scoped>
.composer {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  padding: var(--component-space-md);
  display: grid;
  gap: var(--component-space-sm);
}

h3 {
  margin: 0;
  font-size: var(--component-font-size-title);
}

textarea {
  min-height: 96px;
  resize: vertical;
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  padding: var(--component-space-sm);
  font: inherit;
  color: var(--component-text-main);
  background: var(--component-panel-bg);
}

.actions {
  display: flex;
  gap: var(--component-space-sm);
}

button {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  padding: var(--component-space-xs) var(--component-space-sm);
  color: var(--component-text-main);
}
</style>
