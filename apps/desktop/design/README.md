# Desktop Design Ownership (v0.4.0)

## Token 文件所有权

- `src/styles/tokens.css` 由 Pencil 同步管理。
- Antigravity / Goyais Desktop 作为消费方，仅使用 token，不直接修改 token 定义。

## 同步流程（占位）

1. 设计侧在 Pencil 更新 token。
2. 通过同步流程更新 `tokens.css`。
3. 前端验证布局与占位渲染，不在业务组件内硬编码颜色/间距/字号/圆角。

## 冲突处理原则

- 业务样式冲突优先回到 token 层协商。
- 禁止在业务组件里直接改 token 源值来绕过同步流程。
