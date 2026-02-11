---
name: goyais-repo-norm-check
description: 用于核查根规范、模块规范与契约一致性，定位冻结约束冲突与路径漂移。
---

# goyais-repo-norm-check

## 适用场景

- 方案评审前的规范对齐。
- API/模型/状态机变更前的约束核查。

## 输入

- `AGENTS.md`
- `go_server/AGENTS.md`
- `vue_web/AGENTS.md`
- `go_server/docs/*`
- `.agents/rules/*`

## 输出

- 满足/不满足/待确认清单。
- 冲突项文件与字段定位。

## 严格步骤

1. 读取根 AGENTS 与模块 AGENTS。
2. 对照 `go_server/docs/*` 契约做逐条核查。
3. 输出冲突并给出修订顺序（先契约后实现）。

## 验收

- 每个结论含证据路径与命令。
