<template>
  <header class="topbar">
    <div class="left">
      <slot name="left" />
    </div>
    <div class="right">
      <slot name="right" />
    </div>
  </header>
</template>

<style scoped>
.topbar {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: 1px solid var(--component-topbar-border);
  background: var(--component-topbar-bg);
  border-radius: var(--global-radius-12);
  padding: 0 var(--global-space-12);
}

.left,
.right {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}
</style>
