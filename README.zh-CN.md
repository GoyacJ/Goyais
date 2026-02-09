# Goyais

简体中文 | [English](README.md)

Goyais 是一个全意图驱动、AI 原生的多模态资产编排与治理执行平台。

## 项目状态

`开发前期（设计已完善，代码开发进行中）`

当前仓库采用设计先行流程，核心契约与实施路径定义在 `docs/`。

## 项目价值

- 意图驱动操作：用户通过文本/语音对话即可触发平台行为。
- 统一执行链路：intent -> plan -> 审批/策略 -> workflow/agent run。
- AI 治理能力：RBAC、策略校验、审批闸门、审计追踪。
- 多模态资产闭环：上传/导入/处理/派生/复用，支持血缘追踪。
- 企业级观测：trace/run_event、回放、流式状态、诊断能力。
- 项目级国际化：支持按语言环境输出 API/UI/消息。

## 核心能力

- 资产系统（文件/流/结构化/文本）与不可变血缘。
- 工具/模型/算法注册与解析。
- 工作流引擎（DAG + CAS 上下文补丁）。
- Agent 运行时（plan-act-observe-recover）。
- 全平台意图编排（不仅是 AI 推理任务）。
- 策略与审批引擎（高风险动作治理）。

## 文档索引

设计文档当前以中文维护于 `docs/`：

- 总览：`docs/00-overview.md`
- 架构设计：`docs/01-architecture.md`
- 领域模型：`docs/02-domain-model.md`
- API 设计：`docs/10-api-design.md`
- 前端设计：`docs/11-frontend-design.md`
- 开发启动包：`docs/12-dev-kickoff-package.md`
- 开发计划：`docs/13-development-plan.md`
- 开发进度：`docs/14-development-progress.md`
- 开发规范：`docs/15-development-standards.md`
- 开源治理：`docs/16-open-source-governance.md`
- 国际化设计：`docs/17-internationalization-design.md`

## 开发流程

开始开发前：

1. 阅读 `docs/12-dev-kickoff-package.md`。
2. 阅读 `docs/13-development-plan.md`。
3. 阅读 `docs/15-development-standards.md`。
4. 阅读 `docs/16-open-source-governance.md`。
5. 在 `docs/14-development-progress.md` 把任务标记为 `IN_PROGRESS`。

执行规则：

- 严格按设计文档开发。
- 发现文档错误或冲突时，先修正文档（或同 PR 修正）再继续实现。

## 项目国际化（产品能力）

Goyais 支持产品级国际化，不是仅文档国际化：

- 按请求头与用户偏好进行语言环境协商。
- API 错误与消息支持本地化输出。
- 前端支持语言切换与翻译 Key 管理。
- 审批/通知/策略提示按语言环境渲染。

详见：`docs/17-internationalization-design.md`。

## 开源协作

- 许可证：Apache-2.0（`LICENSE`）
- 贡献指南：`CONTRIBUTING.md` / `CONTRIBUTING.zh-CN.md`
- 安全策略：`SECURITY.md` / `SECURITY.zh-CN.md`
- 治理模型：`GOVERNANCE.md` / `GOVERNANCE.zh-CN.md`
- 行为准则：`CODE_OF_CONDUCT.md` / `CODE_OF_CONDUCT.zh-CN.md`
- 维护者信息：`MAINTAINERS.md` / `MAINTAINERS.zh-CN.md`

## 路线图概览

- S0：API/SSE 骨架、中间件基线、事件存储基础。
- S1：Intent MVP + RBAC + 审批核心能力。
- S2：资产与工作流闭环。
- S3：全 AI 交互（文本+语音）体验完善。
- S4：稳定性、性能、安全审计与发布准备。

## 许可证

本项目基于 Apache License 2.0 开源，详见 `LICENSE`。
