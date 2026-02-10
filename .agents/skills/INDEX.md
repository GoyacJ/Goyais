# Goyais Repo Skills Index

本目录是 Goyais 仓库的 repo skills 入口，服务于 v0.1 冻结决策下的协作、计划、交付与验收。

## 推荐使用顺序

1. `goyais-norms`
   - 先对齐冻结约束与契约边界，避免后续方案偏航。
2. `goyais-progress-next-plan`
   - 先扫描代码与契约证据，再输出下一步详细计划，避免“只看文档”误判。
3. `goyais-project-management`
   - 将需求拆成垂直切片，建立 DoD、验收驱动与风险控制。
4. `goyais-git`
   - 按 GitHub Flow 和 Conventional Commits 落地分支、提交与 PR 评审。
5. `goyais-parallel-threads`
   - 并行 thread 场景下执行“一线程一 worktree”隔离、提交防呆与 master 集成通道。
6. `goyais-fixflow`
   - 修复 bug 时默认先创建独立 worktree，确认后 no-ff 合并 `master` 并自动回收 thread。
7. `goyais-thread2-bootstrap`
   - 启动 Thread #2 工程骨架，优先跑通 single-binary 静态服务验收。
8. `goyais-vertical-slice`
   - 对后续模块重复使用垂直切片模板，确保产出一致可审查。
9. `goyais-single-binary-acceptance`
   - 对单二进制与静态路由/缓存头/Content-Type 做脚本化回归。

## 快速映射

- 需要确认“不能改什么、必须怎么做”：`goyais-norms`
- 需要先看真实实现进度并形成下一步计划：`goyais-progress-next-plan`
- 需要规划里程碑、切片和 DoD：`goyais-project-management`
- 需要发分支、写 commit、提 PR、准备回滚：`goyais-git`
- 需要并行启动多个 thread 且隔离工作区：`goyais-parallel-threads`
- 需要修复 bug 且默认新建 worktree、确认后合并 master：`goyais-fixflow`
- 需要进入 Thread #2 基建落地：`goyais-thread2-bootstrap`
- 需要为某个模块写标准化切片提示词：`goyais-vertical-slice`
- 需要验证 single binary 发布闭环：`goyais-single-binary-acceptance`

## 维护约束

- skills 仅补充执行方法，不得引入与 `AGENTS.md`、`docs/prd.md`、`docs/spec/v0.1.md`、`docs/arch/*`、`docs/api/openapi.yaml`、`docs/acceptance.md` 冲突的新规则。
- 任何涉及契约变化的内容，必须在同一变更中同步更新对应文档。
