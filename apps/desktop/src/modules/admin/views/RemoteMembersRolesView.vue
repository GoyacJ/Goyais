<template>
  <AccountShell
    active-key="remote_members_roles"
    title="成员与角色"
    subtitle="Remote Workspace / Members & Roles"
  >
    <section class="card">
      <div class="card-head">
        <h3>成员列表</h3>
        <button type="button" @click="addMember">新增成员</button>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>显示名</th>
            <th>角色</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in adminStore.users" :key="user.id">
            <td>{{ user.username }}</td>
            <td>{{ user.display_name }}</td>
            <td>
              <select :value="user.role" @change="onAssignRole(user.id, $event)">
                <option v-for="role in roleOptions" :key="role" :value="role">{{ role }}</option>
              </select>
            </td>
            <td>
              <span :class="user.enabled ? 'enabled' : 'disabled'">{{ user.enabled ? 'enabled' : 'disabled' }}</span>
            </td>
            <td>
              <div class="actions">
                <button type="button" @click="editMember(user.id)">
                  <AppIcon name="pencil" :size="12" />
                </button>
                <button type="button" @click="toggleMember(user.id)">
                  <AppIcon name="info" :size="12" />
                </button>
                <button type="button" @click="removeMember(user.id)">
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
        <h3>角色列表</h3>
        <button type="button" @click="addRole">新增角色</button>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>角色</th>
            <th>权限</th>
            <th>状态</th>
            <th>动作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="role in adminStore.roles" :key="role.key">
            <td>{{ role.name }}</td>
            <td>
              <div class="permission-tags">
                <span v-for="permission in role.permissions" :key="permission" class="tag">{{ permission }}</span>
              </div>
            </td>
            <td>
              <span :class="role.enabled ? 'enabled' : 'disabled'">{{ role.enabled ? 'enabled' : 'disabled' }}</span>
            </td>
            <td>
              <div class="actions">
                <button type="button" @click="editRole(role.key)">
                  <AppIcon name="pencil" :size="12" />
                </button>
                <button type="button" @click="toggleRoleState(role.key)">
                  <AppIcon name="info" :size="12" />
                </button>
                <button type="button" @click="assignPermission(role.key)">
                  <AppIcon name="plus" :size="12" />
                </button>
                <button type="button" @click="removeRoleByKey(role.key)">
                  <AppIcon name="trash-2" :size="12" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>
  </AccountShell>
</template>

<script setup lang="ts">
import { onMounted } from "vue";

import {
  adminStore,
  assignPermissionToRole,
  assignRoleToUser,
  createOrUpdateAdminUser,
  createOrUpdateRole,
  deleteAdminUser,
  deleteRole,
  refreshAdminData,
  toggleAdminUser,
  toggleRole
} from "@/modules/admin/store";
import AppIcon from "@/shared/ui/AppIcon.vue";
import AccountShell from "@/shared/shells/AccountShell.vue";
import type { Role } from "@/shared/types/api";

const roleOptions: Role[] = ["viewer", "developer", "approver", "admin"];

onMounted(async () => {
  await refreshAdminData();
});

async function addMember(): Promise<void> {
  const username = window.prompt("用户名", "new.user");
  if (!username) {
    return;
  }

  await createOrUpdateAdminUser({
    username,
    display_name: username,
    role: "developer"
  });
}

async function editMember(userId: string): Promise<void> {
  const target = adminStore.users.find((item) => item.id === userId);
  if (!target) {
    return;
  }

  const displayName = window.prompt("显示名", target.display_name);
  if (!displayName) {
    return;
  }

  await createOrUpdateAdminUser({
    username: target.username,
    display_name: displayName,
    role: target.role
  });
}

async function toggleMember(userId: string): Promise<void> {
  await toggleAdminUser(userId);
}

async function removeMember(userId: string): Promise<void> {
  await deleteAdminUser(userId);
}

async function onAssignRole(userId: string, event: Event): Promise<void> {
  const role = (event.target as HTMLSelectElement).value as Role;
  await assignRoleToUser(userId, role);
}

async function addRole(): Promise<void> {
  const keyInput = window.prompt("角色 key(viewer/developer/approver/admin)", "developer");
  if (!keyInput || !roleOptions.includes(keyInput as Role)) {
    return;
  }

  const name = window.prompt("角色名", keyInput);
  if (!name) {
    return;
  }

  await createOrUpdateRole({
    key: keyInput as Role,
    name,
    permissions: ["read", "write", "execute"],
    enabled: true
  });
}

async function editRole(roleKey: Role): Promise<void> {
  const role = adminStore.roles.find((item) => item.key === roleKey);
  if (!role) {
    return;
  }

  const name = window.prompt("角色展示名", role.name);
  if (!name) {
    return;
  }

  await createOrUpdateRole({ ...role, name });
}

async function toggleRoleState(roleKey: Role): Promise<void> {
  await toggleRole(roleKey);
}

function assignPermission(roleKey: Role): void {
  const permission = window.prompt("新增权限", "resource.manage");
  if (!permission) {
    return;
  }
  assignPermissionToRole(roleKey, permission);
}

async function removeRoleByKey(roleKey: Role): Promise<void> {
  await deleteRole(roleKey);
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

.card-head h3 {
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

.table select {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  padding: 4px var(--global-space-8);
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

.permission-tags {
  display: inline-flex;
  gap: var(--global-space-4);
  flex-wrap: wrap;
}

.tag {
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  border: 1px solid var(--semantic-border);
  padding: 2px var(--global-space-8);
  font-size: var(--global-font-size-11);
}

.enabled {
  color: var(--semantic-success);
}

.disabled {
  color: var(--semantic-danger);
}
</style>
