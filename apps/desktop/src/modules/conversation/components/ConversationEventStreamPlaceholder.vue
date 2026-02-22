<template>
  <section class="events" data-testid="conversation-events">
    <h3>Event Stream</h3>
    <p>执行事件流区域占位（thinking/tool/diff/confirm/result）</p>
  </section>
</template>

<style scoped>
.events {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  padding: var(--component-space-md);
  min-height: 320px;
}

h3 {
  margin: 0 0 var(--component-space-sm) 0;
  font-size: var(--component-font-size-title);
}

p {
  margin: 0;
  color: var(--component-text-subtle);
}
</style>
