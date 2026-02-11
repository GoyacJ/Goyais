## Summary | 变更摘要

- 

## Scope | 影响范围

- [ ] Go backend (`go_server`)
- [ ] Vue web (`vue_web`)
- [ ] Docs / Governance
- [ ] Other

## Motivation & Context | 背景与动机

-

## Contract Sync Checklist | 契约同步检查

若涉及 API/数据模型/状态机/权限策略变更，请确认：

- [ ] Updated `go_server/docs/api/openapi.yaml`
- [ ] Updated `go_server/docs/arch/overview.md`
- [ ] Updated `go_server/docs/arch/data-model.md`
- [ ] Updated `go_server/docs/arch/state-machines.md`
- [ ] Updated `go_server/docs/acceptance.md`

## Validation | 验证证据

- [ ] `bash go_server/scripts/git/precommit_guard.sh`
- [ ] `bash go_server/scripts/ci/source_header_check.sh`
- [ ] `bash go_server/scripts/ci/contract_regression.sh`

执行结果 / Notes:

```text
paste key command outputs here
```

## Risk & Rollback | 风险与回滚

- Risk:
- Rollback plan:

## Additional Notes | 补充说明

-
