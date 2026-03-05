# Week 3 Preflight Checklist（Session/Run 重构）

- 日期：2026-03-05
- 目标：完成 Week 3 Hub 收口，并冻结 Week 4 目录迁移准入边界。
- 范围：`services/hub`（主），`apps/desktop`（仅最小联动验证）。

---

## 1) Week 3 Hub 收口完成项

### 1.1 Runtime 命名与仓储收口

1. sqlite runtime 表/索引命名已收口为无版本后缀（`runtime_sessions`、`runtime_runs`、`runtime_run_events`、`runtime_run_tasks`、`runtime_change_sets`、`runtime_hook_records`）。
2. runtime 仓储装配命名已收口（`RuntimeRepositorySet` / `NewSQLiteRuntimeRepositorySet`）。
3. runtime 日志文案已收口（去版本后缀并统一到 runtime 语义）。

### 1.2 权限键收口

1. Hub 权限键已切换：`conversation.read|conversation.write|execution.control` -> `session.read|session.write|run.control`。
2. 默认角色权限、权限字典、handler 审计键已同步更新。
3. Desktop 权限展示与 workspace 测试断言已完成最小联动。

### 1.3 测试文件命名收口

1. `state_execution_runtime_sync_test.go`
2. `repository_sqlite_test.go`
3. `run_task_query_service_test.go`

---

## 2) DB 破坏式重建验证（W3-C1）

### 2.1 执行证据

1. 新增测试：`TestOpenAuthzStoreSupportsRuntimeSchemaAfterTwoColdStarts`。
2. 每轮步骤：删除 sqlite 文件（含 `-wal/-shm`）-> 冷启动 `openAuthzStore` -> 校验 runtime 仓储链路。
3. 覆盖对象：`session`、`run`、`run events`、`changeset`、`hooks`。

### 2.2 补充重启验证

1. `TestProjectConfigPersistsAcrossRouterRestart`
2. `TestWorkspaceAgentConfigPersistsAndExecutionSnapshotIsFrozen`

---

## 3) Week 4 迁移 preflight（仅准备，不执行目录迁移）

### 3.1 边界冻结

1. 路由与视图入口：`apps/desktop/src/router/index.ts`、`modules/conversation/views/*`。
2. 状态与服务：`modules/conversation/store/*`、`modules/conversation/services/*`。
3. 组件与追踪链路：`modules/conversation/components/*`、`modules/conversation/trace/*`。
4. 测试与 mock：`modules/conversation/tests/*`、`modules/project/store/project-store.spec.ts`。
5. i18n 与消费方：`shared/i18n/messages.*`、`shared/services/sseClient.ts`、`shared/stores/workspaceStatusStore*`。

### 3.2 批次切分（最小可回滚）

1. Batch A：目录迁移（`modules/conversation` -> `modules/session`）+ 路由入口修复。
2. Batch B：import 链路修复（store/services/views/components/tests）。
3. Batch C：i18n key 收口与可见文案回归。
4. Batch D：全量回归与语义审计收口。

### 3.3 回滚粒度

1. 目录迁移失败：回滚 Batch A。
2. import 修复失败：回滚 Batch B。
3. i18n 缺词失败：回滚 Batch C。
4. 回归门禁失败：回滚到最近通过批次并重新切片。

---

## 4) Week 3 / Week 4 准入命令

1. `cd services/hub && go test ./... && go vet ./...`
2. `pnpm contracts:generate && pnpm contracts:check`
3. `pnpm lint && pnpm test`
4. `scripts/refactor/gate-check.sh`
5. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts`
6. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts`
7. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" services/hub apps/desktop/src packages/shared-core/src packages/contracts`
