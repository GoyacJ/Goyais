<template>
  <header class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <h2 class="ui-title text-xl font-semibold leading-tight">{{ title }}</h2>
      <p v-if="subtitle" class="mt-1 text-sm text-ui-muted">{{ subtitle }}</p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>
