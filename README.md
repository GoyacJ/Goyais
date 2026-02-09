# Goyais

Goyais 是一个以 AI 为主入口、同时支持可视化编排的多模态执行平台。当前仓库处于 **v0.1 文档初始化阶段**，本阶段只交付架构、接口、数据模型与验收规范，不包含业务实现代码。

## v0.1 目标

- AI 与 UI 双入口一致（Command-first）。
- 统一权限与隔离（Agent-as-User + Visibility/ACL + Egress）。
- 资产、工作流、插件、流媒体基础契约完整。
- 前端约束明确：Vue + Vite + TypeScript + TailwindCSS + vue-i18n（zh-CN/en-US）+ 深浅色切换。
- 生产发布冻结为单二进制（Go embed 前端 dist）。

## 文档地图

- 权威需求：`/Users/goya/Repo/Git/Goyais/docs/prd.md`
- 协作规则：`/Users/goya/Repo/Git/Goyais/AGENTS.md`
- 架构总览：`/Users/goya/Repo/Git/Goyais/docs/arch/overview.md`
- 数据模型：`/Users/goya/Repo/Git/Goyais/docs/arch/data-model.md`
- 状态机：`/Users/goya/Repo/Git/Goyais/docs/arch/state-machines.md`
- API 契约：`/Users/goya/Repo/Git/Goyais/docs/api/openapi.yaml`
- 规格拆解：`/Users/goya/Repo/Git/Goyais/docs/spec/v0.1.md`
- 验收清单：`/Users/goya/Repo/Git/Goyais/docs/acceptance.md`

## 运行模式（文档约定）

## 1) 最小化运行模式（v0.1 必须可闭环）

组合：
- SQLite
- MediaMTX
- 本地文件存储（local）
- 本地缓存（memory）

默认配置目标：
- `db.driver=sqlite`
- `cache.provider=memory`
- `vector.provider=sqlite`
- `object_store.provider=local`
- `stream.provider=mediamtx`

## 2) 完整模式（推荐）

组合：
- PostgreSQL
- Redis（缓存）+ Redis Stack（向量）
- MinIO（或 S3）
- MediaMTX

## 单二进制发布策略（冻结）

- 生产发布必须为单二进制：Go embed 前端 `dist`。
- API 路径固定 `/api/v1/*`。
- 其余前端路由走 SPA fallback 到 `index.html`。
- `index.html` 响应头固定：`Cache-Control: no-store`。
- 静态资源必须返回正确 `Content-Type`。
- `/favicon.ico` 与 `/robots.txt` 在无占位文件时默认返回 404，不走 fallback。
- 开发模式可采用 Vite dev + proxy。

## 构建流程（文档层）

- 标准构建入口：`make build`
- 验收要求：构建后删除/改名 `web/dist`（甚至 `web/`）仍可运行 `/`、`/canvas`、`/api/v1/healthz`。

## 后续实现顺序（Thread #2 建议）

1. 配置加载与 provider 工厂（ENV > YAML > 默认值）。
2. Command Gate + 统一错误模型 + 审计。
3. Visibility/ACL + Egress 授权链。
4. Asset + Workflow 闭环（模板、运行、回放、血缘）。
5. Registry + Plugin Market。
6. MediaMTX 控制面 + 录制资产化 + 事件触发。
7. 前端基线（i18n + theme）与 single-binary 静态服务整合。
