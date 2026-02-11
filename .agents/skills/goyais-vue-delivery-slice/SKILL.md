---
name: goyais-vue-delivery-slice
description: Vue 端垂直切片交付模板，强调 web-ui 规范、panel 行为一致性与 i18n。
---

# goyais-vue-delivery-slice

## 适用场景

- 页面、组件、布局、i18n、样式相关改动。

## 输入

- `docs/prd.md`
- `vue_web/docs/web-ui.md`
- `vue_web/src/design-system/*`

## 输出

- 页面切片计划与实现证据。

## 严格步骤

1. 保持 token/hook 语义一致。
2. 校验 panel 拖拽/缩放/全屏行为。
3. 维护 `zh-CN` 与 `en-US`。
4. 执行 typecheck/test/build。

## 验收

- 无设计系统语义漂移。
