---
name: goyais-vue-asset-governance
description: 管理 vue_web 资源素材的许可、落库、目录索引与 token 对齐。
---

# goyais-vue-asset-governance

## 适用场景

- 新增/替换图标、插画、背景素材。

## 输入

- `vue_web/src/assets/RESOURCE_CATALOG.yaml`
- `vue_web/src/assets/THIRD_PARTY_NOTICES.md`
- `scripts/validate-assets.sh`

## 输出

- 新素材路径与许可记录。
- catalog/notices 更新。

## 严格步骤

1. 优先复用已有素材。
2. 新素材必须落库到 `vue_web/src/assets/**`。
3. 更新 catalog 与 notices。
4. 执行 `validate-assets.sh`。

## 验收

- 无外链运行时依赖，许可记录完整。
