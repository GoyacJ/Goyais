# Goyais — 根级架构文档

## 变更记录 (Changelog)

| 日期 | 版本 | 说明 |
|---|---|---|
| 2026-03-07 | v2 | 添加 AI 协作指引总览 |
| 2026-03-07 | 初始生成 | 由架构扫描脚本自动生成，覆盖率约 37% |

---

## AI 协作指引总览 (AI Guidance Overview)

### 重构总原则

**彻底重构，零技术债，不做兼容**
- 本次重构目标：消除所有技术债，建立清晰的架构边界
- 不保留废弃代码，不做兼容性实现，不妥协设计原则
- 优先级：正确性 > 可维护性 > 性能 > 开发速度
- 类型安全优先：TypeScript 严格模式，Go 接口驱动设计

### 全局架构约束（不可违反）

**1. 契约优先 (Contract-First)**
- 所有 API 变更必须先修改 `packages/contracts/openapi.yaml`
- 运行 `pnpm contracts:generate` 生成 TypeScript 类型
- 提交前运行 `pnpm contracts:check` 验证同步
- 前端/后端类型定义以 OpenAPI 为唯一真相来源

**2. 依赖方向（单向，不可逆）**
```
apps/desktop ──┐
apps/mobile ───┼──> packages/shared-core ──> packages/contracts
               │
               └──> services/hub (HTTP API)

services/hub 内部：
httpapi -> adapters -> runtime -> extensions -> core
```

**3. 核心不可变文件（修改需架构评审）**
- `services/hub/internal/agent/core/interfaces.go` — Agent v4 核心接口
- `services/hub/internal/agent/core/statemachine/` — Run 状态机
- `packages/contracts/openapi.yaml` — API 契约规范

**4. 运行时能力分支**
- Desktop/Mobile/Web 差异通过 `isRuntimeCapabilitySupported()` 统一处理
- 禁止硬编码平台判断（如 `if (isMobile)`）
- 共享组件必须处理所有运行时目标

### 修改决策树

**我应该修改哪个模块？**

```
需求：添加新 API 端点
├─ 1. packages/contracts/openapi.yaml (定义契约)
├─ 2. pnpm contracts:generate (生成类型)
├─ 3. services/hub/internal/httpapi/ (实现后端)
└─ 4. apps/desktop/src/shared/services/ (实现前端调用)

需求：修改前端 UI
├─ 判断：Desktop 专属 or 共享？
│   ├─ Desktop 专属 → apps/desktop/src/modules/
│   └─ 共享 → apps/desktop/src/shared/ (考虑 Mobile 兼容)
└─ 添加 Vitest 测试

需求：修改后端逻辑
├─ 判断：涉及核心接口？
│   ├─ 是 → services/hub/internal/agent/core/ (需架构评审)
│   └─ 否 → services/hub/internal/agent/runtime/
└─ 添加 Go 测试

需求：修改类型定义
├─ 判断：API 相关？
│   ├─ 是 → packages/contracts/openapi.yaml
│   └─ 否 → packages/shared-core/src/api.ts (手写辅助类型)
└─ 运行 pnpm contracts:generate
```

### 全局反模式 (Global Anti-patterns)

**禁止的做法（任何模块）**
- ❌ 先写代码，后补契约（必须契约优先）
- ❌ 手动修改自动生成的文件（`openapi.ts`）
- ❌ 跨层直接调用（违反依赖方向）
- ❌ 硬编码配置（端口、URL、令牌）
- ❌ 使用 `any` 类型逃避类型检查
- ❌ 在组件中直接调用 HTTP API（必须通过 Pinia store）
- ❌ 硬编码颜色值（必须用 CSS token）
- ❌ 提交前不运行质量门禁

**常见错误模式**
- 忘记运行 `pnpm contracts:generate` 导致类型过期
- 修改 `openapi.yaml` 后未同步更新 Hub 实现
- 在共享组件中使用 Desktop 专属 API
- 测试中未 mock Tauri/Hub 依赖
- 错误处理未包含 `traceId`（影响调试）

### 跨模块协调规则

**API 变更流程（涉及 3 个模块）**
1. `packages/contracts` — 修改 `openapi.yaml`
2. `packages/shared-core` — 运行 `contracts:generate`
3. `services/hub` — 实现 Go handler
4. `apps/desktop` — 实现前端调用
5. 全局 — 运行 `pnpm contracts:check` + `make test`

**前端组件共享（Desktop ↔ Mobile）**
1. 共享组件放在 `apps/desktop/src/shared/`
2. 使用 `isRuntimeCapabilitySupported()` 处理差异
3. 在 `mobile-runtime.spec.ts` 添加测试
4. 避免使用 Desktop 专属 Tauri 插件

**后端分层协调（Hub 内部）**
1. `core` 定义接口（零外部依赖）
2. `runtime` 实现业务逻辑
3. `adapters` 适配协议（HTTP/ACP/CLI）
4. `httpapi` 暴露 REST 端点

### 质量门禁（提交前必过）

**自动化检查**
```bash
# 契约同步检查
pnpm contracts:check

# TypeScript 类型检查
pnpm lint

# 单元测试
pnpm test

# 覆盖率门禁
pnpm coverage:gate

# 质量门禁（文件大小 + 圈复杂度）
pnpm quality:gate

# CSS Token 漂移检查
pnpm check:tokens

# Go 测试
make test-hub

# Go vet
make lint-hub

# E2E 烟雾测试
pnpm e2e:smoke
```

**手动检查清单**
- [ ] 是否修改了 P0 级文件？（需架构评审）
- [ ] 是否违反了依赖方向？
- [ ] 是否添加了对应的测试？
- [ ] 是否更新了相关文档？
- [ ] 是否考虑了 Mobile 兼容性？（如修改共享组件）
- [ ] 是否包含 `traceId` 用于错误追踪？

### 模块优先级与影响范围

**P0: 核心架构（修改影响全局）**
- `packages/contracts/openapi.yaml`
- `services/hub/internal/agent/core/interfaces.go`
- `apps/desktop/src/router/index.ts`
- `apps/desktop/src/shared/runtime/index.ts`

**P1: 关键实现（修改影响单模块）**
- `services/hub/internal/agent/runtime/`
- `apps/desktop/src/modules/session/store/`
- `apps/desktop/src/shared/services/http.ts`

**P2: 辅助代码（修改影响局部）**
- `services/hub/internal/httpapi/handlers_*.go`
- `apps/desktop/src/modules/*/views/*.vue`
- `apps/desktop/src/shared/ui/*.vue`

### 模块文档索引

每个模块都有详细的 AI 协作指引，包含：
- 模块特定的重构原则
- 不可变文件列表
- 常见任务决策树
- 反模式与常见错误
- 文件优先级
- 跨模块协调规则

**查看模块文档：**
- [services/hub](./services/hub/CLAUDE.md) — Go 后端服务
- [packages/contracts](./packages/contracts/CLAUDE.md) — OpenAPI 契约
- [packages/shared-core](./packages/shared-core/CLAUDE.md) — 共享类型
- [apps/desktop](./apps/desktop/CLAUDE.md) — 桌面客户端
- [apps/mobile](./apps/mobile/CLAUDE.md) — 移动客户端

### 快速参考

**我需要...**
- 添加新 API → 先看 [contracts](./packages/contracts/CLAUDE.md)
- 修改后端逻辑 → 先看 [hub](./services/hub/CLAUDE.md)
- 修改前端 UI → 先看 [desktop](./apps/desktop/CLAUDE.md)
- 处理类型定义 → 先看 [shared-core](./packages/shared-core/CLAUDE.md)
- 适配移动端 → 先看 [mobile](./apps/mobile/CLAUDE.md)

---

## 项目愿景

**Goyais** 是一套面向开发者的 AI 协作开发产品（v0.4.0，MIT 协议）。以会话驱动开发流程的 AI 开发工作台，围绕 `Workspace -> Project -> Session -> Run -> ChangeSet` 组织从需求输入、执行编排到变更交付的完整链路。

- 主入口：Desktop（Tauri + Vue 3 桌面客户端）
- 移动访问：Mobile（Tauri Mobile + Vue 3，仅远程工作区）
- 控制面：Hub（Go 后端服务，统一认证/执行/治理）

---

## 架构总览

```
Goyais Monorepo (pnpm + Turborepo)
├── apps/
│   ├── desktop/        # Tauri 桌面端，主产品入口
│   └── mobile/         # Tauri Mobile，远程工作区访问
├── packages/
│   ├── shared-core/    # 共享 TS 类型与 API 工具（发布为 @goyais/shared-core）
│   └── contracts/      # OpenAPI 规范（openapi.yaml，默认端口 8787）
├── services/
│   └── hub/            # Go 控制面服务（主二进制 + ACP sidecar + CLI）
├── docs/
│   ├── site/           # VitePress 文档站
│   ├── slides/         # Slidev 演示文稿
│   └── PRD.md          # 产品需求文档
├── scripts/            # 构建、质量检查、烟雾测试脚本
└── Makefile            # 快速开发命令入口
```

核心数据流：
1. Desktop/Mobile 通过 HTTP（本地 sidecar 或远程 Hub）调用 Hub REST API
2. Hub 提供 `/v1/auth`、`/v1/workspaces`、`/v1/sessions`、`/v1/runs` 等接口
3. Hub 内部 Agent Runtime（Go）执行 AI 模型调用，并通过 SSE 推送事件到客户端
4. `goyais-acp` sidecar 通过 stdio JSON-RPC（ACP 协议）与 Desktop Tauri shell 通信

---

## 模块结构图

```mermaid
graph TD
    Root["(根) Goyais v0.4.0"] --> Apps["apps/"];
    Root --> Pkgs["packages/"];
    Root --> Svc["services/"];
    Root --> Docs["docs/"];

    Apps --> Desktop["desktop"];
    Apps --> Mobile["mobile"];

    Pkgs --> Core["shared-core"];
    Pkgs --> Contracts["contracts"];

    Svc --> Hub["hub"];

    Docs --> Site["site (VitePress)"];
    Docs --> Slides["slides (Slidev)"];

    click Desktop "./apps/desktop/CLAUDE.md" "查看 desktop 模块文档"
    click Mobile "./apps/mobile/CLAUDE.md" "查看 mobile 模块文档"
    click Core "./packages/shared-core/CLAUDE.md" "查看 shared-core 模块文档"
    click Contracts "./packages/contracts/CLAUDE.md" "查看 contracts 模块文档"
    click Hub "./services/hub/CLAUDE.md" "查看 hub 模块文档"
```

---

## 模块索引

| 模块路径 | 包名 | 语言 | 一句话职责 |
|---|---|---|---|
| [apps/desktop](./apps/desktop/CLAUDE.md) | `@goyais/desktop` | TS + Vue 3 + Rust | Tauri 桌面端，本地/远程工作区主入口，会话执行与变更审阅 |
| [apps/mobile](./apps/mobile/CLAUDE.md) | `@goyais/mobile` | TS + Vue 3 + Rust | Tauri Mobile，远程工作区访问与轻量会话查看（iOS/Android） |
| [packages/shared-core](./packages/shared-core/CLAUDE.md) | `@goyais/shared-core` | TypeScript | 共享 API 类型定义与 OpenAPI 生成类型，供 desktop/mobile 消费 |
| [packages/contracts](./packages/contracts/CLAUDE.md) | contracts | YAML (OpenAPI 3.1) | Hub REST API 契约规范，驱动 shared-core 类型生成 |
| [services/hub](./services/hub/CLAUDE.md) | `goyais/services/hub` | Go 1.24 | 控制面服务：认证、工作区、Session/Run 执行、资源配置、管理审计 |

---

## 运行与开发

### 前置条件

- pnpm 10.11.0（`package.json` `packageManager` 字段锁定）
- Go 1.24+
- Rust（用于 Tauri 编译）
- Node.js LTS

### 快速启动（Makefile）

```bash
# 查看所有可用命令（含端口说明）
make dev

# 仅启动 Hub 后端（需要 HUB_INTERNAL_TOKEN 环境变量）
HUB_INTERNAL_TOKEN=xxx make dev-hub

# 启动 Desktop Web 预览（不含 Tauri shell）
make dev-web

# 完整 Desktop Tauri 开发模式（含 sidecar 构建）
make dev-desktop
```

### pnpm Turborepo 脚本

```bash
# 默认：启动 Desktop Vite dev server
pnpm dev

# Mobile Web 预览
pnpm dev:mobile

# 构建 Desktop
pnpm build

# 全量测试（Hub + Desktop）
make test

# Hub + Desktop 契约检查
pnpm contracts:check
```

### Hub 关键环境变量

| 环境变量 | 说明 | 默认值 |
|---|---|---|
| `PORT` | Hub 监听端口 | `8787` |
| `HUB_INTERNAL_TOKEN` | 内部令牌（开发必填） | 无 |

### Desktop / Mobile Vite 环境变量

| 变量 | 说明 |
|---|---|
| `VITE_RUNTIME_TARGET` | `desktop` / `mobile` / `web` |
| `VITE_HUB_BASE_URL` | Hub 地址（desktop 默认 `http://127.0.0.1:8787`） |
| `VITE_REQUIRE_HTTPS_HUB` | 强制 HTTPS（mobile 生产构建自动开启） |
| `VITE_ALLOW_INSECURE_HUB` | 允许 HTTP Hub（开发调试用） |

---

## 测试策略

| 层级 | 工具 | 命令 |
|---|---|---|
| Desktop 单元/集成 | Vitest 3 + jsdom + @vue/test-utils | `pnpm test` / `make test-desktop` |
| Hub 单元/集成 | Go testing (`go test ./...`) | `make test-hub` |
| E2E Smoke | Playwright | `pnpm e2e:smoke` |
| 覆盖率门禁 | Vitest coverage-v8 + 自定义阈值脚本 | `pnpm coverage:gate` |
| 质量门禁 | 文件大小 + 圈复杂度检查 | `pnpm quality:gate` |
| CSS Token 漂移 | 自定义脚本 | `pnpm check:tokens` |
| 契约同步检查 | openapi-typescript `--check` | `pnpm contracts:check` |
| 端到端健康检查 | Shell smoke 脚本 | `make health` |

---

## 编码规范

- **TypeScript**：严格模式，`tsc --noEmit` 作为 lint。避免 `any` 冒泡到 store 层
- **Vue 3**：Composition API + `<script setup>` 优先；Pinia store 按功能模块组织（`session/store`、`workspace/store` 等）
- **Go**：`go vet ./...` 检查；包按 Clean Architecture 分层（`core` 纯接口 -> `adapters` 实现 -> `runtime` 编排 -> `httpapi` 路由）；`core` 包不得有除标准库外的任何依赖
- **API 契约优先**：先改 `packages/contracts/openapi.yaml`，再跑 `pnpm contracts:generate` 生成 TS 类型，最后实现
- **Token 设计系统**：CSS 设计 token 通过 `check:tokens` 防止漂移，不直接写 hex/rgb 颜色值

---

## AI 使用指引

- `services/hub/internal/agent/core/interfaces.go` 是 Agent v4 架构的不可变锚点，修改需显式版本升级
- `packages/contracts/openapi.yaml` 是唯一真相来源，所有对象字段/状态枚举以此为准
- Desktop 前端功能修改需同时考虑 `desktop` 与 `mobile` 运行时目标分支（`isRuntimeCapabilitySupported()`）
- Hub 分层依赖方向：`httpapi` -> `internal/agent/*` -> `core`，禁止反向依赖
- ACP sidecar（`goyais-acp`）通过 stdio JSON-RPC 与 Tauri shell 通信，不直接暴露 HTTP

---

## 核心对象模型速查

```
Workspace (mode=local/remote, hub_url, auth_mode)
  └── Project (repo_path, is_git, 模型配置, Token 阈值)
        └── Session (queue_state: idle/running/queued, rule_ids, skill_ids, mcp_ids)
              └── Run (state: queued→pending→executing→confirming→awaiting_input→completed/failed/cancelled)
                    └── ChangeSet (entries, capability, suggested_message, project_kind=git/non_git)

ResourceConfig     (type=model/rule/skill/mcp, 工作区级)
ProjectConfig      (项目级资源绑定与阈值)
WorkspaceAgentConfig (Agent 默认行为: 轮次/Trace/预算/MCP)
PermissionSnapshot (角色 + 权限集合 + 菜单可见性 + 动作可见性)
```
