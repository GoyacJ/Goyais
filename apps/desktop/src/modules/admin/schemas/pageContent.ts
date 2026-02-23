export const remoteAccountCards = [
  {
    title: "账号信息 Account",
    lines: [
      "账号名: goya.admin@remote-hub",
      "邮箱: goya.admin@company.com",
      "角色: admin / approver",
      "Token 到期: 2026-03-15 09:30 UTC"
    ],
    tone: "default" as const
  },
  {
    title: "工作区信息 Workspace",
    lines: [
      "workspace_id: ws-remote-shanghai-01",
      "workspace_name: Production Hub",
      "mode: remote / tenant: t-goyais-cn",
      "hub: https://hub.goyais.example.com"
    ],
    tone: "default" as const
  },
  {
    title: "连接与会话状态",
    lines: [
      "连接状态: connected",
      "活跃 Conversation: 4 · 队列中: 7",
      "reconnecting/disconnected 时将触发只读与重试提示"
    ],
    tone: "success" as const
  }
];

export const remoteMembersRolesCards = [
  {
    title: "成员列表 Members",
    lines: [
      "操作: 新增 · 编辑 · 删除 · 分配角色 · 禁用",
      "goya.admin@company.com  | admin, approver | enabled",
      "alice.dev@company.com   | developer       | enabled",
      "bob.viewer@company.com  | viewer          | disabled"
    ],
    tone: "default" as const
  },
  {
    title: "角色列表 Roles",
    lines: [
      "操作: 新增 · 编辑 · 删除 · 分配权限 · 禁用",
      "viewer      -> menus: read-only, no execution",
      "developer   -> menus: execute, no approval override",
      "approver/admin -> risk confirm & approval actions"
    ],
    tone: "default" as const
  },
  {
    title: "基线说明",
    lines: ["成员与角色的可见性与操作权限由 Hub 统一控制。"],
    tone: "info" as const
  }
];

export const remotePermissionsAuditCards = [
  {
    title: "菜单树与禁用 Menu Tree",
    lines: [
      "Workspace",
      "  ├─ Conversation (enabled)",
      "  ├─ Resources (enabled)",
      "  └─ Admin (disabled for viewer)"
    ],
    tone: "default" as const
  },
  {
    title: "角色权限分配 RBAC",
    lines: [
      "viewer: menu=read, action=none",
      "developer: menu=read/write, action=execute",
      "approver/admin: menu=all, action=approve/revoke",
      "ABAC 拒绝: 显示 403 inline alert + toast"
    ],
    tone: "danger" as const
  },
  {
    title: "审计列表 Audit",
    lines: [
      "actor | action | resource | result | time",
      "goya.admin | assign_role | member/alice | success | 10:41",
      "bob.viewer | run_execution | conv/42 | 403 denied | 10:44"
    ],
    tone: "default" as const
  }
];
