# services/worker

Goyais v0.4.0 Worker 执行服务（Python + FastAPI）。

## 初始化依赖

```bash
cd services/worker
uv sync
```

## 启动

```bash
cd services/worker
PORT=8788 uv run python -m app.main
```

## 测试

```bash
cd services/worker
uv run pytest
```

## 已实现接口

- `GET /health`
- Worker 启动后自动向 Hub 注册并发送心跳
- Worker 通过 `POST /internal/executions/claim` 主动认领执行
- Worker 通过 `POST /internal/executions/{execution_id}/events/batch` 回传事件
- Worker 通过 `GET /internal/executions/{execution_id}/control` 拉取 confirm/stop 控制命令

> 说明：Worker 依赖 `HUB_BASE_URL` + `HUB_INTERNAL_TOKEN` 与 Hub 内部 API 通信。
