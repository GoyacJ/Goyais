---
name: goyais-project-management
description: 当任务需要拆分垂直切片、定义 DoD、按验收驱动推进并进行风险控制时触发；当任务只是单点微改且无需里程碑或切片管理时不触发。
---

# goyais-project-management

将需求拆解为可交付、可验证、可回滚的垂直切片，并确保契约同步与风险可控。

## 适用场景

- 新功能立项或阶段规划，需要拆 Epic/Story/Slice。
- 需要定义 Definition of Done 并绑定验收条目。
- 需要把执行风险前置并设置控制措施。

## 非适用场景

- 只做一次性文本修改，不需要项目管理产出。
- 与 Goyais 无关的通用项目管理咨询。
- 纯探索性 PoC 且不要求进入正式交付。

## 输入（需要哪些仓库文件）

- `AGENTS.md`
- `docs/prd.md`
- `docs/spec/v0.1.md`
- `docs/acceptance.md`
- `assets/vertical_slice_checklist.md`
- `assets/definition_of_done.md`
- `assets/contract_sync_checklist.md`

## 输出（会改哪些文件/会生成哪些文件）

- 生成切片计划（目标、范围、约束、交付物、验收映射）。
- 生成 DoD 与风险清单。
- 生成契约同步清单与执行证据。

## 严格步骤

1. 先用 `docs/prd.md` 和 `docs/spec/v0.1.md` 确认业务目标、边界与里程碑。
2. 按垂直切片拆分：每个切片必须独立验证并可回滚，避免“横向分层但不可验收”的任务。
3. 使用 `assets/vertical_slice_checklist.md` 固化切片输入/输出/依赖/验收。
4. 使用 `assets/definition_of_done.md` 绑定 DoD，必须包含测试证据与文档同步。
5. 任何契约相关变化，按 `assets/contract_sync_checklist.md` 同步文档，禁止代码先变文档后补。
6. 每个切片都要定义风险项、触发条件、缓解动作和回滚方案。

## 验收方式

- 每个切片都能回答“目标是什么、何时算完成、失败怎么回滚”。
- DoD 清单可逐条勾选，不依赖口头说明。
- 契约变化有明确同步记录，且与 `AGENTS.md` 规则一致。
