<template>
  <section class="ui-card">
    <header v-if="title || subtitle" class="mb-3 border-b border-ui-borderSubtle pb-2">
      <h3 v-if="title" class="ui-title font-semibold">{{ title }}</h3>
      <p v-if="subtitle" class="mt-1 text-sm text-ui-muted">{{ subtitle }}</p>
    </header>
    <div>
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
  subtitle?: string
}>()
</script>
