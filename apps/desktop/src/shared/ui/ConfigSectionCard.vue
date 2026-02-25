<template>
  <BaseCard>
    <template #header>
      <strong>{{ title }}</strong>
    </template>

    <div class="grid gap-[var(--global-space-8)]">
      <p
        v-for="line in lines"
        :key="line"
        class="m-0 text-[var(--global-font-size-12)] leading-[var(--global-line-height-normal)] text-[var(--semantic-text-muted)]"
        :class="[toneClassMap[tone], { '[font-family:var(--global-font-family-code)] whitespace-pre-wrap': mono }]"
      >
        {{ line }}
      </p>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import BaseCard from "@/shared/ui/BaseCard.vue";

withDefaults(
  defineProps<{
    title: string;
    lines: string[];
    tone?: "default" | "info" | "warning" | "danger" | "success";
    mono?: boolean;
  }>(),
  {
    tone: "default",
    mono: false
  }
);

const toneClassMap = {
  default: "",
  info: "text-[var(--component-toast-info-fg)]",
  warning: "text-[var(--component-toast-warning-fg)]",
  danger: "text-[var(--semantic-danger)]",
  success: "text-[var(--semantic-success)]"
} as const;
</script>
