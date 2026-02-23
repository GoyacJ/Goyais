<template>
  <AccountShell
    active-key="remote_permissions_audit"
    title="权限与审计"
    subtitle="Remote Workspace / Permissions & Audit"
  >
    <p v-if="adminStore.error" class="error">{{ adminStore.error }}</p>

    <section class="card">
      <div class="card-head">
        <h3>角色菜单可见性</h3>
        <div class="toolbar">
          <select v-model="selectedRole" @change="onRoleChange">
            <option v-for="role in roleOptions" :key="role" :value="role">{{ role }}</option>
          </select>
          <button type="button" @click="saveRoleVisibility">保存</button>
        </div>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>菜单 key</th>
            <th>名称</th>
            <th>可见性</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="menu in adminStore.menuNodes" :key="menu.key">
            <td>{{ menu.key }}</td>
            <td>{{ menu.label }}</td>
            <td>
              <select :value="menuVisibility(menu.key)" @change="onMenuVisibilityChange(menu.key, $event)">
                <option value="enabled">enabled</option>
                <option value="readonly">readonly</option>
                <option value="disabled">disabled</option>
                <option value="hidden">hidden</option>
              </select>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="card">
      <div class="card-head">
        <h3>菜单定义</h3>
        <button type="button" @click="addMenuNode">新增菜单</button>
      </div>
      <table class="table">
        <thead>
          <tr>
            <th>菜单 key</th>
            <th>名称</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="menu in adminStore.menuNodes" :key="menu.key">
            <td>{{ menu.key }}</td>
            <td>{{ menu.label }}</td>
            <td><span :class="menu.enabled ? 'enabled' : 'disabled'">{{ menu.enabled ? "enabled" : "disabled" }}</span></td>
            <td>
              <div class="actions">
                <button type="button" @click="toggleMenu(menu.key)"><AppIcon name="info" :size="12" /></button>
                <button type="button" @click="removeMenu(menu.key)"><AppIcon name="trash-2" :size="12" /></button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="card">
      <div class="card-head">
        <h3>权限定义</h3>
        <button type="button" @click="addPermission">新增权限</button>
      </div>
      <table class="table">
        <thead>
          <tr>
            <th>权限 key</th>
            <th>名称</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="permission in adminStore.permissionItems" :key="permission.key">
            <td>{{ permission.key }}</td>
            <td>{{ permission.label }}</td>
            <td>
              <span :class="permission.enabled ? 'enabled' : 'disabled'">{{ permission.enabled ? "enabled" : "disabled" }}</span>
            </td>
            <td>
              <div class="actions">
                <button type="button" @click="togglePermission(permission.key)"><AppIcon name="info" :size="12" /></button>
                <button type="button" @click="removePermission(permission.key)"><AppIcon name="trash-2" :size="12" /></button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="card">
      <div class="card-head">
        <h3>ABAC 策略</h3>
        <button type="button" @click="addPolicy">新增策略</button>
      </div>
      <table class="table">
        <thead>
          <tr>
            <th>策略 ID</th>
            <th>名称</th>
            <th>effect</th>
            <th>priority</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="policy in adminStore.abacPolicies" :key="policy.id">
            <td>{{ policy.id }}</td>
            <td>{{ policy.name }}</td>
            <td>{{ policy.effect }}</td>
            <td>{{ policy.priority }}</td>
            <td><span :class="policy.enabled ? 'enabled' : 'disabled'">{{ policy.enabled ? "enabled" : "disabled" }}</span></td>
            <td>
              <div class="actions">
                <button type="button" @click="togglePolicy(policy.id)"><AppIcon name="info" :size="12" /></button>
                <button type="button" @click="removePolicy(policy.id)"><AppIcon name="trash-2" :size="12" /></button>
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
import { computed, onMounted } from "vue";

import {
  adminStore,
  createOrUpdateABACPolicy,
  createOrUpdateMenuNode,
  createOrUpdatePermission,
  deleteABACPolicy,
  deleteMenuNode,
  deletePermissionItem,
  loadNextAdminAuditsPage,
  loadPreviousAdminAuditsPage,
  refreshAdminData,
  refreshRoleMenuVisibility,
  saveRoleMenuVisibilityForActiveRole,
  setMenuVisibilityItem,
  toggleABACPolicy,
  toggleMenuNode,
  togglePermissionItem
} from "@/modules/admin/store";
import AccountShell from "@/shared/shells/AccountShell.vue";
import AppIcon from "@/shared/ui/AppIcon.vue";
import CursorPager from "@/shared/ui/CursorPager.vue";
import type { PermissionVisibility, Role } from "@/shared/types/api";

const roleOptions: Role[] = ["viewer", "developer", "approver", "admin"];

const selectedRole = computed<Role>({
  get: () => adminStore.activeRoleKey,
  set: (value) => {
    adminStore.activeRoleKey = value;
  }
});

onMounted(async () => {
  await refreshAdminData();
});

function menuVisibility(menuKey: string): PermissionVisibility {
  return adminStore.roleMenuVisibility[menuKey] ?? "enabled";
}

async function onRoleChange(): Promise<void> {
  await refreshRoleMenuVisibility(selectedRole.value);
}

function onMenuVisibilityChange(menuKey: string, event: Event): void {
  const visibility = (event.target as HTMLSelectElement).value as PermissionVisibility;
  setMenuVisibilityItem(menuKey, visibility);
}

async function saveRoleVisibility(): Promise<void> {
  await saveRoleMenuVisibilityForActiveRole();
}

async function addMenuNode(): Promise<void> {
  const key = window.prompt("菜单 key", "workspace_new");
  if (!key) return;
  const label = window.prompt("菜单名称", key);
  if (!label) return;
  await createOrUpdateMenuNode({ key, label, enabled: true });
}

async function toggleMenu(menuKey: string): Promise<void> {
  await toggleMenuNode(menuKey);
}

async function removeMenu(menuKey: string): Promise<void> {
  await deleteMenuNode(menuKey);
}

async function addPermission(): Promise<void> {
  const key = window.prompt("权限 key", "resource.new");
  if (!key) return;
  const label = window.prompt("权限名称", key);
  if (!label) return;
  await createOrUpdatePermission({ key, label, enabled: true });
}

async function togglePermission(permissionKey: string): Promise<void> {
  await togglePermissionItem(permissionKey);
}

async function removePermission(permissionKey: string): Promise<void> {
  await deletePermissionItem(permissionKey);
}

async function addPolicy(): Promise<void> {
  const name = window.prompt("策略名称", "allow custom");
  if (!name) return;
  await createOrUpdateABACPolicy({
    id: "",
    workspace_id: "",
    name,
    effect: "allow",
    priority: 100,
    enabled: true,
    subject_expr: {},
    resource_expr: {},
    action_expr: {},
    context_expr: {}
  });
}

async function togglePolicy(policyId: string): Promise<void> {
  await toggleABACPolicy(policyId);
}

async function removePolicy(policyId: string): Promise<void> {
  await deleteABACPolicy(policyId);
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

<style scoped src="./RemotePermissionsAuditView.css"></style>
