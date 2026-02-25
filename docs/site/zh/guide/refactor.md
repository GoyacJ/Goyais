# 前端重构范围（2026-02-25）

本阶段采用“去兼容、清债务”的硬约束：

- 删除前端 fallback/mock 运行时路径
- 全量迁移 Pinia
- 基于 UnoCSS 完成样式体系迁移并对齐 token
- Hub 在保持 `/v1` 的前提下移除 legacy 兼容分支
- 每个任务保留可复核的验证证据与检查点

## 计划文档

- [总计划](https://github.com/GoyacJ/Goyais/blob/main/docs/refactor/2026-02-25-frontend-refactor-master-plan.md)
- [任务板](https://github.com/GoyacJ/Goyais/blob/main/docs/refactor/2026-02-25-frontend-refactor-task-plan.md)
- [规格基线](https://github.com/GoyacJ/Goyais/blob/main/docs/refactor/2026-02-25-frontend-spec-plan.md)

## 当前验收门禁

- desktop lint/test/strict/coverage/token/quality
- hub `go test ./...`
- worker pytest + lint
- smoke E2E
- docs/slides 构建通过
