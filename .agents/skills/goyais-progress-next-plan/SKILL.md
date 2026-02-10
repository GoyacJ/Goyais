---
name: goyais-progress-next-plan
description: 当任务需要浏览当前项目实现状态、确认开发进度并基于代码与文档制定下一步详细开发计划时触发；要求先执行证据扫描（baseline/acceptance/implementation/contract drift）再输出完整计划；当任务是单点微改或直接编码执行时不触发。
---

# goyais-progress-next-plan

将“现状盘点 -> 证据扫描 -> 下一步计划”固化为严格流程，避免只依据文档或印象给计划。

## 适用场景

- 需要回答“当前项目实现到哪一步、进度如何、下一步做什么”。
- 需要给出可直接执行的下一步 Slice 计划（含 DoD/测试/回滚）。
- 需要先扫描代码实现再制定计划，降低契约漂移风险。

## 非适用场景

- 单点微改、直接修 bug、直接落代码。
- 已明确只要执行，不需要状态盘点与计划。
- 与仓库状态无关的泛化讨论。

## 输入（必须读取）

- `/Users/goya/Repo/Git/Goyais/AGENTS.md`
- `/Users/goya/Repo/Git/Goyais/docs/prd.md`
- `/Users/goya/Repo/Git/Goyais/docs/spec/v0.1.md`
- `/Users/goya/Repo/Git/Goyais/docs/arch/overview.md`
- `/Users/goya/Repo/Git/Goyais/docs/arch/data-model.md`
- `/Users/goya/Repo/Git/Goyais/docs/arch/state-machines.md`
- `/Users/goya/Repo/Git/Goyais/docs/api/openapi.yaml`
- `/Users/goya/Repo/Git/Goyais/docs/acceptance.md`
- `/Users/goya/Repo/Git/Goyais/internal/access/http/router.go`
- `/Users/goya/Repo/Git/Goyais/internal/access/http/router_integration_test.go`

## 输出契约（固定 8 段）

1. `Baseline Snapshot`
2. `Acceptance Progress`
3. `Implementation Scan Matrix`
4. `Contract Drift Findings`
5. `Risk Register`
6. `Next Slices (DoD + Tests + Rollback)`
7. `Thread/Worktree Execution Plan`
8. `Evidence Appendix`

## 扫描参数

- `scan_depth=standard|deep`，默认 `deep`。
- `deep` 必须包含回归命令证据；若未执行必须写明“跳过原因 + 风险”。

## 严格步骤（先扫后计划）

### Step A：基线扫描

执行：

```bash
bash .agents/skills/goyais-progress-next-plan/scripts/progress_snapshot.sh
```

要求：记录 branch/HEAD/worktree/dirty，给出绝对路径证据。

### Step B：验收与文档扫描

执行：

```bash
bash .agents/skills/goyais-progress-next-plan/scripts/acceptance_stats.sh
```

要求：输出总项、完成项、未完成项、比例，并列出未完成条目。

### Step C：实现扫描

执行：

```bash
bash .agents/skills/goyais-progress-next-plan/scripts/implementation_scan.sh deep
```

要求：按域输出 `implemented|partial|placeholder|unknown`。

### Step D：契约漂移检测

要求：对比 OpenAPI、router、关键测试，列出 confirmed/partial/unknown 的漂移结论。

### Step E：产出下一步 Slice 计划

要求：每个 Slice 必须包含：

- 目标
- 范围
- 受影响文件（绝对路径）
- DoD
- 测试命令
- 回滚策略

## 证据格式（强制）

每条关键结论必须包含：

- `status`: `confirmed|partial|unknown`
- `path`: 绝对路径
- `command`: 关键命令（可复现）

必须全中文显示

## 与其他 skills 协作

- 约束核查：`goyais-norms`
- 切片与 DoD：`goyais-project-management`
- 并行 thread：`goyais-parallel-threads`

## 输出模板

- 状态盘点清单：`assets/status-audit-checklist.md`
- 实现扫描映射：`assets/implementation-scan-map.md`
- 下一步计划模板：`assets/next-plan-template.md`
- 风险矩阵模板：`assets/risk-matrix.md`

## 验收方式

- 输出必须包含固定 8 段。
- `Implementation Scan Matrix` 不可省略。
- 当存在 `501` 占位域时必须标记 `placeholder`。
- 每个 Slice 必须带 `DoD + Tests + Rollback`。
