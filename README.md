# Goyais

> 全意图驱动的智能 Agent + 多模态 AI 原生编排与执行平台（Go + Vue）  
> An intent-driven intelligent Agent and multimodal AI-native orchestration platform (Go + Vue).

## 1. 项目简介 | Overview

`Goyais` 面向企业级与开源社区，目标对齐 Apache 顶级项目标准（治理透明、契约稳定、可审计交付）。

Goyais is designed for enterprise and open-source communities, with engineering governance aligned to Apache top-level standards (transparent governance, stable contracts, auditable delivery).

核心场景 | Key scenarios:
- AI 意图驱动执行：自然语言/语音 -> 命令 -> 工作流执行 -> 结果沉淀  
  AI intent-driven execution: natural language/voice -> commands -> workflow execution -> persisted outputs
- 可视化 DAG 编排：强校验、可调试、可回放  
  Visual DAG orchestration: strict validation, debugging, replay
- 统一能力生态：Tool / Skill / MCP / Model / Algorithm 统一注册与治理  
  Unified capability ecosystem: Tool / Skill / MCP / Model / Algorithm under one registry and governance
- 多模态与流媒体处理：Asset + MediaMTX + Event-driven workflow  
  Multimodal and streaming processing: Asset + MediaMTX + event-driven workflow

## 2. PRD 对齐目标（v0.1） | PRD-Aligned Goals (v0.1)

依据 `docs/prd.md`，v0.1 必达目标如下：

Based on `docs/prd.md`, v0.1 must deliver:

1. AI 与 UI 双入口一致，统一通过 Command 执行（Command-first）。
2. 复杂可视化编排画布（DAG）可构建、校验、运行、回放。
3. Tool/Skill/MCP/Model/Algorithm 统一能力体系。
4. 插件市场 MVP（上传/安装/启停/升级/回滚）。
5. 完整资产体系（多模态管理、元数据、血缘）。
6. MediaMTX 流媒体接入与事件触发工作流。
7. Agent-as-User + RBAC/ACL/Visibility + Egress Gate。
8. 算法库 MVP（至少 2 个算法包可运行并产出结构化结果+资产）。

## 3. 核心工程原则 | Core Engineering Principles

跨仓强约束见 `AGENTS.md`。关键原则如下：

Cross-repo hard constraints are defined in `AGENTS.md`. Key principles:

- `Command-first`: 副作用动作必须通过 `POST /api/v1/commands`
- `Agent-as-User`: AI 永远代表当前登录用户执行
- `Visibility + ACL + Egress`: 全对象可见性/共享与外发闸门统一治理
- `Contract Sync`: 接口/模型/状态机变更必须同步契约文档
- `Single Binary`: 生产发布支持 Go 单二进制内嵌前端静态资源

## 4. 仓库结构 | Repository Layout

```text
goyais/
├── docs/                  # 业务与治理文档 (Business/Governance)
├── go_server/             # Go 服务端实现与契约 (Go backend + contracts)
├── vue_web/               # Vue Web 前端实现 (Vue frontend)
├── java_server/           # Java 服务端设计期模块 (design-phase)
├── python_server/         # Python 服务端设计期模块 (design-phase)
├── flutter_mobile/        # Flutter 移动端设计期模块 (design-phase)
├── .agents/rules/         # 稳定规则 (stable rules)
└── .agents/skills/        # 可复用工作流技能 (reusable skills)
```

## 5. 模块状态 | Module Status

| 模块 Module | 状态 Status | 说明 Notes |
|---|---|---|
| `go_server` | 迭代中 Iterating | Go 主后端实现，`/api/v1` 与单二进制发布路径 |
| `vue_web` | 迭代中 Iterating | Vue + Vite + TS 前端，含画布与运行态界面 |
| `java_server` | 设计中 Design-phase | 设计期门禁约束，未进入主实现 |
| `python_server` | 设计中 Design-phase | 设计期门禁约束，未进入主实现 |
| `flutter_mobile` | 设计中 Design-phase | 移动端方案设计阶段 |
| `android_mobile` | 预留 Reserved | Android 端预留模块 |

## 6. 快速开始（最小闭环） | Quick Start (Minimal Loop)

### 6.1 环境要求 | Prerequisites

- Go `1.24.3+`
- Node.js `20+`
- `pnpm@10+`
- GNU Make

### 6.2 安装依赖 | Install dependencies

```bash
pnpm -C vue_web install --frozen-lockfile
```

### 6.3 运行质量门禁 | Run quality gates

```bash
bash go_server/scripts/ci/contract_regression.sh
```

该脚本会执行：
- worktree 审计
- precommit 防呆检查
- 路径迁移审计
- 源码头校验（SPDX/Author/Created/Version/Description）
- Go/Vue 测试与类型检查
- 单二进制构建与验证

### 6.4 构建并运行单二进制 | Build and run single binary

```bash
make -C go_server build
GOYAIS_SERVER_ADDR=:18080 ./go_server/build/goyais
```

常用检查接口 | Useful endpoints:
- `GET /api/v1/healthz`
- `POST /api/v1/commands`

## 7. 完整模式依赖（可选） | Full Mode Dependencies (Optional)

可使用仓库内 `docker-compose.full.yml` 启动完整依赖（如 Postgres/Redis/MinIO/MediaMTX）进行集成验证。

Use `docker-compose.full.yml` to bring up full dependencies (for example Postgres/Redis/MinIO/MediaMTX) for integration validation.

## 8. 贡献与协作 | Contributing

开始贡献前，请先阅读：

Before contributing, read:
- `CONTRIBUTING.md`
- `AGENTS.md`
- `docs/prd.md`
- `go_server/docs/acceptance.md`

工作流强约束（简版） | Workflow hard constraints (short):
- 分支前缀：`goya/<thread-id>-<topic>`
- 一线程一 worktree
- 提交前执行：`bash go_server/scripts/git/precommit_guard.sh`

## 9. 开源治理文档 | Open Source Governance Docs

- Code of Conduct: `CODE_OF_CONDUCT.md`
- Security Policy: `SECURITY.md`
- Support Guide: `SUPPORT.md`
- Issue Templates: `.github/ISSUE_TEMPLATE/`
- PR Template: `.github/pull_request_template.md`

## 10. License

Apache License 2.0，见 `LICENSE`。

Licensed under Apache-2.0. See `LICENSE`.
