<template>
  <div class="ui-bg-host ui-bg-stack-console flex h-full bg-ui-bg text-ui-fg">
    <SideNav class="ui-bg-content border-r border-ui-border" />
    <div class="ui-bg-content flex min-w-0 flex-1 flex-col">
      <TopBar />
      <main class="ui-scrollbar min-h-0 flex-1 overflow-auto p-4 lg:p-6">
        <RouterView />
      </main>
    </div>
    <ToastViewport />
  </div>
</template>

<script setup lang="ts">
import SideNav from '@/components/layout/SideNav.vue'
import TopBar from '@/components/layout/TopBar.vue'
import ToastViewport from '@/components/ui/ToastViewport.vue'
import { RouterView } from 'vue-router'
</script>
