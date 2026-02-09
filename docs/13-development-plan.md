# Goyais 整体开发计划清单

> 本文档是实施级开发清单。每次开始开发前必须先阅读本文件、`15-development-standards.md`、`16-open-source-governance.md`，并核对 `12-dev-kickoff-package.md` 冻结契约。

最后更新：2026-02-09

---

## 1. 使用规则

1. 开发前：
   - 阅读 `12-dev-kickoff-package.md`（冻结契约）
   - 阅读本文件（任务清单与优先级）
   - 阅读 `15-development-standards.md`（开发规范）
   - 阅读 `16-open-source-governance.md`（开源治理与发布规范）
   - 确认本次实现严格对齐设计文档（00-11、12、15、16）
   - 在 `14-development-progress.md` 把对应任务标记为 `IN_PROGRESS`
2. 开发后：
   - 更新 `14-development-progress.md`（状态、测试结果、风险）
   - 若有契约变更，先回到 `12` 评审，再更新相关设计文档
3. 开发中发现文档错误：
   - 先修正文档（或提交文档修正变更）再继续实现
   - 禁止以“代码实现为准”绕过文档契约

---

## 2. 里程碑与任务清单

## 2.1 Sprint S0（契约落地与工程骨架）

- [ ] `S0-API-001` 建立 `/api` 路由骨架与统一响应封装
- [ ] `S0-API-002` 实现 `Authorization/X-Tenant-ID/X-Trace-ID/Idempotency-Key` 中间件
- [ ] `S0-EVT-001` 建立 `run_events` 表与写入接口（字段与 02/08 对齐）
- [ ] `S0-EVT-002` 实现 `/api/runs/{id}/events/stream`，固定 `event: run_event`
- [ ] `S0-EVT-003` 实现 `Last-Event-ID` 增量补发与 `seq` 缺口补拉支持
- [ ] `S0-CI-001` 建立基础 CI（lint + test + build）
- [ ] `S0-DOC-001` 输出 API Mock（供前端联调）
- [ ] `S0-OSS-001` 补齐开源治理文件（`CONTRIBUTING.md`、`SECURITY.md`、`GOVERNANCE.md`、`CODE_OF_CONDUCT.md`、`MAINTAINERS.md`）
- [ ] `S0-OSS-002` 建立 GitHub PR 模板与质量门禁清单
- [ ] `S0-OSS-003` 建立发布规范（版本策略、RC、CHANGELOG、SBOM/签名产物）

## 2.2 Sprint S1（Intent 编排 MVP）

- [ ] `S1-INT-001` 实现 `POST /intents` 文本意图入口
- [ ] `S1-INT-002` 实现 `POST /intents/{id}/plan` 重规划
- [ ] `S1-INT-003` 实现 `POST /intents/{id}/execute` 与动作执行状态机
- [ ] `S1-INT-004` 实现澄清问题返回（`INTENT_PARSE_FAILED.clarification_questions`）
- [ ] `S1-RBAC-001` 实现 `users/roles/permissions` 基础 CRUD
- [ ] `S1-APR-001` 实现 `high` 单人审批
- [ ] `S1-APR-002` 实现 `critical` 双人审批（`quorum_reached`）
- [ ] `S1-AUD-001` 审批与意图动作完整审计落库
- [ ] `S1-I18N-001` 实现 API locale 协商（`Accept-Language` / `X-Locale`）与响应 `meta.locale`
- [ ] `S1-I18N-002` 建立前端 i18n 基础设施（`useLocaleStore`、双语资源、运行时切换）

## 2.3 Sprint S2（资源与执行闭环）

- [ ] `S2-AST-001` 实现资产上传、导入、查询与血缘基础能力
- [ ] `S2-AST-002` 意图动作可引用历史资产（`input_assets`）
- [ ] `S2-WF-001` 实现 `POST /workflows/{id}/runs` 与 Intent `workflow.run` 映射
- [ ] `S2-RUN-001` 实现 `POST /runs/{id}/retry`
- [ ] `S2-CAS-001` 实现 Context CAS 冲突检测、回放与恢复
- [ ] `S2-POL-001` 落实 DataAccess 五维校验：
  - `bucket_prefixes`
  - `db_scopes`
  - `domain_whitelist`
  - `read_scopes`
  - `write_scopes`
- [ ] `S2-I18N-001` 完成审批/通知/错误消息的双语模板中心化

## 2.4 Sprint S3（全 AI 交互增强）

- [ ] `S3-VOI-001` 实现 `POST /intents/voice`（音频资产引用与转写输入）
- [ ] `S3-UI-001` 前端 `/assistant` 完整链路联调（确认、审批、重规划、执行轨迹）
- [ ] `S3-UI-002` SSE 异常 UX：断连、补拉、去重、冲突提示
- [ ] `S3-APR-001` 实现 `/approvals/{id}/rewrite` 改写回退重规划
- [ ] `S3-OBS-001` 观测页面联动 run/trace 查询

## 2.5 Sprint S4（稳定性与上线准备）

- [ ] `S4-PERF-001` 并发压测与容量评估（Run/SSE/EventStore）
- [ ] `S4-SEC-001` 安全与权限审计（跨租户、审批、数据访问）
- [ ] `S4-DR-001` 回放与故障恢复演练（Run/Context/审批）
- [ ] `S4-OPS-001` 灰度、回滚、监控告警开关配置
- [ ] `S4-REL-001` 发布说明与上线检查清单签核

---

## 3. 任务状态定义

| 状态 | 说明 |
|---|---|
| `TODO` | 未开始 |
| `IN_PROGRESS` | 开发中 |
| `BLOCKED` | 被依赖阻塞 |
| `READY_FOR_TEST` | 开发完成待验证 |
| `DONE` | 验收通过 |

---

## 4. 任务粒度与拆分规范

1. 每个任务必须有唯一 ID（如 `S2-WF-001`）。
2. 单任务建议在 0.5~2 人日内完成。
3. 每个任务必须能映射到一个验收标准与至少一个测试用例。
4. 任务涉及契约变更时，必须先评审 `12-dev-kickoff-package.md` 冻结条目。
