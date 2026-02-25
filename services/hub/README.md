# services/hub

Goyais v0.4.0 Hub 骨架服务（Go）。

## 启动

```bash
cd services/hub
export HUB_INTERNAL_TOKEN='<shared-random-token>'
PORT=8787 go run ./cmd/hub
```

## 测试

```bash
cd services/hub
go test ./...
```

## 已实现接口

- `GET /health`
- `GET /v1/workspaces|projects|conversations|executions`（空列表占位）
- 以上 `/v1/*` 的其余方法返回 `501 + StandardError`
