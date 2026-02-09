Goyais PRD（完整详细版 v0.0.1）

项目名称：Goyais
项目定位：全意图驱动的智能 Agent + 多模态 AI 原生编排与执行平台（Go + Vue）
核心关键词：AI/页面双入口一致、Capability 统一规范、插件市场、DAG 可视化画布、MediaMTX 流媒体、算法库、权限绑定与数据隔离、可追溯与可回放

1. 背景与愿景

1.1 背景

多模态处理（视频/图片/音频/文档/Excel/流媒体）在业务中通常由分散脚本、临时流程、人工操作构成，难以复用、难以审计、难以规模化。与此同时，AI 交互已成为用户最自然的入口，但传统平台的“AI 功能”往往与页面操作割裂、权限边界模糊、产物不可追溯，导致不可控与难落地。

1.2 愿景

Goyais 构建一个以 AI 为主入口、同时支持强交互页面编排的多模态平台：
•	用户用对话/语音表达意图，平台自动生成/选择工作流（DAG）并执行
•	所有能力均可被 AI 触发，且与 UI 操作语义一致
•	所有输入、过程、产物、外发调用都可追溯、可回放、可治理
•	工具/技能/MCP/算法包通过统一规范与插件市场形成生态

2. 产品概述

2.1 项目简述

Goyais 是一个以 Go 构建的多模态 AI 编排与执行平台。它接收多模态输入（视频、图片、音频、文档、Excel、流媒体），通过统一的能力注册与 DAG 工作流编排进行处理，输出结构化结果、新资产或诊断报告。平台默认 AI 交互为主入口，所有功能点必须支持 AI 触发；同时支持页面操作并保持一致性。

2.2 核心价值
•	从意图到执行闭环：自然语言 → 结构化意图 → 生成/选型工作流 → 执行 → 产物沉淀
•	生态化扩展：Tool/Skill/MCP/Model/Algorithm 全部统一到 Capability 规范，插件可上架安装
•	流媒体完整能力：以 MediaMTX 为底座，接入与控制、事件驱动、录制资产化
•	工程可控：权限绑定当前用户 + 数据隔离 + 外发闸门 + 全链路审计

3. 目标、范围与里程碑

3.1 v0.1 目标（必须达成）
1.	AI 与 UI 双入口一致：统一通过 Command 执行（Command-first）。
2.	复杂可视化编排画布：可视化构建/编辑 DAG，强校验，运行调试与回放。
3.	统一能力体系：Tool/Skill/MCP/Model/Algorithm 的统一注册、发现与调用。
4.	插件市场 MVP：包上传/下载/安装/启用/禁用/升级/回滚，依赖校验，权限上限治理。
5.	资产体系完备：多模态资产管理、元数据抽取、血缘追踪、产物沉淀。
6.	流媒体接入 MediaMTX：StreamingAsset、控制面 API、录制入库、事件触发工作流。
7.	权限与数据隔离：agent-as-user 执行、RBAC+ACL+Visibility、外发闸门、审计。
8.	算法库 MVP：至少 2 个产品化算法包可运行并输出结构化结果+资产。

3.2 v0.1 不做（延后）
•	多人实时协同编辑画布（先做草稿锁）
•	复杂宏/函数式子图复用（先做 Algorithm/子流程引用雏形）
•	完整计费结算
•	深度供应链安全（SBOM/漏洞扫描）可预留接口
•	跨地域容灾、多集群联邦调度

3.3 里程碑建议
•	M0 基础框架：租户/用户/权限、Command 骨架、资产上传与元数据、审计框架
•	M1 闭环跑通：画布编辑+校验+发布，workflow 引擎执行+回放，AI 入口生成 command 并运行
•	M2 生态与流媒体：插件市场 MVP，MediaMTX 接入与事件触发，算法库上架与复用
•	M3 增强：更多控制节点、断点调试、搜索与推荐、权限审批流（可选）

4. 用户画像与使用场景

4.1 用户角色
•	平台管理员：租户/工作区/用户/角色/策略，插件上架与安装治理
•	业务操作员：上传资产、运行算法/工作流、查看报告与产物
•	算法/工具开发者：发布 capability/algo 包，维护版本与依赖
•	只读观察者：查看资产、报告、运行记录（受可见性限制）

4.2 典型场景
1.	意图驱动视频分析：用户说“用这个视频生成异常巡检报告”→ 自动选 workflow → 执行 → 产出报告+结构化异常列表
2.	流媒体实时分析：接入 RTSP/RTMP/SRT → onPublish 事件触发 run → 每 10 秒切片分析 → 告警与回放资产
3.	画布可视化编排：开发者在画布拖拽节点→连线→配置→发布模板/算法→团队复用
4.	插件生态：管理员安装“某目标检测工具包/算法包/MCP provider”→ Registry 生效→AI/画布都可用
5.	权限与隔离：不同工作区互不可见；某资产共享给特定角色可读但不可写/不可外发

5. 产品原则（强约束）
    1.	Agent-as-User：AI 永远是当前登录用户的代理，不拥有独立超权限。
    2.	Command-first：AI 与 UI 最终都落到统一 Command；Command 是授权/审计最小单元。
    3.	Schema 驱动：所有能力输入输出均 schema 化；参数表单与校验自动生成。
    4.	全链路可追溯：run/step、工具调用、外发、产物、血缘完整记录，支持回放。
    5.	统一隔离与可见性：所有对象统一支持 PRIVATE/WORKSPACE/TENANT/PUBLIC + ACL 共享。
    6.	安全外发控制：敏感数据默认禁止原文外发，只允许摘要/脱敏/特征（策略化）。
    7.	可扩展生态：能力以包分发，具备版本、依赖、权限声明、可治理与可审计。

6. 核心概念与术语
   •	Asset：统一资产，包含多模态文件、结构化结果、报告、录制回放等
   •	StreamingAsset：流媒体资产（MediaMTX path/endpoint 状态等）
   •	Intent：用户意图的结构化表示（goal/inputs/constraints/preferences）
   •	WorkflowTemplate：DAG 模板定义（可发布版本）
   •	WorkflowRun：模板实例化后的运行（绑定参数与资产）
   •	StepRun：节点执行实例（输入输出、日志、耗时、产物）
   •	Capability：统一能力（Tool/Model/Skill/MCP 映射工具）
   •	Algorithm：产品化能力组合体（WorkflowTemplate+约束+默认参数+依赖）
   •	PluginPackage：插件包（manifest + capabilities + runtime + docs + signatures）
   •	ContextBundle：统一上下文索引（引用+摘要+检索），服务 AI 与回放
   •	Command：平台动作指令（AI/UI 统一执行入口）
   •	Visibility：可见性等级（PRIVATE/WORKSPACE/TENANT/PUBLIC）
   •	ACL/Share：共享规则（主体 + 权限 + 可选过期）

7. 功能需求（详细）

7.1 AI 入口（对话/语音）

7.1.1 功能列表
•	会话管理：新建/列表/搜索/归档（可选）
•	输入支持：文本（v0.1 必须）、语音（可选或与 ASR 联动）
•	意图解析：生成 IntentDraft（包含目标、资产引用、约束、偏好）
•	计划生成：基于可用能力与策略，选择 Algorithm/WorkflowTemplate 或生成新 workflow（受控）
•	Command 生成：AI 只能输出 Command 列表 + 参数，不得直接调用内部 API
•	执行与流式反馈：运行进度、节点状态、关键日志摘要、产物链接实时推送
•	异常处理：权限不足/资源不可读/参数不合规时给出原因与替代建议
•	上下文融合：将本次执行关键结构化结果写入 ContextBundle（可配置写入策略）

7.1.2 交互与限制
•	AI 输出必须可解释：展示“将执行哪些命令、涉及哪些资源”
•	高危命令（用户/权限变更、PUBLIC 发布、跨租户共享等）默认需策略限制（v0.1 可先禁止或仅管理员允许）
•	所有 AI 行为携带用户身份与 policyVersion，并写入审计

7.2 Command 系统（AI/UI 统一执行层）

7.2.1 目标
•	统一所有平台动作入口，实现一致性、鉴权与审计；防止 AI 直接越权调用内部服务。

7.2.2 Command 类型（示例）
•	资产：asset.upload / asset.read / asset.update / asset.tag / asset.share / asset.delete
•	工作流：workflow.createDraft / workflow.patch / workflow.publish / workflow.run / workflow.cancel / workflow.rerun
•	算法：algo.run / algo.publish / algo.install
•	能力/插件：plugin.upload / plugin.install / plugin.enable / plugin.disable / plugin.rollback
•	流媒体：stream.create / stream.updateAuth / stream.record.start / stream.record.stop / stream.kick
•	用户权限（高危）：user.create / role.bind / policy.update（v0.1 建议仅管理员或默认关闭）

7.2.3 执行管道（强制）
1.	Validate：schema 校验、字段约束
2.	Authorize：动作权限 + 资源权限 + 可见性 + ACL + 外发策略（必要时）
3.	Execute：执行命令并产生事件/运行任务
4.	Audit：记录请求、授权结果、执行结果、影响资源与产物
5.	Event：向 UI/WS 推送进度与结果（可选）


7.3 权限模型（Agent-as-User + RBAC + ACL + Egress）

7.3.1 基本规则
•	AI 永远以当前用户身份执行：ExecutionContext(userId, tenantId, workspaceId, roles, policyVersion, traceId)
•	所有工具、工作流、算法执行都必须携带 ExecutionContext

7.3.2 权限分层
•	Action 权限（RBAC）：能否执行某类命令/动作（例如 workflow.run）
•	Resource 权限（ACL/Ownership）：能否对特定对象读/写/执行/管理
•	Egress 权限：是否允许把数据发送到外部模型/第三方服务（按分级与策略）

7.3.3 授权闸门（必经）
•	Command Gate（动作闸）
•	Resource Gate（资源闸：资产、流、workflow、capability 等）
•	Tool Gate（工具闸：step 执行前再校验一次，防绕过）
•	Egress/Data Gate（外发闸：敏感数据、脱敏、摘要策略）

7.3.4 审计要求
•	用户身份、策略版本、命令、授权 allow/deny 原因、工具调用、产物、外发摘要必须记录

7.4 数据隔离与可见性（全对象统一）

7.4.1 可见性枚举（全对象通用）
•	PRIVATE：仅 owner 可见（可共享）
•	WORKSPACE：工作区内可见
•	TENANT：租户内可见
•	PUBLIC：公开可见（跨租户可发现/可读，但执行仍受权限限制）

PUBLIC 只能由具备 PUBLISH_PUBLIC 权限的角色设置。

7.4.2 共享（ACL）
•	subject：user/role/workspace（v0.1 可先 user+role）
•	permissions：READ/WRITE/EXECUTE/MANAGE/SHARE
•	可选：过期、是否可转授权（默认否）

7.4.3 覆盖对象（必须全支持）
•	Asset / StreamingAsset / Artifact / Report
•	CapabilityProvider / Capability / PluginPackage / Algorithm
•	WorkflowTemplate / WorkflowRun / StepRun / RunLog / ContextBundle

7.4.4 产物可见性继承规则（强制）
输出产物（Asset/Report/Artifact）必须支持继承策略：
•	INHERIT_FROM_RUN（默认推荐）
•	INHERIT_FROM_INPUTS（取最严格：最小可见性原则）
•	EXPLICIT（显式指定，需权限与策略允许）

默认禁止：输入 PRIVATE → 输出 PUBLIC（除非策略显式允许并具备权限）。

7.4.5 统一授权判断顺序（建议）
Tenant Gate → Visibility Gate → ACL Gate → RBAC Gate → Egress Gate

7.5 资产系统（Asset + 血缘）

7.5.1 支持类型
•	视频/图片/音频/文档/Excel
•	结构化资产：json、embedding、索引、诊断结果
•	报告：pdf/html/markdown（以 asset 形式存储）
•	流媒体资产：StreamingAsset（详见 7.9）

7.5.2 功能列表
•	上传：分片/断点（可选）、校验 hash、存储到 S3/MinIO
•	元数据抽取：视频/音频 probe（编码、时长、分辨率）
•	标签/检索：tag、全文（可选）、过滤（按可见性/创建者/类型）
•	权限与共享：visibility + ACL
•	血缘：记录由哪个 run/step 生成（lineage graph）


7.6 Workflow 系统（模板、运行、回放）

7.6.1 WorkflowTemplate（模板）
•	DAG 定义（节点、端口、连线、参数 schema）
•	版本管理：Draft（可不完整）→ Publish（不可变版本）
•	可见性与共享：同数据隔离规则
•	依赖：引用的 capability/algo 版本列表

7.6.2 WorkflowRun（运行实例）
•	实例化模板版本 + 绑定 inputs/params + 继承 visibility 策略
•	状态机：pending/running/success/failed/canceled
•	StepRun：输入输出、日志、耗时、重试信息、产物引用

7.6.3 引擎执行要求
•	并发执行、队列调度、可水平扩展 worker
•	重试策略（指数退避、最大次数）
•	幂等控制（避免重复副作用）
•	执行前与 step 前权限二次校验
•	全链路事件推送（WS/SSE）用于 UI 实时刷新

8. 复杂可视化编排画布（核心模块）

v0.1 必须支持“复杂可视化编排画布”，不再仅模板+参数面板。

8.1 画布目标
•	图形化创建/编辑/调试/发布 DAG
•	强校验：端口类型、必填输入、无环、schema 约束
•	可调试：单步、从节点运行、查看中间产物与日志、回放叠加状态
•	与 AI 一致：AI 输出 workflow patch，画布可视化差异并复用同校验

8.2 画布 UI 架构
•	左侧：节点/能力库（按 Tool/Model/Algorithm/Control 分类，可搜索）
•	中央：画布（拖拽、连线、缩放、框选、多选、撤销重做、minimap、自动布局）
•	右侧：属性面板（节点配置、连线配置、模板配置、校验错误列表）
•	底部：运行/日志/产物面板（可折叠）

8.3 节点类型（v0.1 必须）
1.	Input Node：定义 workflow inputs schema
2.	Tool Node：引用 tool capability
3.	Model Node：引用 model capability
4.	Algorithm Node：引用算法库（子流程封装）
5.	Transform Node：字段映射/拼装（JSONPath/JQ 风格，受控表达式）
6.	Control Nodes：If/Else、Parallel、Join（汇聚）
7.	Output Node：定义 outputs schema 与映射

8.4 端口与连线语义（Typed Ports）
•	端口类型：AssetRef、StreamRef、Text、Json、Number、Boolean、List 等
•	连接规则：类型兼容才允许连线，不兼容给出原因并推荐插 Transform
•	禁止成环：DAG 必须无环
•	必填输入未满足：禁止发布/运行（允许保存草稿）

8.5 参数配置（Schema Form）
•	节点配置表单由 schema 自动生成（JSON Schema）
•	支持赋值来源：常量 / 引用上游输出（可视化选择）/ 受控表达式
•	配置预览：展示最终 resolved 输入（调试友好）

8.6 校验体系（强制）
•	静态校验（编辑时）：无环、类型匹配、必填、schema、权限提示
•	发布校验：依赖版本存在、全校验通过、不可变快照
•	运行时校验：执行前再次校验（含权限/可见性/ACL）

8.7 调试与回放（画布内）
•	Run（全量）
•	Run from here（从某节点起）
•	Test node（单节点试运行，可选）
•	回放：选择历史 run，画布叠加节点状态/耗时；点击节点查看当次输入输出与产物链接

8.8 AI 协同（受控 patch）
•	AI 只输出 workflow patch（新增/删除节点、改参数、改连线）
•	patch 需通过校验
•	画布高亮差异（新增绿色、修改黄色、删除红色）
•	patch 本质是 Command：workflow.patch

8.9 画布验收标准（v0.1）
1.	能创建 Input→Tool→Model→Output 并保存、发布、运行
2.	类型不匹配无法连线并提示原因；可插入 Transform 解决
3.	运行时节点状态实时刷新；可查看日志与产物引用
4.	回放历史 run 叠加状态与耗时
5.	AI 修改画布生成 patch 并可视化差异，且受权限与校验约束

9. 统一能力体系（Tool / Skill / MCP / Model）

9.1 统一抽象：Capability
•	CapabilityProvider：能力提供者（remote/http、container、mcp server）
•	Capability：能力定义（kind=tool/model/skill/algorithmRef 等）
•	Invoke Contract：统一调用协议（inputSchema/outputSchema、timeout、resources、requiredPermissions、egressPolicy）

9.2 Tool
•	形态：HTTP/gRPC、容器任务、命令包装、函数
•	必须声明：schema、版本、超时、资源、权限需求、外发策略

9.3 Model
•	作为 capability 的一种：LLM/VLM/ASR/TTS/Embedding
•	必须声明：输入输出、成本标签（可选）、数据外发策略（关键）

9.4 Skill
•	高阶策略能力：面向意图，通常内部调用 algorithm/tool/model
•	对编排器暴露：适用场景、必需输入、可选参数、输出结构

9.5 MCP
•	MCP Server 作为 CapabilityProvider
•	将 MCP 工具映射为 Capability，纳入同样权限与审计

10. 插件市场（Plugin Market）

10.1 包规范：Goyais Package

建议结构（示例）：

```bash
manifest.yaml
capabilities/tools/*.yaml
capabilities/skills/*.yaml
capabilities/algorithms/*.yaml
runtime/docker/...
docs/...
signatures/...
```

manifest 必含：
•	id、version（SemVer）、type（tool-provider/skill-pack/algo-pack/mcp-provider）
•	capabilities 列表（name、kind、schemaRef、requiredPerms、resources、egressPolicy）
•	runtime（镜像/endpoint）
•	permissions（安装要求与运行上限建议）
•	security（签名信息，v0.1 可预留）

10.2 市场能力（v0.1 必须）
•	上传/下载/安装/启用/禁用
•	升级/回滚
•	依赖校验（缺失则提示并引导安装）
•	安装范围：Workspace / Tenant
•	权限上限（ceiling）：插件声明权限不得超过管理员授予上限
•	审计：安装/启用/回滚记录

10.3 可见性与隔离
•	市场可见性：PUBLIC / TENANT / WORKSPACE
•	安装后可见性：默认 WORKSPACE 或 TENANT（可配置）
•	PUBLIC 仅允许具备权限的管理员发布

11. 流媒体模块（MediaMTX）

11.1 StreamingAsset 定义
•	protocol：RTSP/RTMP/SRT/WebRTC/HLS
•	streamId/path：对应 MediaMTX 路径
•	source：push/pull
•	endpoints：播放/推流 URL 列表
•	state：online/offline、viewers、bitrate、codecs
•	visibility + ACL：与全平台一致

11.2 平台控制面能力（要求覆盖 MediaMTX 完整能力）
•	创建/删除/更新 path
•	推流/拉流鉴权（用户 token 或工作区密钥映射）
•	状态查询（连接、码率等）
•	踢流/断开连接
•	录制控制：开始/停止 → 录制文件入 Asset 并写 lineage
•	事件接入：onPublish/onRead/onConnect/onRecordFinish → 事件总线 → 触发 workflow

11.3 与工作流结合
•	StreamSourceNode：从流窗口化切片/抽帧生成临时资产
•	TriggerNode：订阅事件触发 run
•	规则：只有当用户/工作区具备 execute 权限且流可见时才允许触发

12. 算法库（Algorithm Library）

12.1 定义

Algorithm = 产品化能力组合体：
•	WorkflowTemplate + 默认参数 + 约束 + 依赖 +（可选评估指标）

12.2 要求
•	版本管理（SemVer）
•	依赖声明（capability/model 版本）
•	输入输出 schema 稳定
•	可上架（插件市场）
•	可调用：
•	作为画布节点 Algorithm Node
•	作为 AI 可直接运行的能力 algo.run

12.3 v0.1 示例算法（至少 2 个）
•	algo.video.anomaly_report@1：抽帧 → VLM 检测 → 聚合 → 报告
•	algo.audio.meeting_minutes@1：ASR → 摘要 → 结构化纪要

13. 上下文分层模型（ContextBundle）

13.1 Run Context（运行上下文，强推荐默认开启）

范围：一次 WorkflowRun 及其 StepRun
内容：
•	每个 step 的结构化输出（JSON）
•	关键产物引用（assetId、artifactId）
•	自动摘要（step summary，几十到几百字）
•	错误与诊断信息（stack/exit code 等）

用途：
•	工作流内后续节点复用
•	AI 解释“我做了什么、为什么失败”
•	回放与可追溯

这层是“确定性上下文”，质量最高，应该优先使用。

13.2
Session Context（会话上下文）

范围：同一个用户会话（对话窗口）
内容：
•	用户意图演进（IntentDraft 历史）
•	用户偏好（模型选择、输出格式）
•	与本次任务相关的资产清单（引用）
•	对话摘要（而不是完整对话全文）

用途：
•	多轮对话持续执行同一任务
•	AI 能记住“用户刚才说的约束”

这里要强制做“对话摘要 + 关键引用”，避免 prompt 膨胀。


13.3 Knowledge Context（知识上下文）

范围：Workspace/Tenant 级知识库（RAG 主战场）
内容来源：
•	文档/Excel/报告资产的解析文本与结构化表格
•	历史 run 的报告摘要与结构化结果（可选择写入）
•	插件/算法/工作流的说明文档（docs）
•	业务术语表、规则、SOP（最好单独维护）

用途：
•	跨任务复用知识
•	企业内“规则/流程/文档”问答与辅助编排


13.4 目标

统一 ContextBundle：存什么、不存什么

建议你实现一个实体：ContextBundle（可挂在 Run 或 Session 上），其内容不是大文本，而是“索引化材料”：

推荐字段（概念）
•	bundleId, scopeType (RUN|SESSION|WORKSPACE), scopeId
•	facts[]：结构化事实（JSON）+ 来源引用（stepId/assetId）
•	summaries[]：摘要块（短文本）+ 来源引用
•	refs[]：资产引用列表（assetId + role：input/output/intermediate）
•	embeddingsIndexRefs[]：向量索引引用（指向知识库条目）
•	timeline[]（多模态可选）：时间轴事件（ts、type、desc、refs）

禁忌
•	不要把原始视频/长文全文直接塞 bundle（成本、泄露、不可控）
•	不要默认永久存全量聊天记录（噪声与合规风险）


14. UI 页面清单（v0.1）
    1.	AI 工作台：对话/语音、命令计划展示、运行流式反馈
    2.	画布编排器：新建/编辑/校验/发布/调试/回放
    3.	资产中心：上传/预览/标签/权限/血缘
    4.	运行中心：Run 列表、详情、step 状态、日志、产物
    5.	算法库：列表、详情、输入面板、运行与结果
    6.	流媒体中心：流列表、播放/推流信息、状态、录制、触发规则
    7.	插件市场：浏览、安装、启用、版本、依赖、权限声明
    8.	权限管理：用户/角色/策略（v0.1）

15. 数据模型（核心实体）
    •	Tenant / Workspace / User / Role / Policy（policyVersion）
    •	Asset（含 lineage、visibility、owner、metadata）
    •	StreamingAsset（映射 MediaMTX path、状态、endpoints）
    •	CapabilityProvider / Capability（kind、schema、version、permissions、egressPolicy）
    •	PluginPackage / InstallRecord
    •	WorkflowTemplate（draft/published、graph、schemas、uiState、validation）
    •	WorkflowRun / StepRun（inputs/outputs、artifacts、logs、timing）
    •	Algorithm（templateRef、constraints、defaults、dependencies）
    •	Command / CommandResult
    •	AuditEvent
    •	ContextBundle

16. 非功能需求（NFR）

16.1 性能与可靠性
•	worker 水平扩展；支持并发 run
•	step 超时、重试、幂等
•	关键路径：上传、运行、日志/事件推送稳定

16.2 可观测性
•	Trace：run/step/invoke
•	Log：结构化日志与原始日志（至少摘要可查）
•	Metrics：队列堆积、成功率、耗时分布、资源使用（可选）

16.3 安全
•	强鉴权（JWT/OIDC）
•	全链路授权闸门
•	外发策略：敏感数据默认禁止原文外发
•	审计不可篡改（至少追加写）


17. 风险与对策
    1.	画布控制节点复杂度：If/Parallel/Join 易引入数据结构混乱
          •	对策：v0.1 先定义严格 join schema；复杂循环延后
    2.	插件安全与越权：第三方包风险大
          •	对策：容器隔离优先；权限 ceiling；安装审计；后续签名/扫描
    3.	流媒体状态一致性：事件与状态可能漂移
          •	对策：以 MediaMTX 状态为准，平台缓存短 TTL；事件进入总线统一处理
    4.	上下文膨胀与成本：prompt 过载
          •	对策：索引+摘要+引用；按需检索
    5.	AI 误生成危险命令
          •	对策：Command 白名单+schema 校验+权限闸；高危命令默认禁用或管理员策略审批

18. 验收标准（v0.1 总验收）
    1.	AI 与 UI 一致：同一操作（运行 workflow、安装插件、创建流）在 AI 与 UI 产生同形 Command，执行结果一致。
    2.	画布可用：可视化创建 DAG、强校验、发布、运行、回放、查看日志与产物。
    3.	能力统一：Tool/Model/Skill/MCP 至少一种接入可用并能在画布/AI 中调用；Capability Registry 可查询版本与 schema。
    4.	插件市场 MVP：插件包上传→安装→启用→能力生效→可回滚；依赖校验与权限 ceiling 生效。
    5.	流媒体能力：通过 MediaMTX 创建流、鉴权、查看状态、录制生成资产；onPublish 触发一次分析 run。
    6.	算法库 MVP：至少 2 个算法包可运行，输出结构化结果 + 报告资产。
    7.	数据隔离：资产/工作流/算法/能力等均支持 PRIVATE/WORKSPACE/TENANT/PUBLIC 与共享；默认隔离正确；跨租户除 PUBLIC/显式共享不可访问。
    8.	安全与审计：任意 run 的命令、授权决策、工具调用、外发记录、产物血缘可追溯；越权访问必失败并记录原因。

19. 建议的实现优先级（工程落地顺序）
    1.	统一对象模型（visibility + ACL）与 Authorize 中间件（全对象通用）
    2.	Command 系统（Validate/Authorize/Execute/Audit）
    3.	Registry（Capability/Workflow/Algorithm）+ schema 驱动表单基础
    4.	画布（节点/端口/连线/校验/发布）+ 引擎执行与回放
    5.	资产系统（上传、元数据、血缘）
    6.	插件市场（包管理 + 安装记录 + 权限 ceiling）
    7.	MediaMTX（StreamingAsset + 控制面 API + 事件触发 + 录制入库）
    8.	算法库（2 个示例算法产品化 + 上架）