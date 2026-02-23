<template>
  <div v-if="open" class="modal-mask">
    <section class="modal">
      <header class="title">
        <slot name="title" />
      </header>
      <div class="body">
        <slot />
      </div>
      <footer class="footer">
        <slot name="footer" />
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
defineProps<{ open: boolean }>();
</script>

<style scoped>
.modal-mask {
  position: fixed;
  inset: 0;
  background: var(--semantic-overlay);
  display: grid;
  place-items: center;
  z-index: 1000;
}

.modal {
  width: min(720px, 90vw);
  background: var(--component-modal-bg);
  border: 1px solid var(--component-modal-border);
  border-radius: var(--component-modal-radius);
  padding: var(--global-space-16);
  display: grid;
  gap: var(--global-space-12);
}

.title {
  color: var(--component-modal-title-fg);
}

.body {
  color: var(--component-modal-body-fg);
}

.footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--global-space-8);
}
</style>
