---
name: goyais-norms
description: 当任务涉及冻结决策对齐、架构/接口/鉴权约束核查、契约冲突排查时触发；当任务仅为与仓库规则无关的文案润色、纯展示改动或泛化讨论时不触发。
---

# goyais-norms

将 Goyais v0.1 的硬约束整理为 AI 可执行清单，确保方案与实现不偏离冻结决策。

## 适用场景

- 评审或编写实现方案，需要核对是否违反冻结约束。
- 设计 API、状态机、数据模型、静态路由策略，需先做规则对齐。
- 处理“代码/文档/验收不一致”问题，需给出统一判断依据。

## 非适用场景

- 纯 UI 文案调整、拼写修正、格式整理且不涉及契约或约束。
- 与 Goyais 仓库无关的通用技术问答。
- 仅执行已经确认无争议的机械性改动。

## 输入（需要哪些仓库文件）

- `AGENTS.md`
- `docs/prd.md`
- `docs/spec/v0.1.md`
- `docs/arch/overview.md`
- `docs/arch/data-model.md`
- `docs/arch/state-machines.md`
- `docs/api/openapi.yaml`
- `docs/acceptance.md`
- `references/AGENTS.md.reference.md`
- `references/arch-overview.reference.md`
- `references/acceptance.reference.md`

## 输出（会改哪些文件/会生成哪些文件）

- 生成或更新“冻结约束核对清单”类文档（通常放在任务输出或评审说明中）。
- 明确列出本次任务受影响的契约文档同步项。
- 必要时更新本 skill 的 `references/` 摘录，不改动权威原文。

## 严格步骤

1. 先读取 `AGENTS.md`，再读取架构、API、状态机、验收文档，按“文档优先于实现”建立约束基线。
2. 固定核对 Command-first：所有副作用动作可表达为 Command，规范入口为 `POST /api/v1/commands`。
3. 固定核对 Agent-as-User：执行上下文必须包含 `tenantId/workspaceId/userId/roles/policyVersion/traceId`，且命令闸门与工具闸门都校验授权。
4. 固定核对发布与静态服务：单二进制 embed、路由优先级、`index.html` 必须 `Cache-Control: no-store`、`/favicon.ico` 与 `/robots.txt` 缺省 404。
5. 固定核对配置与 provider：`GOYAIS_*` 与 `snake_case`、`ENV > YAML > 默认值`、provider 矩阵不越界。
6. 固定核对 API 与 i18n 错误模型：`error: { code, messageKey, details }`，`messageKey` 作为 i18n key。
7. 若任务触及 API/实体/状态机/可见性ACL/provider键名/静态路由与缓存策略，必须同步 `docs/api/openapi.yaml`、`docs/arch/data-model.md`、`docs/arch/state-machines.md`、`docs/arch/overview.md`、`docs/acceptance.md`。

## 验收方式

- 输出中逐条标注“满足/不满足/待确认”的冻结约束，不允许只给结论不给证据。
- 任一约束冲突都要指出来源文件与字段，不得引入新规则覆盖权威文档。
- 若存在冲突，先修文档契约达成一致，再安排实现改动。
