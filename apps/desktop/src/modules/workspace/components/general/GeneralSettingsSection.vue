<template>
  <section class="section grid gap-[var(--global-space-8)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-12)]">
    <header class="section-header grid gap-[var(--global-space-8)]">
      <h3 class="m-0">{{ title }}</h3>
      <p v-if="description !== ''" class="m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ description }}</p>
    </header>

    <div class="section-body grid gap-[var(--global-space-8)]">
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
