# Goyais Release Checklist（RC/GA）

- 版本：v0.1
- 更新日期：2026-02-25
- 适用范围：Desktop + Hub + Worker 联合发布
- 责任人：Release Owner（平台工程）
- 签字记录模板：`docs/reviews/2026-02-25-release-signoff-record.md`

## 1. Go/No-Go 结论

- 结论字段：`GO` / `NO-GO`
- 发布批次：`<release-tag>`
- 记录时间：`<yyyy-mm-dd hh:mm>`
- 记录人：`<name>`

## 2. 代码质量门禁（必须全部通过）

1. Hub：`cd services/hub && go test ./... && go vet ./...`
2. Worker：`cd services/worker && uv run ruff check . && uv run pytest -q`
3. Desktop：`cd apps/desktop && pnpm lint && pnpm test && pnpm coverage:gate`

验收标准：
- 任一命令失败即 `NO-GO`。
- 命令输出需保留在发布记录中（CI 链接或日志归档）。

## 3. 安全与隔离门禁（必须全部通过）

1. 控制面接口默认鉴权：未认证请求访问管理接口返回 401/403。
2. 跨 workspace 访问控制：越权请求返回 403。
3. Hub/Worker 内部 token：未配置时（非不安全模式）服务拒绝启动或拒绝请求。
4. Worker 命令执行：危险命令与 shell 注入模式被拒绝。

验收标准：
- 任一项回归失败即 `NO-GO`。

## 4. 运行与配置门禁

1. 环境变量齐备：
   - `HUB_INTERNAL_TOKEN`
   - `WORKER_INTERNAL_TOKEN`
   - 其他部署必需项（见各服务 README）
2. 不安全开关核对：生产环境不得启用 `GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN=1`。
3. Desktop 与 Hub API 模式一致（strict/non-strict 配置一致）。

## 5. 数据与行为一致性门禁

1. SSE 连接与重连正常，事件可持续消费。
2. `last_event_id` 失效时客户端可触发 resync。
3. execution patch 仅包含 execution 关联变更。

## 6. 文档与流程门禁

1. PRD 已更新且与本次发布范围一致（`docs/PRD.md`）。
2. 变更说明已完成（含风险、回滚、影响面）。
3. 值班与回滚负责人已确认。

## 7. 回滚预案（必须预先确认）

1. 回滚触发条件：
   - 鉴权回归
   - 事件链路中断
   - 高优先级故障（P0/P1）
2. 回滚步骤：
   - 回退到上一个稳定 tag
   - 恢复上一版 Hub/Worker 运行配置
   - 验证健康检查与关键接口
3. 回滚验收：关键链路恢复且错误率回落到基线。

## 8. 最终发布签字

- Tech Lead：`<name/date>`
- Platform/Release Owner：`<name/date>`
- PM（范围确认）：`<name/date>`

---

发布规则：
- 仅当第 2~8 节全部满足时可标记 `GO`。
- 任何一项未满足，必须标记 `NO-GO` 并记录整改项与计划时间。
