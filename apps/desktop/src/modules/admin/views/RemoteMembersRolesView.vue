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
      <CursorPager
        :can-prev="adminStore.usersPage.backStack.length > 0"
        :can-next="adminStore.usersPage.nextCursor !== null"
        :loading="adminStore.usersPage.loading"
        @prev="paginateUsers('prev')"
        @next="paginateUsers('next')"
      />
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
import { watch } from "vue";

import {
  adminStore,
  assignPermissionToRole,
  assignRoleToUser,
  createOrUpdateAdminUser,
  createOrUpdateRole,
  deleteAdminUser,
  deleteRole,
  loadNextAdminUsersPage,
  loadPreviousAdminUsersPage,
  refreshAdminData,
  toggleAdminUser,
  toggleRole
} from "@/modules/admin/store";
import AppIcon from "@/shared/ui/AppIcon.vue";
import AccountShell from "@/shared/shells/AccountShell.vue";
import CursorPager from "@/shared/ui/CursorPager.vue";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { Role } from "@/shared/types/api";

const roleOptions: Role[] = ["viewer", "developer", "approver", "admin"];

watch(
  () => workspaceStore.currentWorkspaceId,
  async () => {
    await refreshAdminData();
  },
  { immediate: true }
);

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
    permissions: ["project.read", "project.write", "conversation.read", "conversation.write"],
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

async function assignPermission(roleKey: Role): Promise<void> {
  const permission = window.prompt("新增权限", "resource.manage");
  if (!permission) {
    return;
  }
  await assignPermissionToRole(roleKey, permission);
}

async function removeRoleByKey(roleKey: Role): Promise<void> {
  await deleteRole(roleKey);
}

async function paginateUsers(direction: "prev" | "next"): Promise<void> {
  if (direction === "next") {
    await loadNextAdminUsersPage();
    return;
  }
  await loadPreviousAdminUsersPage();
}
</script>

<style scoped src="./RemoteMembersRolesView.css"></style>
