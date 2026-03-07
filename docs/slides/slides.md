---
theme: default
title: Goyais Refactor 2026-02-25
info: |
  Frontend + Hub debt cleanup without compatibility runtime paths.
highlighter: shiki
lineNumbers: true
transition: fade-out
drawings:
  persist: false
mdc: true
---

<div class="text-center">

# Goyais Refactor
## Frontend / Hub Debt Cleanup

<p class="opacity-80">Date: 2026-02-25</p>

</div>

---
layout: two-cols
---

# Goals

- remove runtime fallback/mock
- migrate state to Pinia
- migrate style system to UnoCSS
- keep `/v1` while removing legacy branches
- enforce verification evidence

::right::

# 目标

- 删除 fallback/mock 运行时兜底
- 状态层迁移到 Pinia
- 样式体系迁移到 UnoCSS
- 保持 `/v1` 同时移除 legacy 分支
- 全程保留验证证据

---

# Key Deliverables

1. Turbo + catalogs + shared package (`@goyais/shared-core`)
2. Desktop strict API-only behavior
3. Modal A11y and keyboard flow fixes
4. Hub compatibility removals (route/auth/runtime/storage)
5. Coverage gate and smoke E2E

---

# Verification Snapshot

```bash
pnpm lint
pnpm test
pnpm coverage:gate
pnpm e2e:smoke
go test ./...            # services/hub
go vet ./...             # services/hub
pnpm docs:build
pnpm slides:build
```

---
layout: center
class: text-center
---

# Thank You

Refactor work is complete when evidence is reproducible.
