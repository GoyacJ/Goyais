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
- `POST /internal/executions`（接收 Execution 并启动执行循环）
- `POST /internal/executions/{execution_id}/confirm`（审批确认）
- `POST /internal/executions/{execution_id}/stop`（停止执行）
- `POST /internal/events`（内部事件接收，兼容测试与回放）

> 说明：Worker 可通过环境变量 `HUB_BASE_URL` + `HUB_INTERNAL_TOKEN` 将执行事件回传 Hub `/internal/events`。
