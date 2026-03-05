# Session/Run 语义统一重构主计划

- 日期：2026-03-05
- 状态：执行基线（Baseline）
- 适用范围：`services/hub`、`apps/desktop`、`packages/shared-core`、`packages/contracts`、`scripts`、`docs`
- 文档角色：本文件是本轮重构唯一主计划；任务排期与风险管理分别见同目录 `task-schedule` 与 `risk-register`。

---

## 1. 目标与非目标

### 1.1 目标

1. 全仓统一运行链路术语为 `session/run`，移除 `conversation/execution` 主语义。
2. 对外 API 保持 `/v1/*`，但对外数据契约与代码命名均收敛为 `session/run`。
3. 移除兼容实现（legacy/compat/fallback/alias），仅保留最新架构与实现路径。
4. 清理内部版本后缀命名（`v1/v2/v3/v4`），避免在代码命名中继续传播版本语义。
5. 建立可持续的周更机制，保障重构按计划推进并可审计收口。

### 1.2 非目标

1. 不进行旧数据兼容迁移（允许破坏式重建）。
2. 不新增平行重构方案文档，不维护双基线。
3. 不在本轮处理第三方或平台强制命名（例如 lockfile、平台目录命名、外部 API URL）。

---

## 2. 架构不变式（不得破坏）

1. Hub 中心化控制不变：Desktop/Mobile 不得绕过 Hub 执行控制。
2. 执行语义不变：单会话同一时刻仅 1 个活跃执行，且会话内 FIFO。
3. 工作区边界不变：授权与数据隔离边界持续有效。
4. 契约一致性不变：Hub API、OpenAPI、shared-core 生成类型保持一致。

---

## 3. 术语与语义映射

| 旧语义 | 新语义 | 说明 |
|---|---|---|
| `conversation` | `session` | 会话主资源统一命名 |
| `execution` | `run` | 执行实体统一命名 |
| `conversation_id` | `session_id` | 事件/请求/响应字段统一 |
| `execution_id` | `run_id` | 事件/请求/响应字段统一 |
| `active_execution_id` | `active_run_id` | 会话活跃执行指针统一 |
| `conversation.*` permission | `session.*` permission | 权限键统一 |
| `execution.control` permission | `run.control` permission | 权限键统一 |

---

## 4. API 与命名规则

### 4.1 API 规则

1. 对外路径继续保留 `/v1/*`。
2. 除路径前缀外，不在对外契约中保留版本命名痕迹。
3. Hook 路径与 payload 使用 `run` 语义（例如 `/v1/hooks/runs/{run_id}`）。

### 4.2 命名规则

1. 内部命名禁止 `v1/v2/v3/v4` 后缀（文件名、类型名、变量名、常量名、方法名、DB 表与索引名）。
2. 禁止新增 `conversation/execution` 主语义命名。
3. 允许白名单：第三方依赖名称、平台强制目录名称、外部 API 字面 URL。

---

## 5. 兼容策略（强制）

1. 无兼容层：不保留 legacy runtime、compat adapter、fallback 路由分支。
2. 无 alias：不保留旧类型别名、旧字段镜像、旧路径别名。
3. 无双字段并存：不再同时输出旧字段与新字段。
4. 无过渡导出：shared-core 仅导出 `session/run` 命名模型。

---

## 6. 影响面清单

1. Hub：`internal/httpapi`、`internal/runtime`、`internal/agent`、`cmd/goyais-cli`。
2. Desktop：`apps/desktop/src/modules/conversation` 全目录与其引用链。
3. Shared Core：`packages/shared-core/src/api*` 与 `generated/openapi.ts`。
4. OpenAPI：`packages/contracts/openapi.yaml`。
5. 脚本：语义门禁脚本与质量门禁入口。
6. 文档：`docs/site`、`docs/slides`、根说明文档中的旧语义与旧计划引用。

---

## 7. 公共接口/类型变更清单（强制执行）

1. 路径仍为 `/v1/*`。
2. 不再暴露 `Conversation/Execution` 为主类型。
3. 不再暴露 `conversation_id/execution_id/active_execution_id`。
4. SSE 不再输出 `legacy_event_type`。
5. Hook 路径与 payload 一律 run 语义。
6. shared-core 仅导出 `session/run` 命名。

---

## 8. Definition of Done

### 8.1 语义收口

1. 首方代码主链路无 `conversation/execution` 旧语义命名残留（白名单除外）。
2. 首方代码主链路无 `v1/v2/v3/v4` 内部版本后缀命名残留（`/v1` 路径除外）。
3. 无 legacy/compat/fallback/alias 运行时路径残留。

### 8.2 契约一致

1. OpenAPI、Hub 路由、shared-core 生成类型一致。
2. Desktop 仅使用新契约字段。

### 8.3 回归门禁命令

1. `pnpm contracts:generate && pnpm contracts:check`
2. `cd services/hub && go test ./... && go vet ./...`
3. `pnpm lint && pnpm test && pnpm test:strict && pnpm e2e:smoke`
4. `pnpm lint:mobile && pnpm test:mobile && pnpm build:mobile && pnpm --filter @goyais/mobile e2e:smoke`
5. `pnpm docs:build && pnpm slides:build`
6. `make health`

---

## 9. 执行期间维护机制

1. 每周更新 `task-schedule`：完成状态、下周计划、阻塞项与解法。
2. 每周更新 `risk-register`：风险状态、触发信号、缓解与回滚动作。
3. 每次跨模块改动同步更新本文件“影响面清单”与“接口变更清单”。
4. 禁止新增平行重构主计划文档，避免基线漂移。

---

## 10. 假设与默认

1. 本轮排期使用 6 周窗口。
2. 对外 API `/v1` 保留。
3. 数据层允许破坏式重建，无历史兼容迁移。
4. 平台/第三方强制命名不纳入去版本与去旧语义硬约束。
5. 单分支分阶段提交，最终一次收口合并。

