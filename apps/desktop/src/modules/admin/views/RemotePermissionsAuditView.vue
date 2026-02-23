<template>
  <AccountShell
    active-key="remote_permissions_audit"
    title="权限与审计"
    subtitle="Remote Workspace / Permissions & Audit"
  >
    <ToastAlert tone="403" message="ABAC 拒绝示例: Forbidden (403)" />

    <section class="card">
      <div class="card-head">
        <h3>菜单树</h3>
        <button type="button" @click="addMenuNode">新增菜单</button>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>菜单 ID</th>
            <th>名称</th>
            <th>可见性</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="menu in adminStore.menuNodes" :key="menu.id">
            <td>{{ menu.id }}</td>
            <td>{{ menu.label }}</td>
            <td>{{ menu.visibility }}</td>
            <td><span :class="menu.enabled ? 'enabled' : 'disabled'">{{ menu.enabled ? 'enabled' : 'disabled' }}</span></td>
            <td>
              <div class="actions">
                <button type="button" @click="toggleMenu(menu.id)">
                  <AppIcon name="info" :size="12" />
                </button>
                <button type="button" @click="removeMenu(menu.id)">
                  <AppIcon name="trash-2" :size="12" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="card">
      <div class="card-head">
        <h3>权限项</h3>
        <button type="button" @click="addPermission">新增权限</button>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>权限 ID</th>
            <th>描述</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="permission in adminStore.permissionItems" :key="permission.id">
            <td>{{ permission.id }}</td>
            <td>{{ permission.label }}</td>
            <td><span :class="permission.enabled ? 'enabled' : 'disabled'">{{ permission.enabled ? 'enabled' : 'disabled' }}</span></td>
            <td>
              <div class="actions">
                <button type="button" @click="togglePermission(permission.id)">
                  <AppIcon name="info" :size="12" />
                </button>
                <button type="button" @click="removePermission(permission.id)">
                  <AppIcon name="trash-2" :size="12" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="card">
      <h3>审计日志</h3>
      <table class="table">
        <thead>
          <tr>
            <th>actor</th>
            <th>action</th>
            <th>resource</th>
            <th>result</th>
            <th>time</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="audit in adminStore.audits" :key="audit.id">
            <td>{{ audit.actor }}</td>
            <td>{{ audit.action }}</td>
            <td>{{ audit.resource }}</td>
            <td :class="audit.result === 'success' ? 'enabled' : 'disabled'">{{ audit.result }}</td>
            <td>{{ formatTime(audit.timestamp) }}</td>
          </tr>
        </tbody>
      </table>
      <CursorPager
        :can-prev="adminStore.auditsPage.backStack.length > 0"
        :can-next="adminStore.auditsPage.nextCursor !== null"
        :loading="adminStore.auditsPage.loading"
        @prev="paginateAudits('prev')"
        @next="paginateAudits('next')"
      />
    </section>
  </AccountShell>
</template>

<script setup lang="ts">
import { onMounted } from "vue";

import {
  adminStore,
  deleteMenuNode,
  deletePermissionItem,
  loadNextAdminAuditsPage,
  loadPreviousAdminAuditsPage,
  refreshAdminData,
  toggleMenuNode,
  togglePermissionItem
} from "@/modules/admin/store";
import AccountShell from "@/shared/shells/AccountShell.vue";
import AppIcon from "@/shared/ui/AppIcon.vue";
import CursorPager from "@/shared/ui/CursorPager.vue";
import ToastAlert from "@/shared/ui/ToastAlert.vue";

onMounted(async () => {
  await refreshAdminData();
});

function addMenuNode(): void {
  const menuId = window.prompt("菜单 ID", "workspace_new");
  if (!menuId) {
    return;
  }

  adminStore.menuNodes.unshift({
    id: menuId,
    label: `菜单 ${menuId}`,
    visibility: "enabled",
    enabled: true
  });
}

function toggleMenu(menuId: string): void {
  toggleMenuNode(menuId);
}

function removeMenu(menuId: string): void {
  deleteMenuNode(menuId);
}

function addPermission(): void {
  const permissionId = window.prompt("权限 ID", "resource.new");
  if (!permissionId) {
    return;
  }

  adminStore.permissionItems.unshift({
    id: permissionId,
    label: `权限 ${permissionId}`,
    enabled: true
  });
}

function togglePermission(permissionId: string): void {
  togglePermissionItem(permissionId);
}

function removePermission(permissionId: string): void {
  deletePermissionItem(permissionId);
}

function formatTime(value: string): string {
  return new Date(value).toLocaleString();
}

async function paginateAudits(direction: "prev" | "next"): Promise<void> {
  if (direction === "next") {
    await loadNextAdminAuditsPage();
    return;
  }
  await loadPreviousAdminAuditsPage();
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

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--global-space-8);
}

.card-head h3,
.card h3 {
  margin: 0;
}

.card-head button {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  padding: var(--global-space-8) var(--global-space-12);
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th,
.table td {
  border-bottom: 1px solid var(--semantic-divider);
  padding: var(--global-space-8);
  text-align: left;
  vertical-align: top;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.table th {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}

.actions {
  display: inline-flex;
  gap: var(--global-space-4);
}

.actions button {
  border: 0;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.enabled {
  color: var(--semantic-success);
}

.disabled {
  color: var(--semantic-danger);
}
</style>
