<template>
  <section
    class="flex flex-col gap-[var(--component-space-xs)] border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] p-[var(--component-space-md)]"
  >
    <h3 class="m-0 text-[var(--component-font-size-title)]">{{ title }}</h3>
    <p class="m-0 text-[var(--component-text-subtle)]">{{ description }}</p>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description: string;
}>();
</script>
