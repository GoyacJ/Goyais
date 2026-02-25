<template>
  <div class="app-frame">
    <RouterView />
  </div>
</template>

<style scoped>
.app-frame {
  width: 100%;
  height: 100%;
  border-radius: var(--global-radius-16);
  overflow: hidden;
  background: var(--component-shell-bg);
}

@media (max-width: 768px) {
  .app-frame {
    border-radius: 0;
    overflow: visible;
  }
}
</style>
