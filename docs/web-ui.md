# Web UI Standards (Thread B)

## 1. 风格原则

### 1.1 Console-first
- 视觉定位是高密度控制台，而不是营销站点。
- 信息组织优先级：状态与结构 > 装饰。
- 页面默认 `compact` 密度，保持高信息吞吐。

### 1.2 Material 3 状态语义
- 全局交互状态统一为：`hover` / `pressed` / `focus` / `disabled` / `loading`。
- 不允许组件自行拼接零散 utility 来实现状态语义，必须复用全局 hook 类。

### 1.3 视觉边界
- 圆角克制：
  - 卡片：10-12px（`--ui-radius-card`）
  - 按钮：8px（`--ui-radius-button`）
  - 画布节点：6-8px（`--ui-radius-canvas-node`）
- 分层策略：边框优先；阴影仅用于浮层（Dialog/Dropdown）。
- 动效仅用于状态变化和小范围过渡，不使用大面积炫技动画。

## 2. Design Tokens

Tokens 定义文件：`/Users/goya/Repo/Git/Goyais/web/src/design-system/tokens.css`

### 2.1 颜色体系
- `neutral`：背景/边框/文本（`--ui-neutral-*`）
- `primary`：选中/主按钮/主交互（`--ui-primary-*`）
- `semantic`：`--ui-success` / `--ui-warn` / `--ui-error` / `--ui-info`

### 2.2 排版与字体
- 主字体：`--ui-font-sans`
- 等宽字体：`--ui-font-mono`
- 日志/ID/技术标识必须使用 `ui-monospace`

### 2.3 间距与形状
- 页面节奏：`--ui-page-gap`
- 控件高度与内边距统一由密度变量控制（见第 4 节）
- 阴影变量：`--ui-shadow-overlay`（仅浮层使用）

### 2.4 Tailwind 映射
- 文件：`/Users/goya/Repo/Git/Goyais/web/tailwind.config.ts`
- 所有颜色/圆角/阴影/字体通过 CSS variables 映射。
- `plugins` 固定为空，不引入会改写样式语义的插件。

## 3. 全局状态 Hook 类（强制）

定义文件：`/Users/goya/Repo/Git/Goyais/web/src/style.css`

- `ui-focus-ring`
  - 仅在 `:focus-visible` 显示。
  - light/dark 下均需高对比可见。
- `ui-pressable`
  - 统一 hover/pressed 反馈与轻量 transition。
- `ui-disabled`
  - 统一禁用视觉和交互阻断。
- `ui-loading`
  - 统一 loading 态光标/可交互策略。

约束：所有交互组件根元素必须组合上述 hook 类。

## 4. Theme / Density / i18n 规范

### 4.1 Theme
- 模式：`system | light | dark`
- 存储键：`goyais.ui.theme`
- 兼容旧键迁移：`goyais.theme`
- `system` 模式跟随 `prefers-color-scheme` 实时变更。
- 初始化先应用主题再挂载应用，避免闪烁。

### 4.2 Density
- 全局入口：`html[data-density='compact|comfortable']`
- 仅允许以下统一变量：
  - `--ui-control-h`
  - `--ui-control-px`
  - `--ui-control-py`
  - `--ui-page-gap`
  - `--ui-table-row-h`
- 组件必须通过 `var(...)` 消费；禁止组件私有密度变量。
- 默认：`compact`
- 存储键：`goyais.ui.density`

### 4.3 i18n
- 语言：`zh-CN` 与 `en-US`
- key 命名：
  - `nav.*`
  - `common.*`
  - `page.*`
  - `status.*`
  - `error.*`
- fallback 链路（固定）：`当前 locale -> en-US -> key`
- 开发态开启 missing warn。
- locale 存储键：`goyais.ui.locale`
- 兼容旧键迁移：`goyais.locale`

### 4.4 后端 messageKey 对齐
- 错误结构：`error: { code, messageKey, details }`
- 前端通过统一翻译入口将 `error.messageKey` 映射为 i18n key。
- 建议统一展示组件：`ErrorBanner`。

## 5. 基础布局规范

### 5.1 AppShell
- 结构：`TopBar + SideNav + Content`
- 侧边导航固定承载 `/` `/canvas` `/commands` `/assets` `/plugins` `/streams` `/settings`
- 路由模式固定 `createWebHistory`，不启用 hash/SSR。

### 5.2 页面模板
- 列表页：`PageHeader + SectionCard(Table + Pagination)`
- 详情页：`PageHeader + 多 SectionCard`（状态优先）
- 画布页：`PageHeader + Canvas Surface`（节点圆角 6-8px）

### 5.3 反馈组件
- 空态：`EmptyState`
- 加载：`SkeletonBlock`
- 错误：`ErrorBanner`

## 6. 组件规范（状态矩阵）

基础组件清单：
- `Button`
- `Input`
- `Textarea`
- `Select`
- `Dialog`
- `Dropdown`
- `Tabs`
- `Badge`
- `ToastViewport`
- `Table`
- `Pagination`

状态矩阵要求：
- Hover：使用 `ui-pressable`
- Pressed：使用 `ui-pressable`
- Focus：使用 `ui-focus-ring`
- Disabled：使用 `ui-disabled`
- Loading：使用 `ui-loading`（适用于可加载组件）

## 7. 新增页面/组件 Checklist

新增页面时：
- [ ] 使用 `PageHeader` + `SectionCard` 组合。
- [ ] 颜色/间距/圆角全部来自 tokens。
- [ ] 交互元素使用四个全局状态 hook 类。
- [ ] 文字 key 落在规范命名空间。
- [ ] 深浅色对比可读。
- [ ] compact/comfortable 两档密度可读。

新增组件时：
- [ ] `var(...)` 消费统一 density 变量。
- [ ] 不新增样式语义型 Tailwind 插件。
- [ ] 无障碍键盘路径可用（focus-visible、ESC、Tab 顺序）。
- [ ] 错误与状态文案使用 i18n key，不写死业务中文。

## 8. 验收项（Thread B）

- `pnpm -C web dev` 可启动。
- 主题 `system/light/dark` 切换生效且刷新后保持。
- `zh-CN/en-US` 切换生效。
- `compact/comfortable` 至少影响按钮、输入、表格行高、页面间距。
- focus ring 在 light/dark 均清晰可见。
- Dialog/Dropdown 键盘可访问性通过：
  - Dialog：焦点进入、`ESC` 关闭、`Tab/Shift+Tab` 循环。
  - Dropdown：键盘打开、焦点进入菜单、`ESC` 关闭、`Tab` 行为可预期。
