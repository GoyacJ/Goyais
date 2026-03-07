#!/usr/bin/env bash
set -euo pipefail

hub_port="${1:-8787}"
desktop_port="${2:-5173}"

cat <<EOT
Goyais v0.4.0 开发启动命令（分别在两个终端执行）

0) 先在每个终端设置同一个内部通信 token:
   export HUB_INTERNAL_TOKEN='<same-random-token>'

1) Hub:
   PORT=${hub_port} make dev-hub

2) Desktop Client (Tauri):
   DESKTOP_PORT=${desktop_port} make dev-desktop

3) Desktop Web (Vite, optional):
   DESKTOP_PORT=${desktop_port} make dev-web

健康检查:
   curl http://127.0.0.1:${hub_port}/health
EOT
