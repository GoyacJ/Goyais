# Goyais UI Guidelines (V1)

## 1. Style Positioning

**Style statement**
- Professional, restrained, high-density, readable, and diagnostic-first.

**Execution principles**
1. Divider over card shadow
- Use `1px` divider and background layering for structure.
- Avoid heavy shadows in main content surfaces.

2. Explicit risk semantics
- Any write/command/network/delete/exfil action must render: `risk color + icon + copy + badge`.
- Never rely on color alone.

3. Controllable density
- Timeline/Tool/Diff default to high density.
- Long output must support collapse, filter, and copy.

## 2. Visual Tokens

Token source files:
- `apps/desktop-tauri/src/styles/tokens.css`
- `apps/desktop-tauri/tailwind.config.ts`

### 2.1 Typography
- `--font-ui`: system sans stack
- `--font-mono`: ui-monospace stack
- `--text-h1-size`: `20-24px` target
- `--text-h2-size`: `18-20px` target
- `--text-body-size`: `13-14px` target
- `--text-small-size`: `12-13px` target
- `--text-mono-size`: `12-13px` target
- `tab-size: 2`
- Diff/log/tool payload uses mono by default.

### 2.2 Spacing (4pt Grid)
- `--space-*`: 4/8/12/16/20/24/32/40/48
- `--page-padding`: 24
- `--panel-gap`: 16
- `--content-padding`: 16
- `--form-gap`: 12
- `--toolbar-height`: 40-44
- `--sidebar-width`: 240-280

### 2.3 Radii
- `--radius-control`: 8
- `--radius-panel`: 12
- `--radius-overlay`: 16

### 2.4 Borders / Dividers
- `--border-width`: `1px`
- `--color-border`
- `--color-border-subtle`

### 2.5 Shadows (Overlay only)
- `--shadow-sm` / `--shadow-md` / `--shadow-lg`
- Main surfaces (`Timeline/Diff/Panel`) use border and background only.

### 2.6 Glass / Blur
- `--glass-blur`: `8-16px`
- `--glass-alpha`: `5-12%`
- `--glass-border`: `1px`
- Applied only to palette/dialog overlays.

### 2.7 Colors (Semantic + Risk)
- Semantic: `background`, `foreground`, `muted`, `border`, `accent`, `success`, `warning`, `info`, `destructive`
- Risk-only: `risk-write`, `risk-exec`, `risk-network`, `risk-delete`, `risk-exfil`

### 2.8 Accessibility
- Body contrast >= `4.5:1`
- Secondary text >= `3:1`
- Focus ring: tokenized (`color/width/offset`), mandatory on all interactive controls.
- Dialog uses focus trap; capability confirmation supports keyboard (`Enter/Esc`, plus queue shortcuts).

## 3. Layout Rules

App shell:
- Left: collapsible sidebar (projects/workspaces/navigation)
- Top: project/model/runtime/sync status
- Main: consistent page padding and scroll behavior

Run page:
- 3-column grid
- Left: run composer (project/session/model/workspace/task)
- Middle: Timeline + Diff-first panel
- Right: Context + Tool Details + Permission Queue

## 4. Component Interaction Rules

### Timeline Event Card
- Header: type badge, seq, timestamp, collapse toggle, action menu
- Body: payload (copyable, mono)
- Long output: collapsed by default after `120` lines (`Show more`)
- Streaming states show cursor/loader without layout jump

### Tool Details
- Tabs: `Input / Output / Logs / Timing`
- Required: tool name, args, cwd/paths/domains, output summary, stack/result copy
- Error path provides `Copy diagnostics` and `Export diagnostics` placeholder

### Diff Panel (Diff-first)
- Hunk-level checkbox selection
- Primary action changes by state (`Apply` vs `Apply selected`)
- Secondary action: `Accept all` with confirmation
- Raw diff is always copyable

### Context Panel
- Shows injection source (`file/retrieval/pasted`)
- Displays size + token estimate
- Supports remove (P0)
- Warns when context budget grows

### Capability Prompt + Queue
- Queue center lists all pending confirmations
- Decision options: `Allow Once / Always Allow / Deny`
- Expandable details: command, cwd, paths, domains
- Outside-workspace path risk explicitly flagged
- Keyboard shortcuts on pending item: `Y` allow once, `A` always, `N` deny

### Command Palette (P1 scaffold)
- `Cmd/Ctrl + K`
- Grouped commands (project/model/tool/settings)
- Includes recent section and shortcut hint
- Current implementation is UI scaffold without backend execution wiring

### Toast / Notifications
- Variants: `success/warning/error/info`
- Error toast supports expandable diagnostics and copy action

## 5. Motion Policy

Allowed:
- Expand/collapse
- State transition feedback (`running -> waiting_confirmation -> done`)
- Streaming indicators

Disallowed:
- Full-page transitions
- Bouncy motion style
- Persistent flashing backgrounds

## 6. Theme Strategy

### A. VSCode-like (P0 implemented)
- Higher contrast and denser reading rhythm
- Divider-driven structure
- Smaller default body typography for code-heavy workflows
- Sharper risk color semantics
- Minimal shadows on main surfaces

### B. Linear-like (documented fallback)
- Softer layer contrast and more whitespace
- Larger body text and relaxed spacing
- Rounded, lighter presentation style
- Better for demos and lower-density review contexts

Recommendation:
- Keep A as default for developer operation efficiency and risk audit clarity.
- Use B for presentation-first or stakeholder demo modes.

## 7. Engineering Conventions

Current front-end structure (`apps/desktop-tauri/src`):
- `app/`: layout and providers
- `components/ui/`: primitive UI building blocks
- `components/domain/`: timeline, diff, permission, context, tool details, feedback, palette
- `pages/`: route-level pages
- `lib/`: helpers (`cn`, `risk`, `events`, `diff`, `api-error`, `shortcuts`)
- `stores/`: zustand slices
- `styles/`: `globals.css`, `tokens.css`
- `api/`: runtime/sync clients
- `types/`: protocol and UI view models

## 8. shadcn-style Primitive Coverage

Implemented primitives:
- `Button`
- `Input`
- `Textarea`
- `Card`
- `Badge`
- `Tabs`
- `Dialog`
- `Sheet`
- `DropdownMenu`
- `Toast` + `Toaster`
- `Tooltip`
- `ScrollArea`
- `Separator`

Recommended next primitives:
1. `Checkbox` (for richer diff and context toggles)
2. `Popover` (quick tool metadata preview)
3. `Command` (full command-palette behavior model)
4. `Table` (project/model list density and sorting)
5. `Skeleton` (structured loading placeholders)

## 9. Acceptance Checklist

- `pnpm --filter @goyais/desktop-tauri typecheck` passes
- `pnpm --filter @goyais/desktop-tauri test` passes
- `pnpm --filter @goyais/desktop-tauri build` passes
- `pnpm --filter @goyais/desktop-tauri lint` passes
- Run page keeps SSE timeline and patch visibility without protocol changes
- Permission confirmations remain queue-based with explicit risk details

## 10. i18n Rules

Scope:
- Locale support: `zh-CN` and `en-US`
- Default locale: `zh-CN`
- Persist key: `localStorage["goyais.locale"]`

Files:
- `apps/desktop-tauri/src/i18n/index.ts`
- `apps/desktop-tauri/src/i18n/types.ts`
- `apps/desktop-tauri/src/i18n/locales/zh-CN/common.json`
- `apps/desktop-tauri/src/i18n/locales/en-US/common.json`

Conventions:
- Use a single namespace for P0: `common`
- Key style: `domain.section.item` (example: `permission.dialog.reviewHint`)
- Do not hardcode UI labels in components; use `useTranslation()` with `t("...")`
- Toast titles/descriptions and diagnostics labels must be translated

Language switching:
- Use `useSettingsStore().setLocale(locale)` only
- `setLocale` updates both i18n runtime language and localStorage
- Provide switch entry in both Topbar (quick) and Settings (formal)

Fallback strategy:
- Detect initial locale order: persisted locale -> `navigator.languages` match -> `zh-CN`
- Unsupported locales fall back to `zh-CN`

Time formatting:
- Use `formatTimeByLocale(isoTs, locale)` from `apps/desktop-tauri/src/lib/format.ts`
- Do not call `new Date(...).toLocaleTimeString()` directly in UI components

Quality gates:
- Keep locale keysets aligned (`src/i18n/__tests__/resources.test.ts`)
- New UI copy must add both `zh-CN` and `en-US` keys in the same change
