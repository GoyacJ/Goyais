# Web UI Standards (v2)

## 1. 目标与边界

### 1.1 风格目标
- Console-first：优先信息密度与扫描效率，遵循“筛选 -> 列表 -> 详情/日志”主路径。
- Material 3 状态语义：hover / pressed / focus-visible / disabled / loading 统一口径。
- Notion/GitHub 克制层级：边框与轻底色主导分层，阴影仅用于浮层。

### 1.2 硬约束
- 禁止在组件内硬编码语义色或状态色（hex、固定 Tailwind 色阶）。
- 状态语义必须通过全局 hook 类注入：
  - `ui-focus-ring`
  - `ui-pressable`
  - `ui-disabled`
  - `ui-loading`
- 必须保留并兼容：
  - Theme 三态：`system | light | dark`
  - Density 双态：`compact | comfortable`
  - i18n 缺失回退：`当前 locale -> en-US -> key`

## 2. 克制层级（Notion/GitHub 风格）规则

### 2.1 背景与分层
- 页面背景使用 neutral bg；主容器与控件表面使用 panel/surface-2。
- 内容分层优先顺序：
  1. 边框（subtle / default / strong）
  2. 轻底色（surface-2 / state layer）
  3. 阴影（仅浮层）

### 2.2 阴影与浮层
- `Dialog / Dropdown / Toast` 使用统一 `ui-overlay-panel` 与 `--ui-shadow-overlay`。
- 常规卡片、列表行、表格区域禁止使用投影做层级。

### 2.3 字体层级
- 仅四档：
  - 标题：`--ui-type-title-size`
  - 正文：`--ui-type-body-size`
  - 说明：`--ui-type-caption-size`
  - mono：`--ui-type-mono-size`
- 禁止在页面内继续扩展无语义字号层级。

### 2.4 Mono 区域
- 日志/代码统一使用 `ui-log-surface + ui-monospace`。
- 禁止自定义“高饱和代码块”背景；保持克制对比可读。

## 3. 状态语义实现细则（Material 3）

### 3.1 Hover / Pressed（state-layer）
- 统一使用 `ui-pressable`：
  - hover：`--ui-state-layer + --ui-state-hover-opacity`
  - pressed：`--ui-state-layer + --ui-state-pressed-opacity`
  - 边框微调：`--ui-state-hover-border / --ui-state-pressed-border`
- 禁止组件自行拼接状态颜色。

### 3.2 Focus-visible
- 统一使用 `ui-focus-ring`，采用 offset + ring + contrast 三层高对比口径。
- 要求在 light/dark 以及纹理背景上都可见。

### 3.3 Disabled vs Loading
- `ui-disabled`：不可交互（`pointer-events: none`、`cursor: not-allowed`、opacity）。
- `ui-loading`：表示处理中（cursor + opacity），默认不强制阻断。
- Button 特例：
  - `blockWhileLoading=true`：追加阻断（`ui-loading-block` 或等效语义）。
  - `blockWhileLoading=false`：允许继续交互。

### 3.4 Reduced Motion
- `prefers-reduced-motion: reduce` 下，禁用非必要动画，仅保留必要状态过渡。

## 4. Token 契约（新增/变更）

主文件：`web/src/design-system/tokens.css`

### 4.1 Neutral 分层（新增/强化）
| Token | 用途 | 说明 |
|---|---|---|
| `--ui-neutral-surface-2` | 次级表面底色 | 用于卡片/控件轻分层 |
| `--ui-neutral-fg-subtle` | 次级文字 | 用于说明性文案 |
| `--ui-neutral-border-subtle` | 轻分割线 | 表格行、卡片内分隔 |
| `--ui-neutral-border-strong` | 强边界 | pressed 或强调边界 |

### 4.2 State-layer（新增）
| Token | 用途 | 说明 |
|---|---|---|
| `--ui-state-layer` | 状态叠层基色 | hover/pressed 叠层 |
| `--ui-state-hover-opacity` | hover 透明度 | 轻反馈 |
| `--ui-state-pressed-opacity` | pressed 透明度 | 明显反馈但克制 |
| `--ui-state-hover-border` | hover 边框强度 | 边框微调 |
| `--ui-state-pressed-border` | pressed 边框强度 | 边框增强 |

### 4.3 Focus 与可读性（新增/变更）
| Token | 用途 | 说明 |
|---|---|---|
| `--ui-focus-ring` | focus 主色 | 跨主题统一 |
| `--ui-focus-ring-offset` | focus 偏移底色 | 与背景隔离 |
| `--ui-focus-ring-contrast` | focus 对比外圈 | 防背景纹理吞焦点 |

### 4.4 日志与浮层（新增/变更）
| Token | 用途 | 说明 |
|---|---|---|
| `--ui-shadow-overlay` | 浮层阴影 | Dialog/Dropdown/Toast 专用 |
| `--ui-log-bg` | 日志背景 | mono 区域 |
| `--ui-log-fg` | 日志文字 | mono 区域 |
| `--ui-log-border` | 日志边框 | mono 区域 |

### 4.5 交互状态（调整）
| Token | 用途 | 说明 |
|---|---|---|
| `--ui-disabled-opacity` | 禁用透明度 | 与 loading 区分 |
| `--ui-loading-opacity` | 加载透明度 | 可与阻断策略解耦 |
| `--ui-loading-cursor` | 加载光标 | 统一 `progress` |

### 4.6 字号层级（新增）
| Token | 用途 | 说明 |
|---|---|---|
| `--ui-type-title-size` | 标题字号 | 组件标题 |
| `--ui-type-body-size` | 正文字号 | 默认阅读 |
| `--ui-type-caption-size` | 说明字号 | 次信息 |
| `--ui-type-mono-size` | mono 字号 | 日志/代码 |

## 5. 禁止用法（必须遵守）
- 禁止在组件模板写入：
  - `bg-primary-500/10`
  - `text-primary-700`
  - `text-error`（直接拼语义状态）
  - 任何 hex 颜色或固定 Tailwind 色阶作为状态色
- 禁止复制独立状态系统（平行于四个全局 hook）。
- 禁止在非浮层组件添加阴影做层级。

## 6. 组件与页面落地口径

### 6.1 组件
- `Button/Input/Textarea/Select/Tabs/Dialog/Dropdown/Table/Toast` 必须统一 token + hook。
- 表格与列表扫描性要求：
  - 行高受 density token 驱动
  - 行分隔线使用 subtle border
  - hover 与 selected 使用统一状态语义

### 6.2 页面
- `/commands` 与 `/assets` 双栏：
  - 左栏：筛选 + 列表扫描优先
  - 右栏：详情/日志分层清晰
  - 空态/错误态使用克制语义，不使用大色块

## 7. Theme / Density / i18n（保持不变）
- Theme：`system | light | dark`
- Density：`compact | comfortable`
- i18n 缺失策略：`当前 locale -> en-US -> key`
- 开发态仍开启 missing warn。

## 8. 验收清单

### 8.1 自动化
- `pnpm -C web typecheck`
- `pnpm -C web test:run`
- `pnpm -C web build`

### 8.2 手测
- 主题三态切换与持久化
- 密度双态切换与间距节奏
- locale 切换与 fallback 行为
- focus ring 在 light/dark + 背景纹理下可见
- Dialog/Dropdown/Tabs/Select 键盘路径与 aria 语义
- `/commands` 与 `/assets` 双栏交互完整
