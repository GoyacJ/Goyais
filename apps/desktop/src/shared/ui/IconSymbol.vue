<template>
  <span class="icon-symbol" :style="iconStyle" aria-hidden="true">{{ name }}</span>
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

<style scoped>
.icon-symbol {
  font-family: "Material Symbols Outlined", "SF Pro Text", "PingFang SC", sans-serif;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  user-select: none;
}
</style>
