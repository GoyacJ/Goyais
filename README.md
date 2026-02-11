<p align="center">
  <img src="./logo.png" alt="Goyais Logo" width="140" />
</p>

<h1 align="center">Goyais</h1>

<p align="center">
  <strong>全意图驱动的智能 Agent + 多模态 AI 原生编排与执行平台（Go + Vue）</strong><br/>
  <strong>An intent-driven intelligent Agent and multimodal AI-native orchestration platform (Go + Vue)</strong>
</p>

<p align="center">
  <img alt="License" src="https://img.shields.io/badge/license-Apache%202.0-1677ff" />
  <img alt="Go" src="https://img.shields.io/badge/Go-1.24.3-00ADD8?logo=go&logoColor=white" />
  <img alt="Node" src="https://img.shields.io/badge/Node-20%2B-339933?logo=nodedotjs&logoColor=white" />
  <img alt="pnpm" src="https://img.shields.io/badge/pnpm-10.x-F69220?logo=pnpm&logoColor=white" />
  <img alt="Milestone" src="https://img.shields.io/badge/milestone-v0.1-6f42c1" />
</p>

<p align="center">
  <a href="#highlights">功能特性</a> |
  <a href="#architecture">系统架构</a> |
  <a href="#quick-start">快速开始</a> |
  <a href="#docs-nav">文档导航</a> |
  <a href="#contributing">贡献协作</a>
</p>

<a id="highlights"></a>
## 功能特性 | Highlights

Goyais 面向企业级与开源社区，工程治理目标对齐 Apache 顶级项目标准（治理透明、契约稳定、可审计交付）。  
Goyais targets enterprise and open-source communities with Apache-grade engineering governance (transparent governance, stable contracts, auditable delivery).

| 能力域 | 说明 |
|---|---|
| 意图执行 | 自然语言/语音 -> Command -> Workflow -> 产物沉淀 |
| DAG 编排 | 可视化编排、强校验、运行调试与回放 |
| 能力生态 | Tool / Skill / MCP / Model / Algorithm 统一注册治理 |
| 多模态资产 | 视频/图片/音频/文档/表格的统一资产化与血缘追踪 |
| 流媒体集成 | MediaMTX 控制面 + 录制资产化 + 事件触发工作流 |
| 安全治理 | Agent-as-User、Visibility+ACL、Egress Gate、全链路审计 |

<a id="architecture"></a>
## 系统架构 | Architecture

### PRD 对齐目标（v0.1） | PRD-Aligned Goals (v0.1)

依据 `docs/prd.md`，v0.1 核心交付：

1. AI 与 UI 双入口一致，统一通过 Command 执行（Command-first）。
2. 复杂可视化编排画布（DAG）可构建、校验、运行、回放。
3. Tool/Skill/MCP/Model/Algorithm 统一能力体系。
4. 插件市场 MVP（上传/安装/启停/升级/回滚）。
5. 完整资产体系（多模态管理、元数据、血缘）。
6. MediaMTX 流媒体接入与事件触发工作流。
7. Agent-as-User + RBAC/ACL/Visibility + Egress Gate。
8. 算法库 MVP（至少 2 个算法包可运行并产出结构化结果+资产）。

### 核心工程原则 | Core Engineering Principles

- `Command-first`: 副作用动作必须通过 `POST /api/v1/commands`
- `Agent-as-User`: AI 永远代表当前登录用户执行
- `Visibility + ACL + Egress`: 全对象可见性/共享与外发闸门统一治理
- `Contract Sync`: 接口/模型/状态机变更必须同步契约文档
- `Single Binary`: 生产发布支持 Go 单二进制内嵌前端静态资源

### 仓库结构 | Repository Layout

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

### 模块状态 | Module Status

| 模块 Module | 状态 Status | 说明 Notes |
|---|---|---|
| `go_server` | 迭代中 Iterating | Go 主后端实现，`/api/v1` 与单二进制发布路径 |
| `vue_web` | 迭代中 Iterating | Vue + Vite + TS 前端，含画布与运行态界面 |
| `java_server` | 设计中 Design-phase | 设计期门禁约束，未进入主实现 |
| `python_server` | 设计中 Design-phase | 设计期门禁约束，未进入主实现 |
| `flutter_mobile` | 设计中 Design-phase | 移动端方案设计阶段 |
| `android_mobile` | 预留 Reserved | Android 端预留模块 |

<a id="quick-start"></a>
## 快速开始 | Quick Start

### 1) 环境要求 | Prerequisites

- Go `1.24.3+`
- Node.js `20+`
- `pnpm@10+`
- GNU Make

### 2) 安装依赖 | Install dependencies

```bash
pnpm -C vue_web install --frozen-lockfile
```

### 3) 运行质量门禁 | Run quality gates

```bash
bash go_server/scripts/ci/contract_regression.sh
```

该脚本统一执行：
- worktree 审计
- merged-thread cleanup 审计（warn-only）
- precommit 防呆检查
- 路径迁移审计
- 源码头校验（SPDX/Author/Created/Version/Description）
- Go/Vue 测试与类型检查
- 单二进制构建与验证

### 4) 构建并运行单二进制 | Build and run single binary

```bash
make -C go_server build
GOYAIS_SERVER_ADDR=:18080 ./go_server/build/goyais
```

常用接口 | Useful endpoints:
- `GET /api/v1/healthz`
- `POST /api/v1/commands`

### 5) 完整依赖模式（可选） | Full mode dependencies (optional)

可使用 `docker-compose.full.yml` 启动完整依赖（Postgres/Redis/MinIO/MediaMTX）进行集成验证。

<a id="docs-nav"></a>
## 文档导航 | Documentation

### 核心入口 | Core entry points

- 产品需求基线 | PRD: `docs/prd.md`
- 根级工程规范 | Root engineering charter: `AGENTS.md`
- Go 技术契约 | Go contracts:
  - `go_server/docs/api/openapi.yaml`
  - `go_server/docs/arch/overview.md`
  - `go_server/docs/arch/data-model.md`
  - `go_server/docs/arch/state-machines.md`
  - `go_server/docs/acceptance.md`
- Web 规范 | Web specification: `vue_web/docs/web-ui.md`

### 开源治理文档 | Open-source governance docs

- `CONTRIBUTING.md`
- `CODE_OF_CONDUCT.md`
- `SECURITY.md`
- `SUPPORT.md`
- `.github/ISSUE_TEMPLATE/`
- `.github/pull_request_template.md`

<a id="contributing"></a>
## 贡献协作 | Contributing

开始贡献前，请先阅读：

- `CONTRIBUTING.md`
- `AGENTS.md`
- `docs/prd.md`
- `go_server/docs/acceptance.md`

工作流强约束（简版）：
- 分支前缀：`goya/<thread-id>-<topic>`
- 一线程一 worktree（默认在 `<repo>/.worktrees/`）
- 线程开启必须执行：`bash .agents/skills/goyais-worktree-flow/scripts/create_worktree.sh --topic <topic>`
- 线程收口必须执行：`bash .agents/skills/goyais-worktree-flow/scripts/merge_thread.sh --thread-branch <goya/...>`
- 禁止手工 `git merge` / `git branch -d` / `git worktree remove` 绕过标准收口
- 提交前执行：`bash go_server/scripts/git/precommit_guard.sh`

## License

Apache License 2.0，见 `LICENSE`。  
Licensed under Apache-2.0. See `LICENSE`.
