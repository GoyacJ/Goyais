# Goyais

Goyais 是一个以 AI 为主入口、同时支持可视化编排的多模态执行平台。当前仓库已合入 Thread #2/#3/#4/#5 的阶段成果，正在进行“契约收敛 + API 可达性修复”。

## 当前冻结约束（v0.1）

- 生产发布必须是单二进制（Go embed 前端 dist）。
- 路由优先级固定：`/api/v1/*` > 静态文件 > `favicon/robots` 缺省 404 > SPA fallback(`index.html`)。
- `index.html`（`/` 与 fallback）返回 `Cache-Control: no-store`。
- `assets/*.js` 返回正确 JS Content-Type。
- 配置优先级：`ENV > YAML > 默认值`，ENV 前缀 `GOYAIS_`，YAML `snake_case`。

## 前端包管理（固定 pnpm）

- 本仓库前端仅使用 pnpm。
- 锁文件：`/Users/goya/Repo/Git/Goyais/web/pnpm-lock.yaml`
- 不使用 `package-lock.json`。

常用命令：

```bash
pnpm -C web install --frozen-lockfile
pnpm -C web build
pnpm -C web dev
```

## 最小化模式（本次主验收）

默认配置（minimal）：
- db: `sqlite`
- cache: `memory`
- vector: `sqlite`
- object_store: `local`
- stream: `mediamtx`

运行与验收：

```bash
make build
bash .agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh
```

## API 落地状态（当前）

- 可用：`/api/v1/commands*`、`/api/v1/shares*`、`/api/v1/assets*`（其中写接口为 Command-first sugar）。
- 占位：`/api/v1/workflow-*`、`/api/v1/registry-*`、`/api/v1/plugin-market-*`、`/api/v1/streams*`，统一返回 `501 NOT_IMPLEMENTED`。

## healthz

接口：`GET /api/v1/healthz`

响应包含：
- `status`
- `timestamp`
- `version`
- `mode`
- `providers`

其中 `version` 来自构建注入（无注入时默认 `dev`）。

## full profile（compose 占位）

本次新增 `docker-compose.full.yml`，用于提供 full profile 所需依赖占位：
- postgres
- redis
- minio
- mediamtx

校验命令：

```bash
docker compose -f docker-compose.full.yml config
```

说明：full compose 在后续垂直切片阶段逐步完善；本次仅保证服务定义与连接参数可用，不承诺业务链路完全跑通。

## 文档与契约

- 需求：`/Users/goya/Repo/Git/Goyais/docs/prd.md`
- 架构：`/Users/goya/Repo/Git/Goyais/docs/arch/overview.md`
- API：`/Users/goya/Repo/Git/Goyais/docs/api/openapi.yaml`
- 验收：`/Users/goya/Repo/Git/Goyais/docs/acceptance.md`
- 仓库规则：`/Users/goya/Repo/Git/Goyais/AGENTS.md`
