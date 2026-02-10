---
name: goyais-web-asset-governance
description: 当任务涉及前端图标、插画、背景素材选型与接入时触发；用于在 Goyais 仓库内按许可白名单获取资源、落库到 web/src/assets、并维护可审计索引与 notices。
---

# goyais-web-asset-governance

为前端开发提供可复用、可分发、可审计的资源治理流程。该 skill 为 AI 工具提供本地资源索引与白名单拉取脚本，避免外链与许可漂移。

## 适用场景

- 新增或替换前端图标、空状态插画、背景素材。
- 需要在不依赖外链的前提下接入视觉资源。
- 需要更新 `RESOURCE_CATALOG.yaml` 与 `THIRD_PARTY_NOTICES.md`。

## 非适用场景

- 纯后端逻辑改动，不涉及前端资源。
- 仅文本样式微调且不新增素材。
- 无需形成可审计资源产物的一次性实验。

## 输入（需要读取）

- `AGENTS.md`
- `docs/web-ui.md`
- `web/src/design-system/tokens.css`
- `web/src/style.css`
- `web/src/assets/RESOURCE_CATALOG.yaml`
- `web/src/assets/THIRD_PARTY_NOTICES.md`
- `assets/approved-sources.md`
- `assets/asset-catalog-spec.md`
- `assets/notices-entry-template.md`
- `assets/decision-matrix.md`

## 输出（需要产出）

- 新增/更新素材到 `web/src/assets/**`
- 更新 `web/src/assets/RESOURCE_CATALOG.yaml`
- 更新 `web/src/assets/THIRD_PARTY_NOTICES.md`
- 若涉及组件接入，更新相关 Vue 组件与 i18n 文案

## 严格步骤

1. 先查本地资源目录与 `RESOURCE_CATALOG.yaml`，优先复用已有素材。
2. 若本地缺失，仅允许按 `assets/approved-sources.md` 的白名单来源获取。
3. 图标优先执行 `scripts/add-heroicon.sh`，禁止使用未在脚本白名单内的名称。
4. 所有新素材必须落库到 `web/src/assets/**`，禁止运行时外链。
5. 对运行时可见 SVG 执行 token 对齐：不写死语义 hex，优先 `currentColor`。
6. 每次新增第三方素材都必须更新 `THIRD_PARTY_NOTICES.md`。
7. 每次新增素材都必须更新 `RESOURCE_CATALOG.yaml` 并填写完整字段。
8. 变更完成后执行 `scripts/validate-assets.sh`，失败不得交付。

## 输出要求

当 skill 被调用并完成资源接入时，输出必须包含：

- 可直接使用的本地路径列表（`web/src/assets/**`）
- 对应许可条目（来源、license、版本或日期）
- 使用方式（组件名或类名，如 `Icon`、`ui-bg-grid`）

## 快速命令

```bash
# 拉取白名单 Heroicons（默认拉全白名单）
.agents/skills/goyais-web-asset-governance/scripts/add-heroicon.sh

# 只拉取指定 icon
.agents/skills/goyais-web-asset-governance/scripts/add-heroicon.sh home command-line

# 资源治理校验
.agents/skills/goyais-web-asset-governance/scripts/validate-assets.sh
```
