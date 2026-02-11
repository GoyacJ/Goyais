# go_server

`go_server` 是 Goyais 的 Go 后端模块，提供 `/api/v1` 接口与单二进制发布能力（embed `vue_web/dist`）。

## 目录

- `cmd/`: 程序入口
- `internal/`: 业务与基础设施实现
- `migrations/`: SQLite/Postgres 迁移
- `scripts/`: CI 与 Git 防呆脚本
- `docs/`: Go 技术文档（架构/API/验收）

## 本地开发

```bash
cd /Users/goya/Repo/Git/Goyais/go_server
go test ./...
make build
```

## 前端联动构建

`make build` 会调用 `../vue_web` 进行前端构建并将产物拷贝到 `internal/access/webstatic/dist`。

## 核心文档

- 产品需求基线：`/Users/goya/Repo/Git/Goyais/docs/prd.md`
- Go 架构：`/Users/goya/Repo/Git/Goyais/go_server/docs/arch/overview.md`
- Go API：`/Users/goya/Repo/Git/Goyais/go_server/docs/api/openapi.yaml`
- Go 验收：`/Users/goya/Repo/Git/Goyais/go_server/docs/acceptance.md`
