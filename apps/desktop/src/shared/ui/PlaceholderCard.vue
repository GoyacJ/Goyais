<template>
  <section class="placeholder-card">
    <h3>{{ title }}</h3>
    <p>{{ description }}</p>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description: string;
}>();
</script>

<style scoped>
.placeholder-card {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  padding: var(--component-space-md);
  display: flex;
  flex-direction: column;
  gap: var(--component-space-xs);
}

h3 {
  margin: 0;
  font-size: var(--component-font-size-title);
}

p {
  margin: 0;
  color: var(--component-text-subtle);
}
</style>
