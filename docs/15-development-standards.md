# Goyais 开发规范文档

> 本文档定义实现阶段的统一工程规范。所有开发任务必须遵守本规范，并与 `12-dev-kickoff-package.md`、`13-development-plan.md`、`14-development-progress.md`、`16-open-source-governance.md` 联动执行。

最后更新：2026-02-09

---

## 1. 强制执行流程

### 1.1 开发前（必须）

1. 阅读 `12-dev-kickoff-package.md`，确认冻结契约与当前任务无冲突。
2. 阅读 `13-development-plan.md`，确认任务 ID、优先级、验收标准。
3. 阅读 `16-open-source-governance.md`，确认本次改动满足开源治理要求。
4. 阅读本文件，确认编码/测试/交付规范。
5. 在 `14-development-progress.md` 将任务状态改为 `IN_PROGRESS`，填写开始时间与负责人。
6. 明确本次开发以设计文档为准，禁止脱离文档自行定义协议或行为。

### 1.2 开发后（必须）

1. 完成代码与测试后，将任务状态更新为 `READY_FOR_TEST` 或 `DONE`。
2. 在 `14-development-progress.md` 记录测试结果、风险、遗留项。
3. 如发生契约变更，先更新 `12-dev-kickoff-package.md`，再同步相关设计文档（00-11）。
4. 若新增 API/事件/错误码，必须同步更新 `10-api-design.md` 与相关领域文档。
5. 若变更影响开源流程（贡献方式、发布方式、安全响应），必须同步更新根目录治理文件与 `16-open-source-governance.md`。
6. 若实现中发现文档错误或冲突，必须先修正文档并完成同步，再继续代码实现。

---

## 2. 分支与提交规范

### 2.1 分支命名

- 开发分支前缀固定：`codex/`
- 推荐格式：`codex/<task-id>-<short-topic>`
- 示例：`codex/S1-INT-001-intent-entry`

### 2.2 提交规范

- 提交必须可回溯到任务 ID。
- 推荐格式：`<type>(<scope>): <summary> [<task-id>]`
- `type` 取值：`feat` `fix` `refactor` `docs` `test` `chore`
- 示例：`feat(intent): add intent planning endpoint [S1-INT-001]`

### 2.3 合并要求

- 禁止直接向主分支提交未评审代码。
- 所有变更通过 Pull Request 合并。
- PR 必须关联任务 ID 与验收标准。

---

## 3. 代码与架构规范

### 3.1 契约优先

1. `02-domain-model.md` 是领域模型权威源，实体与枚举命名不得私自变体。
2. `08-observability.md` 是事件语义权威源，新增事件需补充 payload schema。
3. `10-api-design.md` 是外部接口权威源，接口行为必须与文档一致。
4. 开发实现必须严格执行设计文档，不允许“先写代码后补文档”替代契约评审流程。
5. 发现文档错误时，先修正文档并完成交叉同步，再进行实现或合并。

### 3.2 通用工程要求

- 禁止跨租户数据读取/写入。
- 所有写操作必须带审计上下文（`trace_id`, `tenant_id`, `actor_id`）。
- 幂等端点必须支持 `Idempotency-Key`。
- 关键链路错误必须返回明确域错误码，不可只返回通用错误字符串。

### 3.3 命名与兼容要求

- Go 常量使用 PascalCase，线上 JSON/SSE 事件类型使用 snake_case。
- 新增字段优先“向后兼容扩展”，禁止破坏已有字段语义。
- 任何删除/重命名行为需要迁移方案与兼容窗口。

---

## 4. API 与事件规范

### 4.1 API 规范

- 基础路径固定：`/api`
- 统一追踪头：`X-Trace-ID`
- 统一租户头：`X-Tenant-ID`
- 统一幂等头：`Idempotency-Key`
- 统一响应结构：成功/失败结构必须稳定且可机器解析。

### 4.2 SSE 规范

- `event` 固定：`run_event`
- 业务事件类型放在 `data.type`
- 事件必须带 `id`、`seq`，并支持断线补拉与去重
- 事件顺序按 `seq` 保障可恢复处理

### 4.3 错误码规范

- 错误码按域分组：`INTENT_*`, `POLICY_*`, `CONTEXT_*`, `TOOL_*`, `STREAM_*`
- 所有新错误码需更新 `10-api-design.md` 的错误码章节。

### 4.4 国际化规范

- API 必须支持 `Accept-Language` 协商；支持 `X-Locale` 显式覆盖。
- 响应 `meta` 应包含 `locale` 与 `fallback`。
- 所有用户可见新增文案必须提供 `zh-CN` 与 `en`。
- 错误响应优先使用稳定 `message_key`，文案可本地化替换。

---

## 5. 安全与治理规范

1. RBAC 校验前置，权限不足必须拒绝执行并记录审计日志。
2. 高风险动作走审批流程，`critical` 动作执行双人审批。
3. DataAccess 五维校验必须生效：
   - `bucket_prefixes`
   - `db_scopes`
   - `domain_whitelist`
   - `read_scopes`
   - `write_scopes`
4. 审批、预算、策略拒绝必须产生可检索 RunEvent。

---

## 6. 测试与质量规范

### 6.1 测试分层

- 单元测试：覆盖纯逻辑与边界条件。
- 集成测试：覆盖 API、DB、事件存储、SSE。
- 端到端测试：覆盖“意图输入 -> 计划 -> 审批 -> 执行 -> 观测”主链路。

### 6.2 最低门槛

1. 新增核心逻辑必须有单元测试。
2. 新增端点必须有至少 1 个成功用例和 1 个失败用例。
3. 新增事件类型必须有 schema 校验与序列化测试。
4. 涉及审批/权限的改动必须有拒绝路径测试。
5. 新增用户可见文案必须有 `zh-CN` 与 `en` 双语验证。

### 6.3 回归检查

- 每次合并前至少执行：lint + unit + integration。
- 发布前执行完整回归（含 e2e、性能与恢复演练关键用例）。

---

## 7. 文档与进度更新规范

### 7.1 文档更新触发条件

以下情况必须同步更新文档：

- 领域模型字段/枚举变更
- API 请求或响应契约变更
- RunEvent 类型或 payload 变更
- 审批、策略、安全规则变更
- 前端关键交互或错误处理流程变更

### 7.2 进度更新要求

- 开发开始：在 `14-development-progress.md` 标记 `IN_PROGRESS`
- 开发完成：更新状态、测试结果、风险、影响范围
- 阻塞出现：立即改为 `BLOCKED` 并记录依赖项与解除条件

---

## 8. PR 审核清单

提交 PR 前自检：

- [ ] 已对齐 `12` 冻结契约
- [ ] 已对齐 `13` 任务与验收标准
- [ ] 已更新 `14` 任务状态与测试结果
- [ ] API/事件/错误码文档已同步
- [ ] 权限、审批、审计路径已覆盖
- [ ] 回归测试通过且无新增高风险告警
- [ ] 已满足 `docs/16-open-source-governance.md` 的开源治理与发布要求
- [ ] 实现与设计文档一致；如有文档错误已先修正并记录

---

## 9. DoD（任务完成定义）

任务仅在满足以下条件时可标记 `DONE`：

1. 功能实现满足任务验收标准。
2. 对应测试完成并通过。
3. 文档与进度已同步更新。
4. 不存在未记录的高风险遗留问题。
5. 评审意见已处理或形成可追踪结论。

---

## 10. 开源治理与企业级门禁（新增）

### 10.1 开源治理基线

1. 仓库必须维持 Apache 2.0 许可与清晰版权声明。
2. 必须具备贡献与治理入口文件：
   - `CONTRIBUTING.md`
   - `SECURITY.md`
   - `GOVERNANCE.md`
   - `CODE_OF_CONDUCT.md`
   - `MAINTAINERS.md`
3. 架构变更必须形成可追溯决策记录（建议 ADR/RFC）。

### 10.2 GitHub 质量门禁

1. 默认分支开启保护：禁止直接 push，必须通过 PR。
2. PR 必须至少通过：
   - lint
   - unit tests
   - integration tests（若受影响）
   - 安全扫描（依赖与静态分析）
3. PR 必须包含变更说明、测试证据、兼容性说明。

### 10.3 兼容性与发布要求

1. 外部 API/SSE/错误码采用向后兼容优先策略。
2. 破坏性变更必须提供迁移指南与弃用窗口。
3. 每次正式发布必须包含：
   - 发布说明（CHANGELOG/Release Notes）
   - 可复现构建信息
   - 供应链材料（例如 SBOM）与产物校验信息
