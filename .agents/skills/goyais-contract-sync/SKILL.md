---
name: goyais-contract-sync
description: 当实现触及契约边界时，确保 OpenAPI/架构/状态机/验收文档同步更新。
---

# goyais-contract-sync

## 适用场景

- API、数据模型、状态机、provider 配置、静态路由策略变更。

## 输入

- `go_server/docs/api/openapi.yaml`
- `go_server/docs/arch/overview.md`
- `go_server/docs/arch/data-model.md`
- `go_server/docs/arch/state-machines.md`
- `go_server/docs/acceptance.md`

## 输出

- 同步变更列表。
- 漂移修复说明。

## 严格步骤

1. 标记本次契约影响面。
2. 同步更新对应文档。
3. 执行回归并记录证据。

## 验收

- 无“代码已变更、文档未同步”残留。
