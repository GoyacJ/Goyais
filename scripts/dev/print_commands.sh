#!/usr/bin/env bash
set -euo pipefail

hub_port="${1:-8787}"
worker_port="${2:-8788}"
desktop_port="${3:-5173}"

cat <<EOT
Goyais v0.4.0 开发启动命令（分别在三个终端执行）

0) 先在每个终端设置同一个内部通信 token:
   export HUB_INTERNAL_TOKEN='<same-random-token>'

1) Hub:
   PORT=${hub_port} make dev-hub

2) Worker:
   PORT=${worker_port} make dev-worker

3) Desktop Client (Tauri):
   DESKTOP_PORT=${desktop_port} make dev-desktop

4) Desktop Web (Vite, optional):
   DESKTOP_PORT=${desktop_port} make dev-web

健康检查:
   curl http://127.0.0.1:${hub_port}/health
   curl http://127.0.0.1:${worker_port}/health
EOT
