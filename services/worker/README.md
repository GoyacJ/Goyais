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
export HUB_INTERNAL_TOKEN='<shared-random-token>'
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

## TLS 证书配置（模型请求）

若模型 API 所在网络使用企业代理/自签名证书链，可使用以下环境变量：

- `WORKER_TLS_CA_FILE`：指向自定义 CA 证书文件（PEM）。
- `WORKER_TLS_INSECURE_SKIP_VERIFY=1`：跳过 TLS 校验（仅建议临时调试使用）。
