<template>
  <div class="screen">
    <slot name="sidebar" />

    <section class="content">
      <slot name="header" />
      <slot name="main" />
      <slot name="footer" />
    </section>
  </div>
</template>

<style scoped>
.screen {
  height: 100vh;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--global-space-8);
  padding: 0;
  background: var(--component-shell-bg);
}

.content {
  padding: 0 var(--global-space-8) 0 0;
  display: grid;
  grid-template-rows: 40px 1fr 36px;
  gap: var(--global-space-8);
  border-radius: var(--global-radius-12);
}
</style>
