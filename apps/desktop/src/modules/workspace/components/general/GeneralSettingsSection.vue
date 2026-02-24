<template>
  <section class="section">
    <header class="section-header">
      <h3>{{ title }}</h3>
      <p v-if="description !== ''">{{ description }}</p>
    </header>

    <div class="section-body">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description?: string;
  }>(),
  {
    description: ""
  }
);
</script>

<style scoped>
.section {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

.section-header,
.section-body {
  display: grid;
  gap: var(--global-space-8);
}

.section-header h3,
.section-header p {
  margin: 0;
}

.section-header p {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}
</style>
