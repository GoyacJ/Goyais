---
name: goyais-single-binary-acceptance
description: 当任务需要验证单二进制发布能力及静态路由、缓存头、Content-Type、删除 dist 后运行能力时触发；当任务与发布形态和静态服务行为无关时不触发。
---

# goyais-single-binary-acceptance

将 `docs/acceptance.md` 的 single binary 验收步骤固化为可重复执行脚本与说明。

## 适用场景

- 需要验证 `make build` 后的单可执行文件是否可独立服务。
- 需要验证 `/`、`/canvas`、`/api/v1/healthz`、缓存头、静态资源类型、特殊路径 404。
- 需要在 CI 或本地重复执行同一套验收动作。

## 非适用场景

- 纯后端业务逻辑验证，不涉及静态服务和发布形态。
- 仅做文档编辑，无需运行可执行验收。
- 与 Goyais single-binary 约束无关的脚本任务。

## 输入（需要哪些仓库文件）

- `AGENTS.md`
- `docs/arch/overview.md`
- `docs/spec/v0.1.md`
- `docs/acceptance.md`
- `assets/verify_single_binary_usage.md`
- `scripts/verify_single_binary.sh`

## 输出（会改哪些文件/会生成哪些文件）

- 输出脚本执行日志与通过/失败结论。
- 输出失败项定位信息（状态码、header、资源类型）。
- 必要时更新本 skill 的脚本与使用说明。

## 严格步骤

1. 执行 `scripts/verify_single_binary.sh`，脚本负责构建、探测二进制、启动服务、验证关键路由与 header。
2. 二进制探测策略固定：候选目录、排除规则、按新增/mtime 最新选择。
3. 验证 `web/dist` 缺失场景下服务能力，确保不是“依赖源码目录的伪通过”。
4. 验证 `/` 与 `/canvas` 的 `Cache-Control` 精确为 `no-store`，并验证 `Content-Type`。
5. 验证 `favicon/robots` 缺省 404，不走 SPA fallback。
6. 任一校验失败返回非 0，退出码语义固定。

## 验收方式

- 脚本可重复执行，重复运行不破坏工作区。
- 退出码满足约定：`0/1/2/3/4`。
- 结果可映射回 `docs/acceptance.md` 第 4 节条目。
