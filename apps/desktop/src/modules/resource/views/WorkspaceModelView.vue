<template>
  <WorkspaceSharedShell
    active-key="workspace_model"
    title="模型配置（共享）"
    account-subtitle="Workspace Config / Models (Shared)"
    settings-subtitle="Local Settings / Models (Shared)"
  >
    <section class="card">
      <div class="card-head">
        <h3>模型目录同步</h3>
        <button type="button" :disabled="resourceStore.modelCatalogSyncing" @click="syncCatalog">
          {{ resourceStore.modelCatalogSyncing ? '同步中...' : '手动同步' }}
        </button>
      </div>
      <p>支持厂商：{{ vendors.join(' / ') }}</p>
    </section>

    <section class="card">
      <h3>Vendor -> Models</h3>
      <div class="vendor-grid">
        <article v-for="vendor in vendors" :key="vendor" class="vendor-card">
          <h4>{{ vendor }}</h4>
          <ul>
            <li v-for="item in modelsByVendor[vendor] ?? []" :key="item.model_id">
              <span>{{ item.model_id }}</span>
              <span :class="item.enabled ? 'enabled' : 'disabled'">{{ item.enabled ? 'enabled' : 'disabled' }}</span>
            </li>
            <li v-if="(modelsByVendor[vendor] ?? []).length === 0" class="muted">暂无模型</li>
          </ul>
        </article>
      </div>
    </section>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";

import { refreshModelCatalog, resourceStore, syncWorkspaceModelCatalog } from "@/modules/resource/store";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import type { ModelVendorName } from "@/shared/types/api";

const vendors: ModelVendorName[] = ["OpenAI", "Google", "Qwen", "Doubao", "Zhipu", "MiniMax", "Local"];

const modelsByVendor = computed<Record<ModelVendorName, typeof resourceStore.modelCatalog>>(() => {
  const grouped = {
    OpenAI: [],
    Google: [],
    Qwen: [],
    Doubao: [],
    Zhipu: [],
    MiniMax: [],
    Local: []
  } as Record<ModelVendorName, typeof resourceStore.modelCatalog>;

  for (const item of resourceStore.modelCatalog) {
    grouped[item.vendor].push(item);
  }

  return grouped;
});

onMounted(async () => {
  await refreshModelCatalog();
});

async function syncCatalog(): Promise<void> {
  await syncWorkspaceModelCatalog(vendors);
}
</script>

<style scoped>
.card {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

.card h3,
.card h4,
.card p {
  margin: 0;
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--global-space-8);
}

.card-head button {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  padding: var(--global-space-8) var(--global-space-12);
}

.vendor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--global-space-12);
}

.vendor-card {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-8);
}

.vendor-card ul {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: var(--global-space-4);
}

.vendor-card li {
  display: flex;
  justify-content: space-between;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.enabled {
  color: var(--semantic-success);
}

.disabled {
  color: var(--semantic-danger);
}

.muted {
  color: var(--semantic-text-subtle);
}
</style>
