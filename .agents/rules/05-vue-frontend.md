# Rule 05: Vue Frontend Constraints

## Trigger Conditions

- `vue_web` 下页面、组件、样式、i18n 变更。

## Hard Constraints (MUST)

- 遵循 `vue_web/docs/web-ui.md`。
- 使用 token/hook，不在组件写死语义色。
- 维持 panel 交互一致：拖拽、缩放、全屏。
- 保持 `zh-CN` 与 `en-US` 可用。

## Counterexamples

- 新组件绕过 `ui-focus-ring` 等全局状态语义。
- 破坏 window 布局持久化键规则。

## Validation Commands

- `pnpm -C vue_web typecheck`
- `pnpm -C vue_web test:run`
- `pnpm -C vue_web build`
