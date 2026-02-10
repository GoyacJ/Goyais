---
name: goyais-thread2-bootstrap
description: 当任务处于 Thread #2 工程骨架阶段，需要按 minimal/full profile 与 single-binary 验收优先推进时触发；当任务已经进入具体模块垂直切片实施阶段时不触发。
---

# goyais-thread2-bootstrap

用于 Thread #2（工程骨架阶段）的标准提示词与执行步骤模板。

## 适用场景

- 启动工程基建线程，需要先搭骨架再进入功能切片。
- 需要明确 minimal/full profile，并把单二进制验收放在优先级前列。
- 需要给 AI/工程同学一份可复制的统一提示词。

## 非适用场景

- 已进入某个业务模块细化开发（应改用 `goyais-vertical-slice`）。
- 仅做存量功能修补，不涉及基建 bootstrap。
- 与 Thread #2 无关的常规需求讨论。

## 输入（需要哪些仓库文件）

- `README.md`
- `AGENTS.md`
- `docs/spec/v0.1.md`
- `docs/arch/overview.md`
- `docs/acceptance.md`
- `assets/thread2_prompt.md`
- `assets/thread2_execution_steps.md`

## 输出（会改哪些文件/会生成哪些文件）

- 生成 Thread #2 标准执行提示词。
- 生成 Thread #2 分步执行与验收清单。
- 为后续模块切片输出可复用前置条件。

## 严格步骤

1. 明确目标是“工程骨架”，不是业务功能完善。
2. 固定两套 profile：minimal（sqlite/memory/local/mediamtx）与 full（postgres/redis/minio/mediamtx）。
3. 先落实 single-binary 静态服务验收路径，再推进其他基础能力。
4. 执行顺序遵循：配置系统 -> Command Gate -> 授权链 -> 静态服务与验收脚本。
5. 所有写入动作仍遵循 Command-first 语义与错误模型 `messageKey`。
6. 任何与冻结约束冲突的提议都必须回退并重写方案。

## 验收方式

- 产出的 prompt 和步骤文档可直接复制执行。
- 验收清单覆盖 `/`、`/canvas`、`/api/v1/healthz`、缓存头、Content-Type、特殊路径 404。
- 不引入与权威文档冲突的新默认值或新流程。
