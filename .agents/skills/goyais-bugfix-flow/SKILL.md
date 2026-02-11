---
name: goyais-bugfix-flow
description: 缺陷修复标准流程，包含复现、最小修复、验证、风险与回滚。
---

# goyais-bugfix-flow

## 适用场景

- 功能缺陷、回归问题、线上故障修复。

## 输入

- `AGENTS.md`
- `../goyais-worktree-flow/SKILL.md`

## 输出

- 根因说明。
- 修复说明。
- 验证证据与回滚步骤。

## 严格步骤

1. 在独立 worktree 复现问题。
2. 实施最小修复，避免范围扩散。
3. 执行针对性回归 + 全量回归。
4. 记录风险与回滚命令。

## 验收

- 问题可复现可验证，修复可审计。
