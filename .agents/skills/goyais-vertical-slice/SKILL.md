---
name: goyais-vertical-slice
description: 当任务进入具体模块实施，需要以“目标/范围/约束/产出/测试/验收”模板化推进垂直切片时触发；当任务仍处于基建 bootstrap 或仅做非功能性微调时不触发。
---

# goyais-vertical-slice

提供后续模块统一的垂直切片提示词模板，确保每个切片都可验收、可回滚、可审查。

## 适用场景

- 资产、工作流、插件、流媒体等模块进入实施阶段。
- 需要按统一模板给 AI/工程团队派发切片任务。
- 需要把测试与验收前置到切片定义中。

## 非适用场景

- Thread #2 工程骨架阶段（应使用 `goyais-thread2-bootstrap`）。
- 纯文档润色或无行为变化的小改动。
- 不需要形成可复用模板的一次性任务。

## 输入（需要哪些仓库文件）

- `AGENTS.md`
- `docs/prd.md`
- `docs/spec/v0.1.md`
- `docs/acceptance.md`
- `assets/vertical_slice_prompt_template.md`
- `assets/module_acceptance_template.md`

## 输出（会改哪些文件/会生成哪些文件）

- 生成模块级垂直切片 prompt。
- 生成模块验收模板（测试矩阵、通过标准、证据占位）。
- 输出契约同步与风险控制说明。

## 严格步骤

1. 先定义切片目标和验收终点，再定义实现路径。
2. 使用 `assets/vertical_slice_prompt_template.md` 填充任务上下文与硬约束。
3. 使用 `assets/module_acceptance_template.md` 定义正向、逆向、边界测试。
4. 每个切片必须显式声明 In Scope/Out of Scope，防止范围漂移。
5. 每个切片必须给出回滚策略与风险触发条件。
6. 涉及契约变化时，必须在同一变更同步文档。

## 验收方式

- 切片模板字段完整，无缺省关键项。
- 验收模板可直接用于测试执行与评审。
- 切片产出与冻结约束一致，无新增冲突规则。
