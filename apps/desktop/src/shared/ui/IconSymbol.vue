<template>
  <span class="[font-family:var(--global-font-family-icon)] leading-none inline-flex items-center justify-center select-none" :style="iconStyle" aria-hidden="true">{{ name }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    name: string;
    size?: number;
    weight?: number;
  }>(),
  {
    size: 16,
    weight: 400
  }
);

const iconStyle = computed(() => ({
  fontSize: `${props.size}px`,
  fontVariationSettings: `'FILL' 0, 'wght' ${props.weight}, 'GRAD' 0, 'opsz' ${props.size}`
}));
</script>
