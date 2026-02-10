# Web UI Standards (Thread B)

## 1. 风格原则

### 1.1 Console-first
- UI 面向控制台工作流，优先保证信息密度与状态可读性。
- 默认密度为 `compact`，通过 `comfortable` 提供可读性增强。
- 页面结构优先遵循“筛选 -> 列表 -> 详情/日志”的操作路径。

### 1.2 Material 3 状态语义
- 交互组件必须覆盖：`hover` / `pressed` / `focus-visible` / `disabled` / `loading`。
- 禁止在组件内散落状态 utility；状态统一通过全局 hook 类注入。

### 1.3 视觉边界
- 圆角：卡片 `10-12px`，按钮 `8px`，画布节点 `6-8px`。
- 分层：边框主导；阴影仅用于浮层（Dialog/Dropdown/Toast）。
- 动效：仅状态过渡与必要的进入/退出，不做大面积动效。
- 必须支持 `prefers-reduced-motion: reduce`，自动降低非必要动效。

## 2. Design Tokens

主文件：`web/src/design-system/tokens.css`

### 2.1 命名规范
- 颜色：`--ui-neutral-*` / `--ui-primary-*` / `--ui-success|warn|error|info`
- 字体：`--ui-font-*`
- 圆角：`--ui-radius-*`
- 阴影：`--ui-shadow-*`
- 状态：`--ui-focus-*` / `--ui-disabled-*` / `--ui-loading-*`
- 密度：`--ui-control-*` / `--ui-page-gap` / `--ui-table-row-h`

### 2.2 新增 token 流程
1. 在 `tokens.css` 中新增变量，并同时补齐 light/dark 值。
2. 若需在 Tailwind 使用，同步映射到 `tailwind.config.ts`。
3. 在组件中通过 `var(...)` 或已映射的 Tailwind token 消费。
4. 在本文件记录 token 用途与约束，避免语义漂移。

### 2.3 硬规则
- 不允许在组件中写死语义颜色（例如直接写固定 hex 作为状态色）。
- 状态色只能来自 tokens。

## 3. 全局状态 Hook 类（强制）

文件：`web/src/style.css`

- `ui-focus-ring`：只在 `:focus-visible` 显示高对比 ring。
- `ui-pressable`：统一 hover/pressed 反馈与过渡。
- `ui-disabled`：统一禁用态可视与交互阻断。
- `ui-loading`：统一 loading 光标与透明度反馈。

约束：交互组件根元素必须组合这四类，禁止自行实现平行状态体系。

## 4. 组件状态矩阵

| 组件 | hover | pressed | focus-visible | disabled | loading | 备注 |
|---|---|---|---|---|---|---|
| Button | `ui-pressable` | `ui-pressable` | `ui-focus-ring` | `ui-disabled` | `ui-loading` | `loading` 与 `disabled` 分离，`blockWhileLoading` 默认阻断 |
| Input | `ui-pressable` | `ui-pressable` | `ui-focus-ring` | `ui-disabled` | `ui-loading` | 保持可读 placeholder |
| Textarea | `ui-pressable` | `ui-pressable` | `ui-focus-ring` | `ui-disabled` | `ui-loading` | 多行输入同一控制高度语义 |
| Select(Listbox) | `ui-pressable` | `ui-pressable` | `ui-focus-ring` | `ui-disabled` | `ui-loading` | 选项高亮只用 token |
| Tabs | `ui-pressable` | `ui-pressable` | `ui-focus-ring` | `ui-disabled` | N/A | 选中态使用 primary token |
| Dialog | N/A | N/A | 焦点陷阱 + `ui-focus-ring` | N/A | confirm 按钮可 loading | 遮罩/浮层使用 overlay token |
| Dropdown(Menu) | `ui-pressable` | `ui-pressable` | `ui-focus-ring` | `ui-disabled` | trigger 可 loading | ESC 关闭、键盘可进入菜单 |
| Table | 行 hover 可选 | 行按压可选 | 行 focus 可选 | N/A | `loading` skeleton | `ready/loading/empty/error` 四态，交互行必须支持 Enter + Space |
| Toast | 可关闭按钮 hover | 按压关闭按钮 | `ui-focus-ring` | N/A | N/A | `info/success/warn/error` 级别，容器 `aria-live=polite` |

## 5. Theme / Density / i18n

### 5.1 Theme
- 模式：`system | light | dark`
- 存储键：`goyais.ui.theme`
- 兼容旧键：`goyais.theme`
- `system` 跟随 `prefers-color-scheme`。

### 5.2 Density
- 根入口：`html[data-density='compact|comfortable']`
- 仅允许密度变量：
  - `--ui-control-h`
  - `--ui-control-px`
  - `--ui-control-py`
  - `--ui-page-gap`
  - `--ui-table-row-h`
- 组件必须用 `var(...)` 消费，不得定义组件私有密度体系。

### 5.3 i18n
- 语言：`zh-CN` / `en-US`
- key 命名空间：`nav.*` / `common.*` / `page.*` / `status.*` / `error.*`
- 缺失策略固定：`当前 locale -> en-US -> key`
- 开发态开启 missing warn。

### 5.4 messageKey 对齐
- 后端错误结构：`error: { code, messageKey, details }`
- 前端通过统一翻译入口映射 `messageKey`，并由 `ErrorBanner` 渲染。

## 6. 布局规范

### 6.1 Shell 模式
- 模式枚举：`console | topnav | focus`。
- 偏好存储：`goyais.ui.layout`，值域：`auto | console | topnav | focus`。
- `auto` 按路由 `meta.layoutDefault` 生效；手动选择后全局覆盖直到切回 `auto`。
- 路由仍使用 `createWebHistory`（兼容 single-binary SPA fallback）。

### 6.2 结构规则
- `console`：`TopBar + SideNav + Content`。
- `topnav`：`TopBar + TopNavBar + Content`。
- `focus`：`TopBar + Content`（无常驻导航）。
- `compact` 下 SideNav 默认折叠；hover 临时展开；支持 pin 固定。

### 6.3 窗口化布局（Desktop）
- 三种模式都支持窗口化拖拽/缩放（仅 desktop，mobile 降级为单列卡片流）。
- 页面结构统一：`PageHeader(固定) + WindowBoard(可拖拽窗口区)`。
- 窗口能力：拖拽、右/下/右下缩放、点击置顶、允许重叠。
- 键盘等价能力：`Alt + Arrow` 移动窗口，`Alt + Shift + Arrow` 调整窗口宽高（步进 16px）。
- 每页提供“重置窗口布局”动作。

### 6.4 窗口状态持久化
- 存储键格式：`goyais.ui.windows.<layoutMode>.<routeKey>.v1`。
- 同一路由在不同布局模式下独立持久化，互不污染。
- 路由窗口清单由 `web/src/design-system/window-manifests.ts` 维护。

### 6.5 页面白名单窗口单元（首版）
- `/`：`design-tokens`、`state-hooks`、`status`、`backgrounds`、`empty-states`
- `/commands`：`filters`、`list`、`detail`
- `/assets`：`filters`、`list`、`detail`
- `/canvas`：`canvas-surface`
- `/plugins`：`plugin-catalog`
- `/streams`：`stream-overview`、`stream-logs`
- `/settings`：`preferences`、`component-matrix`
- `/forbidden`：`forbidden-state`
- `not-found`：`not-found-state`

## 7. 新增组件 Checklist

新增组件前：
- [ ] 是否复用 tokens 而非硬编码颜色/间距。
- [ ] 是否接入 `ui-focus-ring`/`ui-pressable`/`ui-disabled`/`ui-loading`。
- [ ] 是否在 light/dark 下保持对比可读。
- [ ] 是否在 compact/comfortable 下尺寸一致。
- [ ] 是否具备键盘路径（Tab、ESC、Enter、Space）与 aria 语义。
- [ ] 文案是否走 i18n key。

新增页面前：
- [ ] 是否遵循控制台信息架构（筛选->列表->详情）。
- [ ] 是否定义空态/加载态/错误态。
- [ ] 是否保持 mock 数据与真实接口契约字段同构。

## 8. 验收要点（Thread B）

- `pnpm -C web typecheck` 与 `pnpm -C web build` 必须通过。
- 主题/语言/密度切换刷新后保持。
- `focus ring` 在 light/dark 可见。
- Dialog/Dropdown 键盘路径通过（focus 进入、ESC 关闭、Tab 路径正确）。
- 窗口键盘路径通过（`Alt+Arrow` 移动、`Alt+Shift+Arrow` 缩放）。
- 系统启用 `reduced-motion` 时动效显著降低。
- `/commands` 与 `/assets` 双栏交互可用。

## 9. 图标与素材规范（Thread 7）

### 9.1 图标体系
- 采用 Heroicons（MIT）并统一封装为 `web/src/components/ui/Icon.vue`。
- 运行时图标名称由 `web/src/design-system/icon-registry.ts` 管理，禁止页面自行拼接路径。
- 已使用图标必须同步落库到 `web/src/assets/icons/heroicons/24/outline/`，便于分发与审计。
- 图标必须保持统一描边与尺寸语义（24 基准，UI 中按 size 缩放）。

### 9.2 空状态插画
- 运行时空状态插画位于 `web/src/assets/illustrations/states/`。
- 必须通过 `EmptyState` 组件使用，不允许页面散落自定义空态样式。
- 插画颜色需与 token 对齐，禁止硬编码 hex 语义色。
- unDraw 原始素材仅作为来源归档，放置于 `web/src/assets/illustrations/undraw/raw/`。

### 9.3 背景资源
- 背景 SVG 资源放在 `web/src/assets/bg/`。
- 可切换类名：
  - `ui-bg-grid`
  - `ui-bg-gradient`
  - `ui-bg-dots`
  - `ui-bg-stack-console`
- 背景层必须使用 `ui-bg-host` + `ui-bg-content` 结构，确保 focus ring 可见且不受遮挡。

### 9.4 资源索引与许可审计
- 资源索引：`web/src/assets/RESOURCE_CATALOG.yaml`
- 许可记录：`web/src/assets/THIRD_PARTY_NOTICES.md`
- 新增第三方素材时，两者必须同一提交更新。
