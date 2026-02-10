<template>
  <div class="ui-card border-dashed text-center">
    <p class="text-sm text-ui-muted">{{ description }}</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  description: string
}>()
</script>
