# 总览

Goyais 由三块运行时组成：

- Desktop：基于 Vue + Tauri 的桌面应用，负责本地/远程工作区编排。
- Hub：Go HTTP API，提供工作区、资源、执行与管理能力。
- Worker：Python 运行时，负责执行编排与安全策略约束。

## 构建与质量命令

- `pnpm lint`
- `pnpm test`
- `pnpm test:strict`
- `pnpm coverage:gate`
- `pnpm e2e:smoke`
- `pnpm docs:build`
- `pnpm slides:build`

## 关键文档入口

- 重构计划：[docs/refactor](https://github.com/GoyacJ/Goyais/tree/main/docs/refactor)
- 发布清单：[docs/release-checklist.md](https://github.com/GoyacJ/Goyais/blob/main/docs/release-checklist.md)
- 评审记录：[docs/reviews](https://github.com/GoyacJ/Goyais/tree/main/docs/reviews)
