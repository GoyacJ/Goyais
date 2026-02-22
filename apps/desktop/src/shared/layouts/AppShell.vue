<template>
  <div class="app-shell">
    <header class="app-header">
      <strong>Goyais v0.4.0</strong>
      <nav>
        <RouterLink to="/workspace">Workspace</RouterLink>
        <RouterLink to="/project">Project</RouterLink>
        <RouterLink to="/conversation">Conversation</RouterLink>
        <RouterLink to="/resource">Resource</RouterLink>
        <RouterLink v-if="showAdminLink" to="/admin">Admin</RouterLink>
      </nav>
    </header>
    <main>
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { canAccessAdmin } from "@/shared/stores/authStore";

const showAdminLink = computed(() => canAccessAdmin());
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  display: grid;
  grid-template-rows: auto 1fr;
  background: var(--component-shell-bg);
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--component-space-md);
  border-bottom: 1px solid var(--component-panel-border);
  background: var(--component-panel-bg);
}

nav {
  display: flex;
  gap: var(--component-space-sm);
  color: var(--component-text-subtle);
}

main {
  padding: var(--component-space-md);
}
</style>
