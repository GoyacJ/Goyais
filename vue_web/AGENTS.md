# vue_web AGENTS

本文件定义 Vue Web 模块的实现级约束，优先级低于根 `AGENTS.md`。

## 1. UI 规范

- 必须遵循 `vue_web/docs/web-ui.md`。
- 禁止组件内硬编码语义色值，使用 design token。
- 复用全局状态 hook：`ui-pressable/ui-focus-ring/ui-disabled/ui-loading`。

## 2. 布局与交互

- 页面需保证 panel 能拖拽、缩放、全屏与恢复。
- 布局模式固定为 `console`（console-only），禁止引入 `topnav/focus` 新路径。
- 窗口状态按 `route + layout` 持久化且互不污染。

## 3. i18n 与主题

- 必须维护 `zh-CN` 与 `en-US`。
- 错误展示基于后端 `messageKey` 映射。
- 需支持 light/dark + density。

## 4. Quality Gates

- `pnpm -C vue_web typecheck`
- `pnpm -C vue_web test:run`
- `pnpm -C vue_web build`
- `pnpm -C vue_web run assets:validate`
