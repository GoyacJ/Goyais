export const workspaceAgentCards = [
  {
    title: "基础执行配置",
    lines: [
      "默认模式: Agent",
      "Stop 策略: hard-stop + graceful cleanup",
      "并发限制: 多 Conversation 并行, 单 Conversation FIFO"
    ],
    tone: "default" as const
  },
  {
    title: "Agent 模式",
    lines: ["LangGraph 模式: 稳定流程编排", "Deep Agents 模式: 深度推理与多阶段计划"],
    tone: "default" as const
  },
  {
    title: "SubAgent 特性",
    lines: ["enable: true · max_subagents: 4 · scheduler: priority+quota"],
    tone: "info" as const
  }
];

export const workspaceModelCards = [
  {
    title: "厂商配置 Vendor",
    lines: [
      "OpenAI · Google · Qwen · Doubao · Zhipu · MiniMax · Local",
      "操作: 新增厂商 · 编辑密钥 · 删除厂商"
    ],
    tone: "default" as const
  },
  {
    title: "模型配置 Vendor -> Models",
    lines: ["OpenAI: gpt-5.3, gpt-5-mini", "Google: gemini-3.1-pro-preview", "默认模型: gpt-5.3"],
    tone: "default" as const
  },
  {
    title: "分享工作区与风险提示",
    lines: ["共享模型密钥属于高风险操作，需 approver/admin 确认。"],
    tone: "warning" as const
  }
];

export const workspaceRulesCards = [
  {
    title: "规则列表",
    lines: ["secure-defaults (shared, approved)", "repo-guard (private, pending)"],
    tone: "default" as const
  },
  {
    title: "规则编辑器",
    lines: [
      "rule \"block_secret_write\" {\n  when resource == \"mcp.key\"\n  require role in [\"approver\",\"admin\"]\n}"
    ],
    tone: "default" as const,
    mono: true
  },
  {
    title: "作用域与审批状态",
    lines: ["状态: pending / approved / denied / revoked"],
    tone: "info" as const
  }
];

export const workspaceSkillsCards = [
  {
    title: "技能列表",
    lines: ["review-pr (shared, approved)", "deploy-release (private, pending)"],
    tone: "default" as const
  },
  {
    title: "技能编辑器",
    lines: [
      "skill: mcp.debug\ndescription: inspect mcp server health\ninput: server_id, timeout"
    ],
    tone: "default" as const,
    mono: true
  },
  {
    title: "来源与作用域",
    lines: ["private/shared 与审批状态会影响可见性与可执行性。"],
    tone: "info" as const
  }
];

export const workspaceMcpCards = [
  {
    title: "MCP 列表与状态",
    lines: [
      "github-mcp     connected",
      "slack-mcp      disconnected",
      "local-fs-mcp   connected"
    ],
    tone: "default" as const
  },
  {
    title: "MCP 管理",
    lines: ["操作: 新增 · 编辑 · 删除 · 启停 · 测试连接", "受权限约束时仅展示只读信息和禁用操作"],
    tone: "default" as const
  },
  {
    title: "安全与审计",
    lines: ["连接信息、密钥变更与启停动作都会进入审计日志。"],
    tone: "warning" as const
  }
];

export const workspaceProjectConfigCards = [
  {
    title: "项目范围绑定",
    lines: [
      "绑定维度: 模型 / 规则 / 技能 / MCP",
      "作用域: project_id",
      "Conversation 创建时自动继承"
    ],
    tone: "default" as const
  },
  {
    title: "覆盖语义",
    lines: [
      "Conversation 可临时覆盖模型、规则、技能、MCP",
      "覆盖不会反写 ProjectConfig"
    ],
    tone: "info" as const
  },
  {
    title: "治理与审计",
    lines: [
      "远程工作区按 RBAC/ABAC 控制编辑权限",
      "变更记录写入审计并附带 trace_id"
    ],
    tone: "warning" as const
  }
];
