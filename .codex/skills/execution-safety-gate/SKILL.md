---
name: execution-safety-gate
description: 对写入/执行/网络/删除类动作实施风险分级与确认门禁。
---

# 执行安全门禁技能

在提出或执行高风险动作前必须使用本技能。

## 触发条件

- write_fs, apply_patch, run_command, network/mcp_call, delete/revoke_key
- 任意可能导致源码变更、命令执行、数据外发的动作

## 输入

- 动作意图
- 目标文件/命令/端点
- workspace/project 边界上下文

## 工作流

1. 进行风险分级：
   - low: read/search/list
   - high: write/apply_patch, run_command, network/mcp_call
   - critical: delete/revoke_key
2. 校验防护：
   - path guard（仓库/worktree 范围）
   - command guard（仅安全模式）
   - boundary guard（Hub 权威链路）
3. 判定是否必须确认。
4. 定义审计预期。

## 输出契约

- `risk_level`
- `requires_confirmation`（yes/no）
- `guard_checks`（path/command/boundary）
- `audit_expectation`（应记录与可追踪内容）
- `blocked_reasons`（如被禁止）
- `rollback_audit_requirements`（涉及回滚时必填）

## 护栏

- high/critical 动作不得静默视为安全。
- 超出边界或明显危险动作必须阻断并解释原因。
- 与用户的风险说明默认使用中文。
- 涉及 `rollback` 的动作必须要求事件审计至少包含：request/apply/complete。
- 涉及模型目录同步失败时必须要求可追踪错误与重试审计。
