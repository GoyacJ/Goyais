# Contract Sync Checklist

## 变更类型 -> 必同步文档

- API 路径/请求/响应/错误/分页变化
  - `docs/api/openapi.yaml`
  - `docs/acceptance.md`

- 实体字段/状态机变化
  - `docs/arch/data-model.md`
  - `docs/arch/state-machines.md`
  - `docs/acceptance.md`

- 可见性/ACL/授权判定变化
  - `docs/arch/overview.md`
  - `docs/arch/data-model.md`
  - `docs/acceptance.md`

- provider 抽象/配置键名/默认值变化
  - `docs/arch/overview.md`
  - `docs/spec/v0.1.md`
  - `docs/acceptance.md`

- 静态路由/缓存头/Content-Type 策略变化
  - `docs/arch/overview.md`
  - `docs/api/openapi.yaml`（如涉及行为契约）
  - `docs/acceptance.md`
