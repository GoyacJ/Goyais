 ---                                                                                                                                                                                                                                      
一、核心原则：AI 重构 ≠ AI 从零写代码

这次重构的本质是拆解一个 2,572 行的 God Object 并重新布线，不是从空白画布上建新系统。AI 工具在"理解现有代码并精确迁移"这件事上远比"凭空生成"更容易出错。必须围绕这个特点设计工作流。
                                                            
---
二、项目准备工作（动工前必做）

1. 补全 AGENTS.md + CLAUDE.md（双轨）— 给 AI 注入架构上下文

当前根目录缺少双轨指令文档，`services/hub/CLAUDE.md` 也仅有初始化占位内容。这意味着每次 AI 会话都可能从零理解项目。建议创建并同步维护：

# AGENTS.md / CLAUDE.md（正文一致）

## 项目概述
Goyais 是一个 AI Agent 平台。Hub（Go 后端）是执行中枢。
当前处于 Agent v4 重构中，目标是将分散在 agentcore + httpapi 的能力
统一到 internal/agent/ 下的 Claude Code 风格架构。

## 重构上下文（必读）
- 架构方案：docs/refactor/2026-03-03-agent-v4-refactor-plan.md
- 任务计划：docs/refactor/refactor-taks-plan-table.md
- 当前阶段：Phase E（进行中，更新于 2026-03-04）
- 阶段判定依据：
  - A0（前置盘点与决策）文档已齐备并已落库。
  - A1（core 合同基线）已完成：`core/interfaces.go` 与核心类型/状态机/事件模型可编译；`runstate.go` 关键函数覆盖率达 100%。
  - A2/B 已完成：`internal/agent/runtime/loop`、`context/settings`、`context/prompt`、`runtime/compaction` 已落地并具备测试。
  - E 正在收敛：CLI/ACP 已统一接线；HTTP 默认模式已切至 `hybrid`，v4 成功时优先走 v4 主链。
  - 尚未满足 E→F 门禁：legacy fallback 与旧枚举/旧 orchestrator 清理尚未完成。

## 关键约束
1. internal/agent/ 是所有新代码的根目录
2. 不直接引用 AppState，通过接口注入
3. 不引入新的 map[string]any payload 读取
4. 所有新 Go 文件必须在 internal/agent/ 子包下
5. 新代码的 import 禁止引用 internal/agentcore 或 internal/httpapi

## 目录结构约定
internal/agent/
core/       — 状态机、事件、payload、errors
runtime/    — Engine 实现、loop、model、compaction
tools/      — registry、executor、diff、checkpoint
policy/     — permission、sandbox、approval
context/    — settings、prompt、mentions、composer
extensions/ — hooks、skills、subagents、teams、plugins、mcp
transport/  — subscriber manager、SSE/WS 桥接
adapters/   — httpapi、cli、acp、runtimebridge

## 验证命令
cd services/hub && go test ./... && go vet ./...

2. 为 internal/agent/ 创建子包级 AGENTS.md + CLAUDE.md

每个子包放一对简短的 `AGENTS.md` 与 `CLAUDE.md`，声明该包的职责边界、依赖方向、禁止事项，并保持正文一致。这比依赖 AI "记住"方案文档有效得多：

# internal/agent/runtime/AGENTS.md
## 职责：Engine 接口的真实现，执行循环控制
## 可依赖：core/、tools/（接口）、policy/（接口）
## 禁依赖：httpapi/、agentcore/、直接 AppState
## 迁移来源：execution_orchestrator.go 方法组 A+B+C+D（L53-480）

# internal/agent/runtime/CLAUDE.md
（与同目录 AGENTS.md 保持正文一致，允许标题不同）

3. 创建接口契约文件作为锚点

在动工前先手工创建 internal/agent/core/interfaces.go，把方案 §10.2 的 10 个接口签名落地为可编译的 Go 代码。这个文件是整个重构的不动锚点——AI 生成的所有实现代码都必须实现这些接口。

  ---
三、任务分配策略

按风险等级选择人/AI分工

┌──────┬───────────────────────────┬──────────────────────┬───────────────────────────────────────────────────────────────┐
│ 风险 │         任务类型          │      建议执行者      │                             原因                              │
├──────┼───────────────────────────┼──────────────────────┼───────────────────────────────────────────────────────────────┤
│ 高   │ A0（架构决策）            │ 人工                 │ 决策文档需要判断力，AI 容易产生看似合理但实际有隐患的架构选择 │
├──────┼───────────────────────────┼──────────────────────┼───────────────────────────────────────────────────────────────┤
│ 高   │ A2（Engine 真实现的骨架） │ 人工写骨架 + AI 填肉 │ 核心执行循环的编排逻辑必须人工把关，具体方法体可让 AI 迁移    │
├──────┼───────────────────────────┼──────────────────────┼───────────────────────────────────────────────────────────────┤
│ 高   │ F1（旧代码删除）          │ 人工                 │ 删除操作不可逆，需要人工确认每个引用确实已被替代              │
├──────┼───────────────────────────┼──────────────────────┼───────────────────────────────────────────────────────────────┤
│ 中   │ B1-B4、C1-C5              │ AI 为主 + 人审       │ 逻辑相对独立，有明确的输入/输出/验收标准                      │
├──────┼───────────────────────────┼──────────────────────┼───────────────────────────────────────────────────────────────┤
│ 低   │ D3-D7（新功能）           │ AI 为主              │ 从零实现，参照方案即可，风险较低                              │
└──────┴───────────────────────────┴──────────────────────┴───────────────────────────────────────────────────────────────┘

每个 AI 任务的标准 prompt 模板

## 任务：[任务编号] [任务名]

## 背景
你正在执行 Agent v4 重构的 [Phase A / A2]。
整体架构方案见 docs/refactor/2026-03-03-agent-v4-refactor-plan.md §[N]。

## 迁移来源（如适用）
- 源文件：internal/httpapi/execution_orchestrator.go L[start]-L[end]
- 源方法：[列出具体方法名]

## 目标输出
- 目标包：internal/agent/[package]/
- 必须实现的接口：core.XXXInterface
- 新文件：[列出预期文件名]

## 约束
1. 不引用 internal/agentcore 或 internal/httpapi
2. 不使用 map[string]any 作为业务读取入口
3. 通过接口注入依赖，不持有具体 struct 引用
4. 遵循 AGENTS.md 的 Change Surface Lock

## 验收标准
1. go build ./internal/agent/[package]/... 通过
2. go test ./internal/agent/[package]/... 通过
3. [具体业务验收条件]

## 参考代码
[粘贴或 @引用 相关源代码片段]

  ---
四、关键实操建议

1. 一个任务一个分支，一个分支一个 AI 会话

develop
├── refactor/a1-core-types
├── refactor/a2-engine-loop
├── refactor/b1-settings-merge
└── ...

不要在一个 AI 会话中跨任务工作。 每个任务完成后 commit、审查、合并回 develop 再开下一个。原因：
- AI 的上下文窗口有限，跨任务容易混淆
- 出问题时回退粒度清晰
- 人工 review 范围可控

2. "迁移"任务的正确姿势

对于从 EO 迁移方法的任务（占总工作量 ~40%），不要让 AI "参照旧代码重写"，而是：

步骤 1：让 AI 把旧方法原样复制到新位置
步骤 2：让 AI 编译，列出所有编译错误（依赖断裂点）
步骤 3：让 AI 逐个修复编译错误——每个断裂点就是一个需要注入接口的地方
步骤 4：让 AI 写测试覆盖迁移后的行为
步骤 5：人工审查接口设计是否合理

这比"请根据旧代码的逻辑重新实现"可靠得多，因为它保留了原始行为，只改变了依赖结构。

3. 用编译器当守卫，不靠 AI 自觉

在每个 Phase 完成后，添加编译期断言防止回退：

// internal/agent/core/guard.go
// 编译期检查：确保 Engine 接口被正确实现
var _ Engine = (*loop.EngineImpl)(nil)

// 编译期检查：确保 payload 关联正确
var _ EventPayload = (*OutputDeltaPayload)(nil)
var _ EventPayload = (*ApprovalNeededPayload)(nil)

4. 用 Golden Test 锁定行为

在 Phase A 完成后、开始拆 EO 之前，为 EO 的关键路径录制事件流快照：

// internal/httpapi/execution_orchestrator_golden_test.go
func TestGolden_NormalCompletion(t *testing.T) {
// 固定输入 → 录制事件序列 → 存为 testdata/golden/*.json
}
func TestGolden_ApprovalFlow(t *testing.T) { ... }
func TestGolden_ToolCallFailure(t *testing.T) { ... }

后续每个 Phase 完成后，用新 Engine 跑同样的输入，对比事件序列。这是方案 §13.2 的落地，也是 AI 迁移后最靠谱的回归手段。

5. 不要让 AI 一次生成超过 300 行

大量实践表明 AI 在单次生成 300+ 行代码时质量显著下降。拆解策略：

┌───────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│        不好的做法         │                                好的做法                                │
├───────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│ "请实现整个 hooks 子系统" │ "请实现 HookDispatcher 接口和 command handler 类型"                    │
├───────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│ "请迁移 EO 的模型调用"    │ "请先迁移 invokeOpenAICompatibleModelTurn（L2072-2127），仅这一个方法" │
├───────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│ "请为 policy 写所有测试"  │ "请为 Bash(npm run *) 规则写表驱动测试"                                │
└───────────────────────────┴────────────────────────────────────────────────────────────────────────┘

6. 利用 AGENTS.md 的 Change Surface Lock

你的项目已经有很好的 Change Surface Lock 机制。在 AI 执行每个任务时，明确声明：

Locked existing files:
- internal/httpapi/execution_orchestrator.go (只读参考，不修改)
- internal/agentcore/protocol/run_event.go (只读参考)

Target new files:
- internal/agent/runtime/loop/engine.go
- internal/agent/runtime/loop/engine_test.go

这可以防止 AI "顺手"修改不该动的文件。

7. Phase 间设置硬门禁

每个 Phase 完成后运行门禁脚本再进入下一阶段：

#!/bin/bash
# scripts/refactor/gate-check.sh

echo "=== Phase Gate Check ==="

# 1. 编译通过
cd services/hub && go build ./... || exit 1

# 2. 测试通过
go test ./... || exit 1

# 3. 旧包引用检查（Phase E+ 后应逐步降为 0）
AGENTCORE_REFS=$(grep -r "internal/agentcore" internal/agent/ | wc -l)
echo "agentcore refs in agent/: $AGENTCORE_REFS"

# 4. map[string]any payload 读取检查
MAPANY_READS=$(grep -rn 'Payload\[' internal/agent/ | grep -v '_test.go' | wc -l)
echo "map[string]any payload reads: $MAPANY_READS"

# 5. AppState 直接引用检查
APPSTATE_REFS=$(grep -r "AppState" internal/agent/ | grep -v 'adapter' | wc -l)
echo "AppState direct refs: $APPSTATE_REFS"

  ---
五、工具选择建议

┌──────────────────────────────┬───────────────────────────────┬─────────────────────────────────────────────────────┐
│             场景             │           推荐工具            │                        原因                         │
├──────────────────────────────┼───────────────────────────────┼─────────────────────────────────────────────────────┤
│ 结构拆解、方法迁移           │ Claude Code（交互模式）       │ 需要对话式迭代，逐步调整接口设计                    │
├──────────────────────────────┼───────────────────────────────┼─────────────────────────────────────────────────────┤
│ 批量代码生成（新功能、测试） │ Codex / Claude Code claude -p │ 明确输入输出的生成任务，适合非交互批量执行          │
├──────────────────────────────┼───────────────────────────────┼─────────────────────────────────────────────────────┤
│ 跨文件重命名/引用更新        │ IDE 重构工具（GoLand/gopls）  │ AI 做批量 rename 容易遗漏，IDE 的 rename 是确定性的 │
├──────────────────────────────┼───────────────────────────────┼─────────────────────────────────────────────────────┤
│ Golden test 录制             │ 人工 + go test -update        │ 测试基准必须人工验证正确性                          │
├──────────────────────────────┼───────────────────────────────┼─────────────────────────────────────────────────────┤
│ 合约同步（OpenAPI/TS）       │ Codex 后台任务                │ 机械性转换，适合 AI 批量处理                        │
├──────────────────────────────┼───────────────────────────────┼─────────────────────────────────────────────────────┤
│ 删除旧代码                   │ 人工 + IDE                    │ 删除操作需要确认每个引用已替代，不适合 AI 自主决策  │
└──────────────────────────────┴───────────────────────────────┴─────────────────────────────────────────────────────┘

  ---
六、最容易踩的坑

1. AI 会"创造性地"绕过接口约束

当 AI 发现通过接口注入比直接引用 AppState 麻烦时，它会倾向于添加一个"便利方法"直接透传。必须在 AGENTS.md 与 CLAUDE.md 中明确禁止，并用编译期检查兜底。

2. AI 会把两套混在一起而不是统一

让 AI "统一 ExecutionState 和 RunState"时，它可能会保留两套并加一个转换层——这恰恰是现在的问题。Prompt 中必须明确："最终产出中 ExecutionState 这个类型名不应存在"。

3. AI 在大文件迁移中会丢失语义

EO 的 executeSingleOpenAIToolCall（L1184-1407，223 行）包含 pre-hook → approval → execute → retry → post-hook 五段流程。AI 在迁移时容易丢失重试循环的边界条件。建议这类复杂方法人工审查每一行。

4. 测试覆盖的幻觉

AI 生成的测试经常"看起来全面"但实际上只测试了 happy path。对于核心模块（Engine、PermissionGate、HookDispatcher），要求 AI 明确列出"没有覆盖的边界场景"比要求它"写全面的测试"更有效。

  ---
七、推荐执行节奏

Week 1:  A0（人工）— 决策文档 + AGENTS.md/CLAUDE.md 双轨配置 + 接口契约文件 + Golden test 录制
Week 2:  A1-A3（AI+人审）— core/ 包 + Engine 骨架 + 枚举统一
Week 3:  B1-B4（AI 为主）— context/ 全部 + compaction
Week 4:  C1-C3（AI+人审）— tools/ 迁移（EO 拆解主战场）
Week 5:  C4-C5（AI+人审）— transport/ + model/ 拆解
Week 6:  D1-D2（AI+人审）— policy/ + hooks（安全关键）
Week 7:  D3-D7（AI 为主）— skills/subagents/teams/plugins/mcp
Week 8:  E1-E4（AI+人审）— 适配层 + session 生命周期
Week 9:  F1-F3（人工为主）— 删旧代码 + 合约同步 + 全量回归

最重要的一条：不要跳过 A0。没有锚定的接口契约和 Golden test 基线就开始让 AI 迁移代码，等于在流沙上建楼。
