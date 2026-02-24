---
name: change-impact-sync
description: 为 v0.4.0 语义变更生成精确的跨文档同步矩阵。
---

# 变更影响同步技能

用于任何会影响语义的一类改动，确保文档一致性不漂移。

## 触发条件

- API/接口变化
- 状态机或队列语义变化
- 权限/安全/风险策略变化
- 执行流程变化
- 发布门槛或验收标准变化

## 输入

- 拟变更或已变更摘要
- 触及文件清单
- v0.4.0 权威文档

## 工作流

1. 对变更进行类型归类。
2. 映射到必须更新的权威文档。
3. 输出明确的同步矩阵。
4. 在完成前标记缺失项。

## 输出契约

输出表格字段：

- `change_type`
- `code_or_doc_change`
- `required_docs_to_update`
- `required_sections`
- `status`（done/missing）

建议附加字段（可选但推荐）：

- `evidence_refs`（对应章节或接口证据）

强制映射：

- 业务规则 => `PRD.md`
- 接口/状态/模型 => `TECH_ARCH.md`
- 阶段/门禁/规范 => `IMPLEMENTATION_PLAN.md` + `DEVELOPMENT_STANDARDS.md`

v0.4.0 当前优先同步锚点：

- 接口/状态/模型：`TECH_ARCH.md` 3/7/9/11/14/20
- 阶段与验收：`IMPLEMENTATION_PLAN.md` Phase 3/4/5/7/9 + 测试计划映射
- 门禁与 DoD：`DEVELOPMENT_STANDARDS.md` 10/11/13/14/15

## 护栏

- 必需同步项缺失时，不得宣称完成。
- 术语必须与 v0.4.0 一致（`Conversation` 主名）。
- 用户说明默认使用中文。
- 若包含 `rollback` 语义变更，必须显式核对快照回滚范围定义。
