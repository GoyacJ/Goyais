# Goyais 发布签字记录（2026-02-25）

- 记录日期：2026-02-25
- 发布版本/Tag：`<待填写>`
- 环境：`<待填写：staging/prod>`
- 发布单链接：`<待填写>`

## 1. Go/No-Go 决议

- 最终决议：`<GO / NO-GO>`
- 决议时间：`<yyyy-mm-dd hh:mm TZ>`
- 决议说明：`<待填写>`

## 2. 技术门禁证据（已完成）

证据来源：`docs/reviews/2026-02-25-rc-rehearsal-record.md`

1. Hub：`go test ./... && go vet ./...` 通过
2. Worker：`uv run ruff check . && uv run pytest -q` 通过（41 passed）
3. Desktop：`pnpm lint && pnpm test && pnpm coverage:gate` 通过（108 passed，coverage gate OK）
4. 关键安全/一致性回归：
   - Hub workspace 授权边界关键用例通过
   - Worker internal token / command guard 关键用例通过
   - Desktop SSE/合并/resync 关键用例通过

## 3. 发布前人工确认（待签字）

1. 生产环境变量与密钥配置复核完成（含 `HUB_INTERNAL_TOKEN` / `WORKER_INTERNAL_TOKEN`）。
2. 生产环境未启用不安全开关（如 `GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN=1`）。
3. 值班、回滚与升级窗口确认完成。

## 4. 回滚信息（待填写）

- 回滚目标版本：`<待填写>`
- 回滚执行人：`<待填写>`
- 回滚触发阈值：`<待填写>`

## 5. 签字区

- Tech Lead：`<姓名 / 日期 / 结论>`
- Release Owner：`<姓名 / 日期 / 结论>`
- PM：`<姓名 / 日期 / 结论>`

---

说明：
- 若任一签字结论为 NO-GO，本记录应附带整改项与预计完成时间。
- 本记录与 `docs/release-checklist.md` 一起归档到同一发布单。
