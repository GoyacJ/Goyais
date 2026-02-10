# Goyais 仓库协作规范（v0.1）

本文件定义 Goyais 仓库内的工程协作规则与架构冻结约束。除非明确升级版本，本文件中的“必须（MUST）”为强约束。

## 1. 核心原则

### 1.1 Command-first（必须）
- 所有会产生副作用的动作都必须可表达为 Command。
- `POST /api/v1/commands` 是副作用动作的规范入口（canonical）。
- AI 入口必须只通过 Command 执行，不允许 AI 直接绕过 Command 调用内部服务。
- Domain 写接口可作为语法糖保留，但内部必须转换为 Command 执行并记录审计。

### 1.2 Agent-as-User（必须）
- AI 永远代表当前登录用户执行，不拥有独立超管权限。
- 执行上下文至少包含：`tenantId/workspaceId/userId/roles/policyVersion/traceId`。
- 服务端必须在命令闸门和工具闸门都做授权校验，禁止绕过。

### 1.3 Visibility + ACL（全对象统一）
- 全对象必须支持 `visibility`：`PRIVATE | WORKSPACE | TENANT | PUBLIC`。
- 全对象必须支持 ACL 共享能力（至少 user/role 维度）。
- ACL 权限集合：`READ | WRITE | EXECUTE | MANAGE | SHARE`。
- `PUBLIC` 仅允许具备发布权限的角色设置。

### 1.4 Egress Gate（必须）
- 对外发送数据（模型/第三方服务）必须经过外发闸门（Egress Gate）。
- 默认禁止敏感数据明文外发，需策略允许后才可摘要/脱敏外发。
- 外发行为必须可审计。

## 2. 契约同步规则（Contract Sync）

以下任一项发生变化，必须在同一变更中同步更新文档：
- API 路径、请求/响应、错误结构、分页结构。
- 核心实体字段、状态机状态、生命周期转换。
- 可见性与 ACL 判定规则。
- provider 抽象与配置键名、默认值、优先级。
- 静态路由、缓存策略与 Content-Type 策略。

至少同步以下文件：
- `docs/api/openapi.yaml`
- `docs/arch/data-model.md`
- `docs/arch/state-machines.md`
- `docs/arch/overview.md`
- `docs/acceptance.md`

## 3. Provider 抽象与最小化运行

### 3.1 支持矩阵（v0.1）
- DB：`sqlite`、`postgres`
- Cache：`memory`、`redis`
- Vector：`sqlite`（fallback）、`redis_stack`
- Object Store：`local`、`minio`、`s3`
- Stream：`mediamtx`

### 3.2 最小化运行（必须可闭环）
- 组合：`SQLite + MediaMTX + 本地文件存储 + 本地缓存(memory)`。
- 要求：无外部云依赖也能完成 v0.1 基础闭环。

### 3.3 完整模式（推荐）
- 组合：`Postgres + Redis/Redis Stack + MinIO(S3 兼容) + MediaMTX`。

## 4. 配置规范（ENV/YAML）

### 4.1 命名规范
- ENV 统一前缀：`GOYAIS_`（全大写下划线）。
- YAML 顶层与字段统一 `snake_case`。

### 4.2 优先级
- `ENV > YAML > 默认值`。

### 4.3 映射规则
- 约定映射：`GOYAIS_X_Y_Z` ↔ `x.y.z`（或 `x_y_z`，由配置加载器定义并在文档中固定）。

## 5. API 统一约定

### 5.1 版本与上下文
- API 前缀固定：`/api/v1`。
- `v0.1` 是产品里程碑，不体现在 URL。
- 默认上下文来自 JWT claims：`tenantId/workspaceId/userId/roles`。
- 允许 `X-Workspace-Id`（可选 `X-Tenant-Id`）显式选择上下文。
- 服务端必须校验 header 选择范围属于 JWT 可访问范围，否则拒绝。

### 5.2 列表分页（Hybrid）
- 默认支持 cursor（适用于 run/step/log/事件等增长型数据）。
- 管理类列表保留 `page/pageSize`。
- 统一返回：
  - `items: []`
  - `pageInfo: { page, pageSize, total }`（可选）
  - `cursorInfo: { nextCursor }`（可选）
- 请求侧：若传 `cursor`，忽略 `page`。

### 5.3 写接口响应
- Domain 写接口统一返回：
  - `resource`（创建/更新后的资源快照）
  - `commandRef: { commandId, status, acceptedAt }`
- 对天然异步动作（如 `workflow.run`、`plugin.install`、`stream.record.start`），`resource` 可仅返回最小快照（`id/status`）。

### 5.4 错误响应与 i18n
- 统一错误结构：`error: { code, messageKey, details }`。
- `messageKey` 即 i18nKey，前端基于 `vue-i18n` 做本地化展示。

## 6. 前端与国际化约束

- 前端技术栈固定：`Vue + Vite + TypeScript + TailwindCSS`。
- 必须支持深色/浅色模式切换。
- 必须支持 `vue-i18n`，至少提供 `zh-CN` 与 `en-US` 资源。

## 7. 生产发布形态（Single Binary）

### 7.1 发布要求
- 生产发布必须是单二进制：Go 服务通过 embed 内嵌 Vite `dist`。
- 构建入口：`make build`（产物为单可执行文件）。
- 开发模式可分离：Vite dev server + API proxy。

### 7.2 路由优先级（必须）
1. `/api/v1/*` → API
2. 命中已存在静态文件 → 返回静态资源
3. 特殊路径 `favicon/robots` 策略（见 7.4）
4. 其他前端路由 → 回退 `index.html`（SPA fallback）

### 7.3 响应头约束（必须）
- 静态资源必须返回正确 `Content-Type`。
- `index.html`（包含 `/` 与 SPA fallback 命中场景）必须返回：`Cache-Control: no-store`。

### 7.4 favicon/robots 默认策略（v0.1）
- `GET /favicon.ico`：若无静态文件，占位不强制，默认返回 `404`。
- `GET /robots.txt`：若无静态文件，占位不强制，默认返回 `404`。
- 上述路径不走 SPA fallback 到 `index.html`。

## 8. 文档优先级

- `docs/prd.md` 是产品需求权威来源。
- 架构与接口以以下文档为实现前置契约：
  - `docs/arch/overview.md`
  - `docs/arch/data-model.md`
  - `docs/arch/state-machines.md`
  - `docs/api/openapi.yaml`
  - `docs/spec/v0.1.md`
  - `docs/acceptance.md`

出现冲突时处理顺序：
1. 先修订契约文档保持一致。
2. 再落地实现。
3. 禁止“代码先变、文档后补”导致契约漂移。

## 9. 并行 Thread Git 隔离规范（MUST）

### 9.1 Worktree 隔离（必须）
- 每个 thread 必须使用独立 `git worktree`；一个 worktree 只允许承载一个 thread 分支。
- 禁止在同一 worktree 来回切换多个 thread 分支。
- 主仓库工作树（仓库根目录）仅用于 `master` 集成、回归和发布前检查，不用于 thread 日常开发。

### 9.2 分支创建与命名（必须）
- thread 分支必须从本地 `master` 创建，命名为：`codex/<thread-id>-<topic>`。
- 推荐命令：
  - `git worktree add <path> -b codex/<thread-id>-<topic> master`

### 9.3 提交前防呆检查（必须）
- 每次提交前必须执行以下检查命令：
  - `git diff --cached --name-only`
  - `git diff --cached --name-only | rg '^(data/objects/|.*\.db$|build/|web/dist/|web/node_modules/|\.agents/)' && exit 1 || true`
- 若命中禁止路径，必须先清理 staged 后再提交。

### 9.4 分支指针安全（必须）
- 任何分支指针移动（如 `update-ref`、`reset`、`branch -f`）前，必须先创建 `backup tag + backup branch`。
- 若远端历史需要覆盖，仅允许 `push --force-with-lease`，且执行前必须输出“将要 push 的 commit 列表”与风险说明。

### 9.5 分支回收（必须）
- thread 合并完成后，必须移除对应 worktree 并执行 `git worktree prune`。
- 禁止长期保留无主分支或无用途 worktree，避免误操作污染。
