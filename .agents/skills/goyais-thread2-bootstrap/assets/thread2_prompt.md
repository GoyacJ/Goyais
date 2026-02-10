# Thread #2 标准 Prompt 模板

你正在实现 Goyais Thread #2（工程骨架阶段）。

## 目标

- 建立可持续扩展的工程骨架。
- 同时支持 minimal 与 full profile。
- 优先通过 single-binary 静态服务验收。

## 范围

- in scope：配置加载、provider 选择、命令闸门骨架、授权链路骨架、静态服务路由与缓存策略。
- out of scope：具体业务功能完善、非阻断 UI 细节优化。

## 硬约束

- Command-first：副作用动作统一可表达为 Command，入口 `/api/v1/commands`。
- Agent-as-User：保留执行上下文并做双闸门授权校验。
- 配置优先级：`ENV > YAML > default`。
- 发布形态：single binary（Go embed dist）。
- 静态策略：`index.html` 必须 `Cache-Control: no-store`；`favicon/robots` 缺省 404。
- 错误模型：`error: { code, messageKey, details }`。

## 产出

- 工程骨架实现清单。
- 验收执行记录（特别是 single-binary）。
- 文档同步清单（如触及契约）。

## 测试与验收

- minimal/full profile 基线验证。
- `/`、`/canvas`、`/api/v1/healthz` 连通。
- 缓存头与 Content-Type 验证。
- `/favicon.ico` 与 `/robots.txt` 缺省 404 验证。
