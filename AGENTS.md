# Goyais Codex 治理规范（仅适用于 v0.4.0）

本仓库使用严格的权威文档驱动治理模型。
所有方案、实现与评审决策都必须对齐 v0.4.0 文档体系。

## 使命

Codex 仅用于构建 Goyais v0.4.0。
任何建议、计划、实现都必须以 v0.4.0 文档为依据。

## 对话语言（强制）

- 与用户的自然语言对话必须始终使用中文。
- 代码、命令、路径、协议字段名可保留原文（通常为英文）。
- 若用户要求输出双语或翻译，可在中文说明下附带目标语言结果。

## 权威链路

以下文件是唯一实现权威来源：

1. `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/PRD.md`（业务权威）
2. `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/TECH_ARCH.md`（架构权威）
3. `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/IMPLEMENTATION_PLAN.md`（实施阶段权威）
4. `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/DEVELOPMENT_STANDARDS.md`（工程规范权威）

若文档冲突，先按 PRD 的业务语义裁决，再同步其余文档。

## 范围约束

- v0.1.x、v0.2.x、v0.3.x 文档不得作为实现权威。
- 历史版本仅可用于背景参考，不可作为决策依据。

## 核心领域约束

- 主术语必须使用 `Conversation`，不得以 `Session` 作为主名。
- 对象模型必须保持：`Workspace -> Project -> Conversation -> Execution`。
- 执行模型必须保持：
  - 多 Conversation 可并行。
  - 单 Conversation 严格 FIFO 且同一时刻仅一个活动执行。
- 架构信任边界必须保持：`Desktop -> Hub -> Worker`。
  Desktop 不得绕过 Hub 执行权威控制动作。

## 风险与安全约束

对于写入/命令执行/网络/删除语义，必须执行显式风险确认与可审计策略。

风险基线：

- `read/search/list` => low
- `write/apply_patch` => high
- `run_command` => high
- `network/mcp_call` => high
- `delete/revoke_key` => critical

## P0 / P1 发布约束

- P0 是发布阻塞项，必须满足 Go 条件。
- P1 是增强项，不得阻塞 v0.4.0 发布。

## 工程阈值（来自 v0.4.0 标准）

### 文件行数硬阈值

- Go: <= 400
- Python: <= 350
- TypeScript/TSX/Vue: <= 300
- Rust: <= 350

### 复杂度阈值

- 圈复杂度 <= 10
- 认知复杂度 <= 15

### 覆盖率阈值

- 核心模块 >= 80%
- 总体 >= 70%

核心模块包括权限、资源共享、密钥治理、执行调度。

### 前端结构与样式约束

- 采用 feature-first 模块组织。
- 不得使用全局平铺 `src/views/*` 作为主组织方式。
- 必须使用 token 三层：global -> semantic -> component。
- 组件内不得硬编码颜色/字体/间距/圆角。

## 文档同步义务

行为语义变更时，必须同提交更新权威文档：

- 业务规则变化 => 更新 `PRD.md`
- 接口/状态/模型变化 => 更新 `TECH_ARCH.md`
- 阶段/门禁策略变化 => 更新 `IMPLEMENTATION_PLAN.md` 与 `DEVELOPMENT_STANDARDS.md`

不得合并会导致权威文档语义不一致的改动。

## Multi-Agent 策略

仅在任务独立且无共享可变状态时启用并行分派。

- 2 个及以上独立子任务：优先并行分派。
- 存在共享可变状态或严格时序依赖：必须单代理执行。
- 必须执行最终整合校验：
  - 冲突文件与改动
  - 不变量保持
  - 权威文档一致性
  - 安全约束满足
