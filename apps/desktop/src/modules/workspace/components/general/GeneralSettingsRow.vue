<template>
  <article class="row">
    <div class="meta">
      <p class="label">{{ label }}</p>
      <p v-if="description !== ''" class="description">{{ description }}</p>
      <p v-if="status !== ''" class="status">{{ status }}</p>
      <p v-if="unsupportedReason !== ''" class="unsupported">{{ unsupportedReason }}</p>
    </div>

    <div class="control">
      <slot />
      <p v-if="hint !== ''" class="hint">{{ hint }}</p>
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

<style scoped>
.row {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(240px, 360px);
  gap: var(--global-space-12);
  align-items: start;
  padding: var(--global-space-8) 0;
  border-top: 1px solid var(--semantic-divider);
}

.row:first-child {
  border-top: 0;
  padding-top: 0;
}

.meta,
.control {
  display: grid;
  gap: var(--global-space-4);
}

.label,
.description,
.status,
.hint,
.unsupported {
  margin: 0;
}

.label {
  color: var(--semantic-text);
  font-size: var(--global-font-size-13);
  font-weight: var(--global-font-weight-600);
}

.description,
.hint,
.status {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.unsupported {
  color: var(--semantic-warning);
  font-size: var(--global-font-size-12);
}

@media (max-width: 1180px) {
  .row {
    grid-template-columns: 1fr;
  }
}
</style>
