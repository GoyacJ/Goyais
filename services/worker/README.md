# services/worker

Goyais v0.4.0 Worker 骨架服务（Python + FastAPI）。

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
- `POST /internal/executions`（501 占位）
- `POST /internal/events`（501 占位）
