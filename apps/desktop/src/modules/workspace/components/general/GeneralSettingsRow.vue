<template>
  <article
    class="row grid items-start gap-[var(--global-space-12)] border-t border-[var(--semantic-divider)] py-[var(--global-space-8)] first:(border-t-0 pt-0) [grid-template-columns:minmax(220px,1fr)_minmax(240px,360px)] max-[1180px]:[grid-template-columns:minmax(0,1fr)]"
  >
    <div class="meta grid gap-[var(--global-space-4)]">
      <p class="label m-0 text-[var(--global-font-size-13)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">{{ label }}</p>
      <p v-if="description !== ''" class="description m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ description }}</p>
      <p v-if="status !== ''" class="status m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ status }}</p>
      <p v-if="unsupportedReason !== ''" class="unsupported m-0 text-[var(--global-font-size-12)] text-[var(--semantic-warning)]">{{ unsupportedReason }}</p>
    </div>

    <div class="control grid gap-[var(--global-space-4)]">
      <slot />
      <p v-if="hint !== ''" class="hint m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ hint }}</p>
    </div>
  </article>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string;
    description?: string;
    status?: string;
    hint?: string;
    unsupportedReason?: string;
  }>(),
  {
    description: "",
    status: "",
    hint: "",
    unsupportedReason: ""
  }
);
</script>
