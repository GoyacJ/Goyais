# Runtime Agent 重构计划：真正的 Agentic Loop

## Context

### 问题
当前 `runtime/python-agent/` 的 agent 层是一个**线性管道**（plan_node → patch_node → END），而非真正的 agentic loop：
- LLM **不能调用工具** — 工具存在但 LLM 无法在执行过程中使用它们
- **无反馈循环** — 工具结果不会重新进入 LLM 上下文
- **硬编码 README.md** — agent 只能读写 README.md
- **OpenAI Provider 错误** — 使用了错误的 API（`responses.create` + `input` 参数）
- **Skills/MCP 不可执行** — 仅加载元数据，无法实际调用
- **提示词过于简单** — 仅 1-2 句话的系统提示

### 目标
基于 Claude Code 的架构模式（learn-claude-code），重构为**真正的 agentic loop**：LLM 主动调用工具、获取结果、循环推理直到任务完成。实现两种可切换的后端：
1. **Vanilla Loop** — 纯 while-loop，参考 Claude Code 的不可变循环模式
2. **LangGraph ReAct** — 基于 LangGraph StateGraph 的 ReAct 循环

通过 `GOYAIS_AGENT_MODE` 环境变量切换：`vanilla`（默认）| `langgraph`

### 保留不变的组件
| 组件 | 文件 | 原因 |
|------|------|------|
| HubReporter | `app/services/hub_reporter.py` | 事件批量上报，设计良好 |
| WorktreeManager | `app/services/worktree_manager.py` | Git worktree 隔离 |
| PathGuard | `app/security/path_guard.py` | 路径沙箱 |
| CommandGuard | `app/security/command_guard.py` | 命令白名单 |
| 工具实现 | `app/tools/*.py` | file_tools, command_tools, patch_tools 实现正确 |
| 内部 API | `app/api/internal_executions.py` | Hub 合约不变 |
| DB 层 | `app/db/` | aiosqlite 仓库 |
| 配置/认证 | `app/config.py`, `app/security/auth_token.py` | 环境变量 |

---

## 实现计划

### Phase 1：共享基础层（两种模式共用）

#### 1.1 消息类型 — `app/agent/messages.py`（新建）

统一的消息数据结构，两种后端共用：

```python
@dataclass(slots=True)
class ToolCall:
    id: str           # 唯一 call ID
    name: str         # 工具名
    input: dict       # 参数

@dataclass(slots=True)
class ToolResult:
    tool_call_id: str
    content: str      # 序列化输出
    is_error: bool = False

@dataclass(slots=True)
class Message:
    role: str         # "user" | "assistant" | "tool_result"
    content: str = ""
    tool_calls: list[ToolCall]   # assistant 的工具调用
    tool_results: list[ToolResult]  # 工具执行结果

@dataclass(slots=True)
class ChatResponse:
    stop_reason: str  # "end_turn" | "tool_use" | "max_tokens"
    text: str
    tool_calls: list[ToolCall]
    usage: dict[str, int]  # {input_tokens, output_tokens}
```

#### 1.2 Provider 重写 — `app/agent/providers/`

**`base.py`** — 从 text-in/text-out 改为 messages + tools + streaming：

```python
@dataclass(slots=True)
class ToolSchema:
    name: str
    description: str
    input_schema: dict  # JSON Schema

@dataclass(slots=True)
class ChatRequest:
    model: str
    system_prompt: str
    messages: list[Message]
    tools: list[ToolSchema] = field(default_factory=list)
    max_tokens: int = 4096
    temperature: float = 0.0

class ProviderAdapter(ABC):
    async def chat(self, request: ChatRequest) -> ChatResponse: ...
    async def chat_stream(self, request: ChatRequest) -> AsyncIterator[dict]: ...
```

**`anthropic_provider.py`** — 使用原生 Messages API + tool_use：
- `messages.create()` 带 `tools` 参数
- 解析 `tool_use` content block 为 `ToolCall`
- 解析 `stop_reason` 映射（`"tool_use"` → `"tool_use"`, `"end_turn"` → `"end_turn"`）
- 流式：使用 `messages.stream()` 或 `with client.messages.stream(...) as stream`

**`openai_provider.py`** — 修复为 Chat Completions API：
- 使用 `chat.completions.create()`（而非当前错误的 `responses.create()`）
- `tools` 参数使用 `{"type": "function", "function": {...}}` 格式
- 解析 `finish_reason` 映射（`"tool_calls"` → `"tool_use"`, `"stop"` → `"end_turn"`）
- 流式：使用 `stream=True` + 解析 SSE chunks

**`provider_router.py`** — 最小改动，返回新的 ProviderAdapter：
- `OPENAI_COMPATIBLE_PROVIDERS` 列表保持不变
- `build_provider()` 签名不变，仅内部构造新类型

#### 1.3 工具注册表 — `app/agent/tool_registry.py`（新建）

工具定义 + 派发映射，两种模式共用：

```python
@dataclass(slots=True)
class ToolDef:
    name: str
    description: str
    input_schema: dict          # JSON Schema
    handler: Callable           # 同步或异步
    requires_confirmation: bool = False

class ToolRegistry:
    def register(self, tool: ToolDef) -> None
    def get(self, name: str) -> ToolDef | None
    def to_schemas(self) -> list[ToolSchema]  # 给 LLM 看的
    async def execute(self, name: str, args: dict) -> str
```

`build_builtin_tools(workspace_path: str) -> list[ToolDef]` 函数：
- `read_file` — 绑定 `file_tools.read_file`，不需确认
- `write_file` — 绑定 `file_tools.write_file`，需确认
- `list_dir` — 绑定 `file_tools.list_dir`，不需确认
- `search_in_files` — 绑定 `file_tools.search_in_files`，不需确认
- `apply_patch` — 绑定 `patch_tools.apply_patch`，需确认
- `run_command` — 绑定 `command_tools.run_command`，需确认

扩展 `tool_injector.py` — 从 Hub 加载 Skills 时返回 `ToolDef`（含 callable wrapper），不再仅返回元数据 dict。

#### 1.4 系统提示词 — `app/agent/system_prompts.py`（新建）

替代当前仅 2 行的 `prompts.py`，构建丰富的上下文感知提示词：

```python
def build_system_prompt(
    *,
    mode: str,              # "plan" | "auto"
    workspace_path: str,
    project_summary: str,   # 项目结构、技术栈
    tool_registry: ToolRegistry,
    skill_descriptions: list[str] | None = None,
) -> str
```

包含：
- **角色定义**：你是 Goyais，专业的软件工程助手
- **模式指令**：plan 模式只读分析 → auto 模式直接执行
- **工具文档**：每个工具的名称、描述、参数说明
- **技能描述**：Layer 1 简要摘要在 system prompt 中（注意力预算）
- **安全约束**：不修改工作区外文件，不执行危险命令
- **项目上下文**：工作区路径、关键文件

#### 1.5 上下文管理 — `app/agent/context_manager.py`（新建）

参考 Claude Code 的三层压缩策略：

```python
def estimate_tokens(messages: list[Message]) -> int
    # 粗估：total_chars / 3.5

def truncate_old_tool_results(messages, *, max_chars=2000, preserve_last_n=6)
    # 微压缩：截断旧工具结果

async def compact_context(messages, *, provider, model, token_limit)
    # 自动压缩：保留首尾，中间用 LLM 总结
```

---

### Phase 2：Vanilla Loop 后端

#### 2.1 核心循环 — `app/agent/loop.py`（新建）

参考 Claude Code 的不可变 while-loop 模式：

```python
@dataclass(slots=True)
class LoopCallbacks:
    on_tool_call: Callable[[ToolCall, bool], Awaitable[None]] | None = None
    on_tool_result: Callable[[str, str, bool], Awaitable[None]] | None = None
    on_text_delta: Callable[[str], Awaitable[None]] | None = None
    on_confirmation_needed: Callable[[ToolCall], Awaitable[bool]] | None = None

async def agent_loop(
    *,
    provider: ProviderAdapter,
    model: str,
    system_prompt: str,
    messages: list[Message],
    tools: ToolRegistry,
    callbacks: LoopCallbacks | None = None,
    max_iterations: int = 100,
) -> ChatResponse:
```

**核心逻辑（~80 行）**：
```
while iteration < max_iterations:
    response = await provider.chat(request)
    messages.append(assistant_message)

    if response.stop_reason != "tool_use":
        return response          # 唯一退出条件

    for each tool_call in response.tool_calls:
        callback.on_tool_call(tc)
        if needs_confirm:
            approved = await callback.on_confirmation_needed(tc)
            if not approved: → append denied result, continue
        output = await tools.execute(tc.name, tc.input)
        callback.on_tool_result(tc.id, output)
        results.append(ToolResult(...))

    messages.append(tool_results_message)

    # 微压缩：每轮截断旧工具结果
    truncate_old_tool_results(messages)
```

**特点**：
- 循环不可变 — 所有功能通过 tools 和 callbacks 注入
- 直接使用 asyncio.Future 确认 — 与现有 Hub 确认流完美契合
- 消息列表原地修改 — 可直接检查调试
- 无外部框架依赖

---

### Phase 3：LangGraph ReAct 后端

#### 3.1 状态定义 — `app/agent/langgraph_state.py`（新建）

```python
from langgraph.graph import add_messages

class AgentState(TypedDict):
    messages: Annotated[list, add_messages]
    workspace_path: str
    mode: str                # "plan" | "auto"
    iteration_count: int
```

#### 3.2 图节点 — `app/agent/langgraph_nodes.py`（新建）

```python
async def agent_node(state, config) -> dict:
    """调用 LLM，返回 AI 消息"""
    provider = config["configurable"]["provider"]
    # ... 构建 ChatRequest，调用 provider.chat()
    # 返回 {"messages": [ai_message], "iteration_count": +1}

async def tools_node(state, config) -> dict:
    """执行工具调用，返回 ToolMessage 列表"""
    registry = config["configurable"]["tool_registry"]
    reporter = config["configurable"]["reporter"]
    confirmation_handler = config["configurable"]["confirmation_handler"]

    for tc in last_message.tool_calls:
        # 上报 tool_call 事件
        # 确认检查：调用 confirmation_handler(tc) 而非 interrupt()
        # 执行工具
        # 上报 tool_result 事件

def should_continue(state) -> str:
    """条件边：有工具调用 → "tools"，否则 → END"""
```

#### 3.3 图构建 — `app/agent/langgraph_graph.py`（新建）

```python
def build_react_graph() -> CompiledGraph:
    graph = StateGraph(AgentState)
    graph.add_node("agent", agent_node)
    graph.add_node("tools", tools_node)
    graph.set_entry_point("agent")
    graph.add_conditional_edges("agent", should_continue, {
        "tools": "tools",
        END: END,
    })
    graph.add_edge("tools", "agent")  # 循环回 agent
    return graph.compile()
```

**关于确认流的设计决策**：
- **不使用 LangGraph `interrupt()`** — 因为它需要检查点持久化 + 外部恢复循环，与现有的 asyncio.Future 机制冲突
- **使用 config 注入的 confirmation_handler** — 在 tools_node 内直接 `await confirmation_handler(tc)`，底层仍走 `ExecutionService._wait_for_confirmation()`
- 这使得两种后端的确认流逻辑完全一致

#### 3.4 消息转换 — `app/agent/langgraph_compat.py`（新建）

LangGraph 使用 `langchain_core.messages`（HumanMessage, AIMessage, ToolMessage），而我们内部用 `Message`。需要双向转换：

```python
def internal_to_langgraph(msg: Message) -> BaseMessage
def langgraph_to_internal(msg: BaseMessage) -> Message
def chat_response_to_ai_message(resp: ChatResponse) -> AIMessage
```

---

### Phase 4：ExecutionService 重构

#### 4.1 入口切换 — `app/services/execution_service.py`

```python
class ExecutionService:
    def __init__(self, *, repo, agent_mode: str):
        self.agent_mode = agent_mode  # "vanilla" | "langgraph"

    async def execute(self, context):
        # ... 不变：创建 reporter, worktree ...
        await self._execute_agent(execution_id, context, workspace_path, reporter)

    async def _execute_agent(self, execution_id, context, workspace_path, reporter):
        # 构建工具注册表
        # 注入 Hub Skills
        # 构建系统提示词
        # 构建回调（桥接到 HubReporter）
        # 根据 agent_mode 分发到 _run_vanilla 或 _run_langgraph
```

#### 4.2 Vanilla 执行路径

```python
async def _run_vanilla(self, ...):
    messages = [Message(role="user", content=user_msg)]
    if context.get("mode") == "plan":
        # 先生成计划，await 确认
        ...
    await agent_loop(...)
```

#### 4.3 LangGraph 执行路径

```python
async def _run_langgraph(self, ...):
    graph = build_react_graph()
    config = {"configurable": {
        "provider": provider,
        "model": ...,
        "tool_registry": registry,
        "reporter": reporter,
        "confirmation_handler": lambda tc: self._wait_for_confirmation(...),
    }}
    await graph.ainvoke(initial_state, config=config)
```

#### 4.4 删除的方法
- `_execute_graph()` — 被 `_run_vanilla` / `_run_langgraph` 替代
- `_execute_mock()` — 不再需要 mock 模式
- `_generate_plan()` / `_generate_patch()` — LLM 通过工具循环自主工作
- `_extract_unified_diff()` — 同上
- `_emit_patch_flow()` — patch 现在是工具调用结果

---

### Phase 5：配置与依赖更新

#### 5.1 `app/config.py`

`agent_mode` 取值从 `"mock" | "graph" | "deepagents"` 改为 `"vanilla" | "langgraph"`，默认值改为 `"vanilla"`。

#### 5.2 `pyproject.toml`

```toml
dependencies = [
  # 保留
  "fastapi>=0.115.11",
  "uvicorn>=0.34.0",
  "sse-starlette>=2.2.1",
  "pydantic>=2.10.6",
  "orjson>=3.10.15",
  "aiosqlite>=0.21.0",
  "openai>=1.66.3",
  "anthropic>=0.49.0",
  "jsonschema>=4.23.0",
  "unidiff>=0.7.5",
  "httpx>=0.28.1",
  # 保留（LangGraph 后端需要）
  "langgraph>=0.5.0",
  # 新增
  "langchain-core>=0.3.0",
  # 移除: "deepagents>=0.0.8"
]
```

#### 5.3 `app/main.py`

`ExecutionService` 构造时传入 `agent_mode=settings.agent_mode`（逻辑不变，仅值域变化）。

---

### Phase 6：清理

#### 删除文件
- `app/agent/graph_agent.py` — 被 `langgraph_graph.py` 替代
- `app/agent/mock_agent.py` — 不再需要
- `app/agent/prompts.py` — 被 `system_prompts.py` 替代

#### 新增文件汇总
| 文件 | 用途 | 两种模式 |
|------|------|---------|
| `app/agent/messages.py` | 统一消息类型 | 共用 |
| `app/agent/tool_registry.py` | 工具定义 + 派发 | 共用 |
| `app/agent/system_prompts.py` | 丰富系统提示词 | 共用 |
| `app/agent/context_manager.py` | 上下文压缩 | 共用 |
| `app/agent/loop.py` | Vanilla agentic loop | Vanilla |
| `app/agent/langgraph_state.py` | LangGraph 状态 | LangGraph |
| `app/agent/langgraph_nodes.py` | agent/tools 节点 | LangGraph |
| `app/agent/langgraph_graph.py` | ReAct 图构建 | LangGraph |
| `app/agent/langgraph_compat.py` | 消息格式转换 | LangGraph |

#### 重写文件
| 文件 | 改动范围 |
|------|---------|
| `app/agent/providers/base.py` | 全部重写 |
| `app/agent/providers/anthropic_provider.py` | 全部重写 |
| `app/agent/providers/openai_provider.py` | 全部重写 |
| `app/agent/provider_router.py` | 最小改动 |
| `app/services/execution_service.py` | 重大重构 |
| `app/services/tool_injector.py` | 扩展返回 ToolDef |
| `app/config.py` | agent_mode 值域变更 |
| `pyproject.toml` | 依赖调整 |

---

### 实施顺序

```
Phase 1: 共享基础 ─────────────────────────────
  1.1 messages.py（消息类型）
  1.2 providers/base.py → anthropic_provider.py → openai_provider.py
  1.3 tool_registry.py（工具注册表）+ tool_injector.py 扩展
  1.4 system_prompts.py（系统提示词）
  1.5 context_manager.py（上下文管理）

Phase 2: Vanilla Loop ────────────────────────
  2.1 loop.py（核心循环）

Phase 3: LangGraph ReAct ─────────────────────
  3.1 langgraph_state.py
  3.2 langgraph_nodes.py
  3.3 langgraph_graph.py
  3.4 langgraph_compat.py

Phase 4: 集成 ─────────────────────────────────
  4.1 execution_service.py 重构
  4.2 config.py + pyproject.toml 更新
  4.3 main.py 适配

Phase 5: 清理 ─────────────────────────────────
  5.1 删除旧文件
  5.2 更新测试

Phase 6: 验证 ─────────────────────────────────
  见下方验证计划
```

---

## 验证计划

### 单元测试
1. **Provider 测试**：mock httpx 验证 Anthropic/OpenAI 请求格式正确（消息转换、工具 schema 格式）
2. **ToolRegistry 测试**：注册、查找、执行、schema 生成
3. **agent_loop 测试**：mock provider 返回 tool_use → 验证工具执行 → 返回 end_turn → 验证退出
4. **LangGraph 图测试**：mock provider + 验证 agent ↔ tools 循环
5. **context_manager 测试**：token 估算、截断、压缩

### 集成测试
1. **Vanilla 端到端**：`GOYAIS_AGENT_MODE=vanilla`，发送执行请求 → 验证 HubReporter 收到 tool_call/tool_result/done 事件
2. **LangGraph 端到端**：`GOYAIS_AGENT_MODE=langgraph`，同上验证
3. **Plan 模式**：两种模式下验证 plan → confirmation_request → approval → execution 流
4. **确认拒绝**：验证工具确认拒绝 → tool_result(denied) → 循环继续
5. **Plan 拒绝**：验证 PlanRejectedError → done(cancelled)

### 命令行验证
```bash
cd runtime/python-agent

# 运行测试
uv run pytest tests/ -v

# Vanilla 模式启动
GOYAIS_AGENT_MODE=vanilla uv run uvicorn app.main:app --port 8040

# LangGraph 模式启动
GOYAIS_AGENT_MODE=langgraph uv run uvicorn app.main:app --port 8040

# 手动触发执行（需要 Hub 运行）
curl -X POST http://localhost:8040/internal/executions \
  -H "Content-Type: application/json" \
  -d '{"execution_id":"test-1","trace_id":"t-1","user_message":"List all files","mode":"auto","model_config_id":"..."}'
```
