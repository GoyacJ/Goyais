<template>
  <aside class="sidebar" data-testid="conversation-sidebar">
    <h3>Sidebar</h3>
    <ul>
      <li>Workspace Placeholder</li>
      <li>Project Placeholder</li>
      <li>Conversation Placeholder</li>
    </ul>
  </aside>
</template>

<style scoped>
.sidebar {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  padding: var(--component-space-sm);
  min-height: 100%;
}

h3 {
  margin: 0 0 var(--component-space-sm) 0;
  font-size: var(--component-font-size-title);
}

ul {
  margin: 0;
  padding-left: var(--component-space-md);
  color: var(--component-text-subtle);
  display: grid;
  gap: var(--component-space-xs);
}
</style>
