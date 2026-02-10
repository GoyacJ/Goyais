# Web UI Standards (v3)

## 1. Goals and Guardrails

### 1.1 Visual + Interaction Goals
- Notion-like restrained hierarchy: content-first, decoration-second.
- Console-first density: scanable lists and two-pane workflow (`filter -> list -> detail/log`).
- Material 3 state semantics: unified hover/pressed/focus-visible/disabled/loading behavior.

### 1.2 Hard Constraints
- Do not hardcode semantic/status colors in components (hex or fixed color scale).
- Keep global hooks compatible:
  - `ui-pressable`
  - `ui-focus-ring`
  - `ui-disabled`
  - `ui-loading`
- Keep existing system behavior unchanged:
  - Theme: `system | light | dark`
  - Density: `compact | comfortable`
  - i18n fallback: `current locale -> en-US -> key`

## 2. Restrained Hierarchy (Notion-like)

### 2.1 Layering Rules
- Use neutral surfaces and subtle borders to organize hierarchy.
- Keep shadows only for overlays (`Dialog`, `Dropdown`, `Toast`).
- Keep accent colors small-area only (indicator, icon, tiny marker).

### 2.2 Do / Don't
- Do:
  - Use subtle separators and low-contrast surface layering.
  - Keep spacing rhythm consistent and density-driven.
  - Keep mono blocks structured and readable (`id/hash/uri/log`).
- Don't:
  - No heavy card shadows for normal content blocks.
  - No large saturated selected backgrounds.
  - No component-local state system drifting from global hooks.

## 3. Unified Row Semantics (List/Table)

### 3.1 Shared Classes
- `ui-list-row`
- `ui-list-row--interactive`
- `ui-list-row--selected`
- `ui-list-row--focus`

### 3.2 Behavioral Contract
- Hover: lightweight state-layer only.
- Selected: `2px` left indicator + lightweight state-layer + subtle border.
- Focus-visible: offset + contrast ring visible in both light/dark and textured background.
- Row rhythm (`height`, `padding`, separators): driven by density tokens.

## 4. Detail Block Semantics

### 4.1 Shared Classes
- `ui-detail-block`
- `ui-detail-header`
- `ui-detail-meta`
- `ui-detail-mono`

### 4.2 Structure Contract
- Header: title + key status/action.
- Meta: compact property grid.
- Mono: high-readability machine values (`traceId`, `hash`, `uri`, logs).

## 5. State Semantics (Material 3)

| State | Visual rule | Interaction rule |
|---|---|---|
| `hover` | light state-layer | non-blocking |
| `pressed` | stronger state-layer than hover + clearer border | non-blocking |
| `focus-visible` | offset + primary ring + contrast ring | keyboard focus must be obvious |
| `disabled` | dimmed + blocked pointer | always blocked |
| `loading` | dimmed + progress cursor | default non-blocking |
| `loading + blockWhileLoading` | same as loading | blocked for Button only |

## 6. Token Contract (Small-Step Only)

### 6.1 Alias Tokens (non-breaking)
- `--ui-surface-0/1/2`
- `--ui-fg-subtle`
- `--ui-border-subtle`

### 6.2 Row/Selection Tokens
- `--ui-list-selected-indicator`
- `--ui-list-selected-opacity`
- `--ui-list-row-px`
- `--ui-list-row-py`

### 6.3 State Opacity
- Keep hover/pressed split; pressed must be stronger.
- Keep background texture low-noise so content remains strongest contrast target.

## 7. Component Checklist

### 7.1 Required
- `Button/Input/Textarea/Select/Tabs/Dialog/Dropdown/Table/Toast` must follow token + hook semantics.
- Overlay shadow only for overlay components.

### 7.2 Keyboard and A11y
- Tabs: roving tabindex + Arrow/Home/End.
- Interactive list/table rows: Enter/Space activation and visible focus.
- Dialog/Dropdown/Select: keyboard path must not regress.

## 8. Page Checklist

### 8.1 `/commands`
- Left pane must be compact scan list (`type/status/time/id(mono)`).
- Selection updates right-side detail/log.
- Selected row must stay restrained (no large saturated block).

### 8.2 `/assets`
- Left pane row semantics must match `/commands`.
- Right pane follows `header/meta/mono` split.

## 9. Strong Acceptance Points (Notion Feel)

1. Hover uses lightweight state-layer only.
2. Selected row uses `2px` indicator + light state-layer + subtle border.
3. Row separators and density rhythm are subtle and consistent.
4. Focus ring remains clear in light/dark + textured background.
5. `/commands` left pane clearly improves scan density.
6. Right detail panel is clearly split into `header/meta/mono` with restrained tone.

## 10. Validation Commands

- `pnpm -C web typecheck`
- `pnpm -C web test:run`
- `pnpm -C web build`
