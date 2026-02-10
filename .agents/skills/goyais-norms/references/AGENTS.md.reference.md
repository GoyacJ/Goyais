# AGENTS.md 约束引用（链接+关键摘录）

- source_path: `AGENTS.md`
- authority_scope: 仓库协作规则与 v0.1 冻结约束最高优先级之一

## key_constraints

- Command-first：副作用动作必须可表达为 Command，规范入口是 `POST /api/v1/commands`。
- Agent-as-User：AI 代表当前登录用户执行，不得拥有独立超管权限。
- 全对象 Visibility + ACL：`PRIVATE | WORKSPACE | TENANT | PUBLIC`，权限集合 `READ | WRITE | EXECUTE | MANAGE | SHARE`。
- Egress Gate：外发必须经闸门并可审计，敏感数据默认不明文外发。
- 配置优先级：`ENV > YAML > 默认值`。
- 单二进制与静态路由冻结：`index.html` 必须 `Cache-Control: no-store`，`favicon/robots` 缺省 404。
- 错误结构：`error: { code, messageKey, details }`。
- 并行 thread Git 隔离（MUST）：一线程一 worktree，禁止同一工作树切换多个 thread 分支。
- 分支指针安全（MUST）：指针移动前必须 `backup tag + backup branch`；覆盖远端历史仅允许 `--force-with-lease` 且需先给出 commit 清单与风险说明。

## sync_note

该文件仅做引用和摘录，不替代 `AGENTS.md`。如原文更新，必须同步更新本摘录。
