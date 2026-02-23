<template>
  <section class="ui-card">
    <header v-if="$slots.header" class="card-header">
      <slot name="header" />
    </header>
    <div class="card-body">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="card-footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<style scoped>
.ui-card {
  background: var(--component-card-bg);
  border: 1px solid var(--component-card-border);
  border-radius: var(--component-card-radius);
  padding: var(--component-card-padding);
  display: grid;
  gap: var(--global-space-12);
}

.card-header {
  color: var(--component-card-header-fg);
}

.card-body {
  color: var(--component-card-body-fg);
}

.card-footer {
  background: var(--component-card-footer-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
}
</style>
