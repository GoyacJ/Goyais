# Goyais Codex Engineering Charter (v0.2)

本文件是 Goyais 仓库内 Codex 协作的根级权威规范（authoritative policy）。

## 1. Scope 与目标

- 仓库目标：构建面向企业级与开源社区的 AI 原生编排平台，工程目标对齐 Apache 顶级项目标准（治理透明、契约稳定、可审计交付）。
- 本规范覆盖：`go_server`、`vue_web`、`java_server`、`python_server`、`flutter_mobile`。
- 模块内可有子级 `AGENTS.md`，但不得与根规范冲突。

## 2. Source of Truth

- 产品与业务基线：`docs/prd.md`。
- 根目录 `docs/`：业务文档、需求文档、治理文档（business/governance focus）。
- Go 技术契约：`go_server/docs/`（arch/api/spec/acceptance）。
- Web 技术规范：`vue_web/docs/`（含 `web-ui.md`）。

## 3. Core Principles (MUST)

### 3.1 Command-first

- 所有副作用动作必须可表达为 Command。
- 规范入口：`POST /api/v1/commands`。
- Domain 写接口可保留为 sugar，但服务端必须转换为 Command 并记录审计。

### 3.2 Agent-as-User

- AI 永远代表当前登录用户执行，不拥有独立超管权限。
- 执行上下文至少包含：`tenantId/workspaceId/userId/roles/policyVersion/traceId`。
- 命令闸门与工具闸门都必须做授权校验。

### 3.3 Visibility + ACL + Egress

- 全对象支持 `visibility`：`PRIVATE | WORKSPACE | TENANT | PUBLIC`。
- ACL 权限集合：`READ | WRITE | EXECUTE | MANAGE | SHARE`。
- `PUBLIC` 仅允许具备发布权限角色设置。
- 对外发送数据必须经过 Egress Gate 并写入审计。

## 4. Contract Sync (MUST)

出现以下任一变更，必须同变更同步更新契约文档：

- API 路径、请求/响应、错误结构、分页语义。
- 核心实体字段、状态机、生命周期转换。
- 可见性与 ACL 判定规则。
- provider 抽象、配置键名、默认值、优先级。
- 单二进制静态路由、缓存策略、Content-Type 策略。

强制同步落点：

- `go_server/docs/api/openapi.yaml`
- `go_server/docs/arch/overview.md`
- `go_server/docs/arch/data-model.md`
- `go_server/docs/arch/state-machines.md`
- `go_server/docs/acceptance.md`

## 5. Repo Layout 语义

- `docs/`: 业务与治理（non-implementation-centric）。
- `go_server/`: Go 服务端实现与技术契约。
- `vue_web/`: Vue Web 实现与 UI 规范。
- `java_server/`: Java 服务端设计期模块。
- `python_server/`: Python 服务端设计期模块。
- `flutter_mobile/`: Flutter 移动端设计期模块。

## 6. Git + Worktree Policy (MUST)

- 每次编码改动必须在独立 `git worktree` 执行。
- 一线程一 worktree；禁止在同一 worktree 切换多个线程分支。
- 分支前缀固定：`goya/<thread-id>-<topic>`。
- 主仓库工作树仅用于集成、回归、发布前检查。
- 提交前必须执行：
  - `git diff --cached --name-only`
  - `bash go_server/scripts/git/precommit_guard.sh`

## 7. Engineering Role 与实现原则

- 角色：全栈高级工程师（Go/Java/Python/Vue/Flutter 等），先产品后技术，避免过度设计。
- 必须优先选择可验证、可回滚、可审计方案。
- 修改必须限定在本次任务范围，禁止夹带无关变更。

## 8. Frontend 与 Backend 强约束

- Web：必须遵循 `vue_web/docs/web-ui.md` 与 design-system token/hook；页面需支持多 panel 拖拽、缩放、全屏、一致性行为。
- Go：必须遵循 `/api/v1`、统一错误模型 `error: { code, messageKey, details }`、配置优先级 `ENV > YAML > default`。
- Java/Python/Flutter：当前阶段采用设计期强约束，先完成接口草案、DoD、验收矩阵与风险清单，再进入编码。

## 9. Release Shape (Single Binary)

- 生产发布必须支持单二进制：Go embed `vue_web/dist`。
- 路由优先级：`/api/v1/*` -> 静态资源 -> `favicon/robots` -> SPA fallback。
- `index.html`（含 fallback 命中）必须返回 `Cache-Control: no-store`。

## 10. Rules 与 Skills

- 规则目录：`.agents/rules/`。
- 技能目录：`.agents/skills/`。
- 使用顺序建议：
  1. `goyais-repo-norm-check`
  2. `goyais-cross-stack-plan`
  3. 进入具体模块 skill（go/vue/java/python/flutter）
  4. `goyais-contract-sync`
  5. `goyais-release-regression`

## 11. Language Policy

- 规范文档采用中英双语风格：中文主叙述 + 英文术语与命令。
- 对外协作文档保持术语一致，避免中英文语义漂移。

## 12. Hard Stops

- 禁止绕过 Command 执行副作用。
- 禁止未同步契约文档就合入契约变更。
- 禁止在主工作树日常开发。
- 禁止提交构建产物、大型二进制、数据库文件。
