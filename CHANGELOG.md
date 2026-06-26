# PacketMind 更新日志

## [unreleased] - 2026-06-10

### 修复问题
- **Agent 初始 trace 噪声**：移除 `Agent.Analyze()` 中手动追加和广播的合成 `start` trace/event，分析事件流现在直接从 runtime 真实事件开始
- **Agent 模型下拉渠道名错误**：`AgentAPI.ListModels()` 的 grouped provider 名称现在读取 provider 显示名，避免配置为 `id=zhipu,name=Ark` 时下拉仍显示 `zhipu`
- **Provider 默认回退硬编码**：移除模型配置中对 `zhipu` 的默认 provider 回退，active provider 缺失时改为选择第一个已配置 provider，无 provider 时保持为空
- **模型配置弹窗渠道行布局**：修复 Channel 列表中 `Active` 徽标换行到下一行的问题，并将 API Type 输入框改为可扩展的 select 控件（当前仅提供 `OpenAI-Compatible`）
- **模型配置弹窗视觉优化**：移除不再需要的 `Refresh Configured Models` 按钮，并将 Channel/Model 操作按钮改为更贴近 PacketMind 桌面 inspector 风格的低圆角灰阶按钮、紧凑列表和分隔线语言

### 新增功能
- **Agent 体验 Phase 1**：AgentPanel 新增折叠式推理过程摘要、输入区上下文 chips、模型选择器渠道+模型双标签显示、错误卡片 Retry/Copy 操作按钮；agentStore 新增 `lastUserQuery` 支持失败后快速重试
- **手动模型管理**：`ModelsStore` 新增 `UpsertModel`/`DeleteModel` 方法，`ConfigAPI` 暴露 `UpsertModel`/`DeleteModel` Wails 绑定；前端 `ModelConfigModal` 新增模型添加/编辑/删除 UI，支持手动填写 Model ID、Display Name、Context 长度、Output 长度，并可在同一弹窗保存当前 provider、Base URL、active model、max tokens 与 temperature
- **手动 LLM 渠道管理**：`ModelConfigModal` 新增 LLM Channel 新建、编辑、删除入口，直接复用 `ConfigAPI.UpsertAgentProvider` / `DeleteAgentProvider` 保存 provider ID、显示名、OpenAI-compatible API Type、API Key 与 Base URL

### 代码改进
- **前端去除自动拉模型**：删除 `gui/src/api/wails.ts` 中 `FetchModels` 导入、`DynamicModelFetchRequest`、`agentApi.fetchModels()`、`fetchDynamicModels`；`ModelConfigModal.vue` 删除 Connect 按钮、`handleInitializeModels`、auto-connect watch
- **AgentModel 类型精简**：`gui/src/types/index.ts` 中 `AgentModel` 从 `{ id, name, provider, description, max_tokens, supports_streaming }` 精简为 `{ id, name, max_tokens }`
- **ListModels 绑定重构**：`bindings/agent.go` 中 `ListModels()` 改为从 `ModelsConfig.Models` map 构造 `{ id, name, max_tokens }` 对象，直接使用 map key 作为模型 ID
- **agentStore 简化**：`switchModel` 不再依赖 `model.provider`，改为从 `activeProvider` 取；删除 `fallbackProviderName`/`buildFallbackGrouping`；`normalizeGroupedModels` 简化

### 代码改进
- **MCP 内置工具前缀清理**：`builtin_server.go` 内置工具适配器使用空前缀，工具 schema 保持原始名称（`get_request` 等），LLM 无需学习 `builtin_` 前缀即可直接调用
- **MCPManager 双阶段工具分发**：`mcp.go` `ExecuteTool` 新增无前缀工具名 fallback 路由，按原始名称匹配空前缀适配器（内置工具）
- **Executor 默认 Session ID 注入**：`executor.go` `Execute` 调用 MCP 前将 `defaultSessionID` 注入 tool arguments（`session_id` 不存在时），确保 handler 始终能获取会话上下文
- **MCP 结果结构化传输**：`builtin_server.go` handler wrapper 将 `ToolExecutionResult`（content + summary + request_ids）编码为 JSON 经 MCP 回传；executor `buildToolResult` 解析结构化结果，保留 Summary/RequestIDs 元数据
- **`ExecuteToolWithJSON` 简化**：`mcp.go` `MCPManager.ExecuteToolWithJSON` 返回原始文本而非 `{"ok":true,"content":"..."}` 双层包装；工具错误直接返回 Go error

### 修复问题
- **AgentPanel 崩溃**：`selectedModel.provider.toUpperCase()` 和 `selectedModel.description` 引用已删除字段导致渲染错误；改为显示 `selectedModel.name` + `selectedModel.max_tokens`
- **AgentPanel 模型搜索残留旧字段**：`filteredGroupedModels` 和 `filteredFallbackModels` 中残留 `model.provider`/`model.description` 引用，已清理
- **Wails 绑定过期**：`gui/wailsjs/go/` 中 `AgentAPI.js` 仍导出 `FetchModels`，`models.ts` 中 `ModelConfig` 类仍有旧字段；执行 `wails generate module` 重新生成
- **内置工具 `unknown tool` 错误**：`MCPManager.ExecuteTool` 无前缀名称路由缺失，7 个 agent 测试因工具名不匹配 `builtin_` 前缀而失败；添加空前缀适配器 fallback 后修复
- **`ExecuteToolWithJSON` 内容双层 JSON 编码**：handler JSON 输出被 `{"ok":true,"content":"..."}` 二次封装，消费端无法直接检查字段；改为返回原始文本
- **工具调用 `session_id` 缺失**：handler 通过 `args["session_id"]` 获取会话 ID，但 executor 未注入 `defaultSessionID`；缺失时注入修复

## [unreleased] - 2026-06-09

### 代码改进
- **移除 `llmcore/types.go`**：删除 `ModelInfo`/`ModelLister` 类型，移除 provider 自动模型发现能力；`provider/openai/client.go` 删除 `ListModels` 方法，`registry.go` 删除 `NewLister` 工厂函数，`bindings/agent.go` 删除 `FetchModels` Wails 绑定
- **ModelConfig 改为 map 格式**：`ModelConfig` 从 `[]ModelConfig` 切片改为 `map[string]ModelConfig`（JSON 中以模型 ID 为键）；删除 `ID`/`Provider`/`MaxTokens`/`SupportsStreaming`/`Description` 字段，新增 `limit.context`/`limit.output` 和 `modalities.input`/`modalities.output` 字段；同步重写 `models_store.go`、`normalize.go`、`config_test.go` 适配新格式，更新 `configs/models.json`
- **清理 `llmtypes/types.go`**：删除未使用的 `TypeInteger`/`TypeNumber`/`TypeBoolean`、`WithMaxTokens`/`WithTemperature`/`WithAllowedToolNames`、`LLMOptions.AllowedToolNames`、`CloneToolDefinitions`、`LLMClient.WithTools` 接口方法及所有实现
- **清理测试文件**：`bindings/agent_test.go` 删除对已移除 `recordSessionInteraction` 方法的引用，改为调用 `persistUserMessage` + `persistAssistantMessage`；`provider/openai/client_test.go` 删除 `TestClient_ListModels`；agent/agent_test.go 和 runtime_test.go 删除 stub 中的 `WithTools`

### 修复问题
- **`Go build ./...` 重复符号 `fallbackProviderName`**：`models_store.go` 和 `normalize.go` 各声明了一次 `fallbackProviderName`；移除 `models_store.go` 中的副本，保留 `normalize.go` 中的定义，build 恢复通过

## [unreleased] - 2026-06-02

### 新增功能
- **MCP stdio 客户端集成**：新建 `internal/agent/mcp_client.go`，用 `github.com/mark3labs/mcp-go v0.54.1` 实现 `MCPClient` 接口（`NewStdioMCPClient` 负责启动子进程并完成 MCP Initialize 握手）；`ListTools` 将 `mcp.ToolInputSchema` 序列化为 `map[string]interface{}`，`CallTool` 通过 type-switch 将 `TextContent`/`ImageContent` 转为统一 `MCPContentBlock`。
- **MCP 配置支持**：`internal/config/app_settings.go` 新增 `MCPServerConfig`（name/command/args/env/enabled）和 `MCPSettings`（servers 列表）类型，`AppSettings` 添加 `MCP MCPSettings` 字段；`configs/packetmind.json` 新增 `mcp` 配置节，含 Playwright MCP 示例条目（默认 `enabled: false`）。
- **应用启动自动装配 MCP**：`app.go` `startup()` 中提取 `initMCPManager()` 辅助函数，遍历 `AppSettings.MCP.Servers` 中已启用条目，依次创建 `StdioMCPClient`、注册到 `MCPManager`、调用 `RegisterAll()` 完成工具发现，失败时 best-effort 记录日志继续；成功后调用 `agent.SetMCPManager()` 将 MCP 工具注入 Agent 运行时。

### 修复问题
- **MCP Manager 未注入 Agent 分析实例**：`app.go` `startup()` 初始化的 `MCPManager` 只赋给了局部变量 `ag`，该变量随即被丢弃，`bindings/agent.go` 每次分析新建 `baseAgent` 时从未调用 `SetMCPManager`，导致 MCP 工具对 Agent 不可见。修复方式：新增 `agent.DefaultMCPManager` 全局单例（`internal/agent/global.go`），`startup()` 将 manager 写入该全局；`runAgentAnalysis` 在 `SetStore` 之后检查并调用 `baseAgent.SetMCPManager(agent.DefaultMCPManager)`，与 `storage.Default`/`proxy.Default` 的现有模式保持一致。
- **移除 provider 客户端硬编码 SOCKS5 代理**：`internal/agent/provider/openai/client.go` 的 `newClient` 之前无条件将所有 AI 请求路由到 `socks5://127.0.0.1:8889`，导致 `provider/openai` 包所有单元测试（`httptest.NewServer`）因请求经本地代理转发而获得 EOF 失败；移除硬编码 `Proxy` 字段后测试全部通过，生产流量仍可通过系统/环境代理正常路由。

## [unreleased] - 2026-06-01

### 代码改进
- **bash 工具切换为 go-cmd/cmd**：`internal/agent/tools/builtin/shell.go` 使用 `github.com/go-cmd/cmd v1.4.3` 替换原 `os/exec` 实现；核心改进是进程组级别的正确终止（`Stop()` 杀整个进程树，避免孤儿进程），以及内建的竞态安全 stdout/stderr 收集；`detectShell` 重构为同时返回 shell 路径和参数切片，超时/context 取消均通过 `select` + `c.Stop()` 处理。

## [unreleased] - 2026-05-28

- **删除 Mock provider 全部逻辑**：从 `internal/agent/agent.go` 删除 `mockAgentLLMClient` 类型；从 `registry.go` 删除 mock provider 注册块；重写 `internal/config/normalize.go` 移除 mock provider 的强制插入逻辑（空列表时不再补 mock 种子）；从 `models.go`、`models_store.go` 移除所有 `"mock"` 硬编码回退赋值和 `DeleteProvider` 中对 mock 的不可删除特判（所有 provider 现在均可删除）；从 `internal/api/bindings/agent.go` 删除 `apiType == "mock"` 的模型列表特殊分支；从 `configs/models.json` 删除 mock provider 条目；同步修复关联测试（`stubLLMClient` 替代 `mockAgentLLMClient`、更新 config 测试断言），并恢复 `loadChatHistoryMessages` 中缺失的 12 条尾部窗口截断逻辑（`maxAnalysisChatHistoryMessages` 常量已重新声明并生效）。

### 代码改进
- **优化 Agent 提示词**：重写 `internal/agent/prompts/agent_role.txt`，移除未注册的 `search_requests_by_host` 工具描述，精简 bash 权限说明，将语言跟随规则提升至 Response style 首条，统一权限拦截的处理说明；`initial_prompt_with_target.txt` 新增明确约束"不要对同一 request_id 重复调用 get_request"并注明 session_id 使用范围；`initial_prompt_no_target.txt` 新增"不要用猜测或伪造的 request_id 调用 get_request"约束，减少模型在 ReAct 循环中无效工具调用。

### 修复问题
- **请求详情面板点击后变灰**：修复点击 `RequestList` 中某条请求后右侧详情面板变为空灰屏的 bug。根因有两处：① `RequestList.vue` 的 `@select` 只调用 `store.selectRequest(id)`，仅设置 `selectedId` 但不保证请求已在 `requests` 数组中，导致 `selectedRequest` computed 返回 `null`；修复为改用 `store.selectRequestById`，会先通过 `requestApi.get(id)` 加载缺失请求再设选中态。② `RequestSection.vue` 上次清理 `ContextMenu` 时遗留了三处模板残余：`<ContextMenu>` 组件标签（引用了已删除的 `contextMenu.visible/position/fieldName` 等 state）、hex 视图的 `@contextmenu.stop.prevent="openContextMenu(...)"` 绑定、raw 视图同类绑定；这些引用导致组件渲染时抛出 `TypeError: Cannot read properties of undefined (reading 'visible')`，使整个 Contents 面板崩溃白屏。全部清除后面板正常渲染 Headers/Body。
- **AgentPanel `store is not defined`**：`AgentPanel.vue` 第 635 行 `isEstimatedTokenUsage` computed 里误用了裸 `store.tokenUsage`（应为 `agentStore.tokenUsage`），导致 AI 面板每次渲染都抛 `ReferenceError`；修正变量名后 console 错误消失。
- **AI 回复流式传输**：修复 Agent 回复一次性出现而非逐 token 流式展示的问题。根因是 `runtime/runner.go` 调用 `llmtypes.CollectStream()` 会把 provider 的所有 SSE chunk 全部读完再触发回调。现在改为在 `llmtypes` 新增 `CollectStreamWithTextDelta(reader, onDelta)`，在聚合的同时对每个纯文本 chunk（无 tool calls）立即触发 `onDelta` 回调；`runner.go` 通过 `collectStreamWithDelta` 在 `onDelta` 里发射 `text_delta` 类型的 `AgentEvent`，前端 `useWailsEvents.ts` 和 `agentStore` 已有对应处理逻辑，无需改动前端。Tool call 轮次的 arguments 仍静默聚合，不发 `text_delta`。`CollectStream` 保持向后兼容，内部改为调用 `CollectStreamWithTextDelta(reader, nil)`。
- **恢复并注册 `bash` 内置工具**：重写 `internal/agent/tools/builtin/shell.go`（`BashResult`、`ExecuteBash`、`newBashHandler`、`detectShell`），在 `catalog.go` 中补回 `ToolBash` 常量和 schema，在 `register.go` 中完成正式注册；Agent 现可通过 `bash` 工具在 `data/agent-workspace` 沙箱目录执行 shell 命令，输出上限 50 000 字节，超时默认 60 秒最长 300 秒。
- **删除 `Agent.SetExecutor()`**：从未被任何调用方使用，`SetStore` 内部已通过 `resetExecutor` 完成 executor 的初始化；无需单独暴露 executor 注入接口。
- **删除 `Agent.systemPromptOverride` 字段**：字段无 setter、从不被赋值，`Analyze()` 中的相关 if 分支永远不会执行；一并移除死代码。
- **删除 `SystemPromptBuilder.WithContext()` 和 `Reset()`**：两个方法在整个代码库内无调用者，属于未使用的 API 面；删除后 `prompt.go` 仅保留实际被使用的 `WithRole`、`WithInstructions`、`WithSection`、`Build` 四个方法。
- **精简字段右键 AI 分析入口**：删除 `gui/src/components/ContextMenu.vue`（专用于字段右键分析的组件）；清理 `RequestSection.vue` 和 `HeadersView.vue` 中所有 `@contextmenu` 绑定、`ContextMenu` import、`openContextMenu`/`closeContextMenu`/`handleAnalyze`/`ContextMenuState` 等相关代码；AI 分析渠道统一收紧至 Agent 聊天框，不再支持字段级右键快捷分析。
- **删除 `AgentRequest` Target 字段三元组**：从 `internal/agent/types.go`（`AgentRequest`）、`internal/api/bindings/agent.go`（`AnalyzeRequest`）、`gui/wailsjs/go/models.ts`（`AnalyzeRequest` 类）中一并移除 `TargetField`/`TargetLocation`/`TargetValue` 三字段；bindings 层删除 `buildAgentQuery()` 辅助函数，`runAgentAnalysis` 直接使用 `req.Query`；`internal/agent/agent.go` 的 `buildInitialPrompt` 和 `internal/agent/prompts/initial_prompt_with_target.txt` 同步清理 Target 字段注入与条件块。
- 删除 `internal/agent/tools/suggest.go`：移除 `fuzzyMatchTool` / `levenshteinDistance` 模糊匹配兜底；`Executor.Execute()` 在 unknown tool 分支现在直接返回 `error`，由 `SafeExecute` 的 panic/timeout recovery 包装成结构化错误传回 runner，不再给模型返回带 `suggestion` / `hint` 的软性 observation；同步删除已无调用者的 `toolNames()` 私有方法。
- 删除 `internal/agent/runtime/truncate.go`：`TruncateOutput` 只作用于前端 UI 的 `AgentEvent.Result`，对 LLM 的 transcript 内容没有影响；上下文大小管理的正确位置在工具构造阶段（`builtin/helpers.go` 的 `previewBody` 8000 字符上限 + `include_full_body=true` 按需拉全量机制）。`emitObservationFromMessage` 现在直接把完整 result 传给 UI，不再经过落盘截断。
- 删除 `internal/agent/errors.go`、`internal/agent/retry.go` 两个纯转发壳层；`AgentError`/构造函数/重试基础设施的实现就在 `llmcore/` 包，调用方直接 import，无需中间代理。
- 裁减 `internal/agent/types.go`：从 155 行全量 re-export 缩减为 45 行，只保留 `agent` facade 自有的三个类型（`AgentRequest`、`AgentTraceStep`、`AgentResult`）和两个必要的 runtime type alias（`AgentEvent`/`AgentEventHandler`）；`llmtypes`/`llmcore`/`tools` 常量/类型由各子包调用方直接 import。
- 将 `internal/agent/errors_test.go`（测试已删除转发壳）迁移为 `internal/agent/llmcore/errors_test.go`，测试真实实现。
- `internal/agent/registry.go`、`mcp.go`、`agent.go`、测试文件的 import 已同步更新，直接引用 `llmtypes`/`llmcore`/`agenttools` 具体标识符。

## [unreleased] - 2026-05-25

### 新增功能
- `internal/storage` 新增 `chat_message` memdb table 与 `storage.ChatMessage` 模型，Agent 自由问答/追问消息现按 capture session ID 持久化到 storage，支持 `SaveChatMessage()`、`ListChatMessages()` 与 `DeleteChatMessages()`。

### 代码改进
- 已将 `internal/agent` 工具体系从父包中提取为独立子包：新增 `internal/agent/tools/`（承载内置 tool schema/catalog、参数解析、schema 校验、tool executor、模糊匹配）与 `internal/agent/tools/builtin/`（承载 `get_request`、header/body/response/search_all_fields 检索、provenance tracing、diff、encoding、batch、bash 等内置工具实现）。
- `internal/agent/agent.go` 现改为持有 `executor *tools.Executor` 与 `store *storage.Storage`；`Analyze()`、`runtime.Runner` 装配与 tool schema 获取统一走 executor，不再依赖父包中的 giant switch dispatch。
- `internal/agent` 内置工具已从直接读取 `storage.Default` 改为通过 `Agent.SetStore(store)` 注入的显式 `*storage.Storage` 实例执行；`internal/api/bindings/agent.go` 在创建 provider-backed agent 后会显式调用 `SetStore(storage.Default)` 完成运行时装配。
- `internal/agent/agent.go` 现修正 request-targeted analysis：当 `RequestID` 存在但 `SessionID` 为空时，先按 request ID 直接读取目标请求并回填其 session，避免把 `RequestID` 误当成 session id 传给 `GetRequest()`。
- `internal/agent/agent.go` 的 storage-backed chat history 加载现增加固定尾部窗口，仅把最近 12 条 `user` / `assistant` 历史消息送入模型，避免无界历史继续膨胀 prompt，也不再把持久化的 `system` / `tool` 记录直接回放给模型。
- Agent 模型侧 builtin tool 集合现临时移除 `bash` 暴露；在恢复真实用户审批流前，不再允许模型直接拿到 shell tool schema/handler。
- `internal/agent/mcp.go` 现改为基于 schema slice 管理 MCP 暴露工具，`MCPManager` 通过 `Schemas()` / `ExecuteToolWithJSON()` 与新的 `tools.Executor` 对接；不再要求生产代码继续依赖旧 `ToolRegistry` 作为主执行路径。
- 删除旧的 `internal/agent/agent_dispatch.go`、`tool_registry.go`、`interfaces.go`、`schema_validate.go`、`tools.go`、`tools_batch.go`、`tools_diff.go`、`tools_encoding.go`、`tools_search.go`、`sandbox.go` 与 `tool_registry_compat.go`；`registry_test.go` 也一并删除，避免继续为已移除的 ToolRegistry 维护无生产价值的兼容层/测试。
- `internal/agent/agent.go` 已移除生产态 `SessionMemory` 依赖：`Analyze()` 现在只会通过注入的 `*storage.Storage` 从 `chat_message` 表按时间顺序恢复最近尾部窗口内的 `user` / `assistant` turn，并在末尾追加本次用户 prompt 后交给 runtime。
- `internal/api/bindings/agent.go` 已移除 per-session `SessionMemory` LRU/cache、预热与淘汰逻辑；`runAgentAnalysis()` 不再创建/挂接内存会话，`recordSessionInteraction()` 仅持久化到 storage `chat_message`，`GetSessionContext()` 也改为返回 storage-backed 统计。
- `internal/storage/export.go` 的 `ExportAll()` / `ImportAll()` 现按 session 级别 round-trip `chat_messages`，删除 capture session 时也会在 `DeleteSession()` 事务中级联清理 chat messages。

### 测试更新
- `internal/agent/agent_test.go`、`runtime_test.go`、`mcp_test.go` 已切换到新的 executor/store wiring：测试通过 `NewAgent(...)+SetStore(...)` 构造 agent，并直接验证 `executor.Execute(...)`、schema-based MCP registration 与 `runtime.WithExecuteTool(agent.executor.SafeExecute)` 行为。
- `internal/agent/agent_test.go` 新增回归：覆盖 `RequestID` 单独传入时仍能正确解析目标请求 session，以及 storage-backed chat history 只加载尾部 12 条消息进入分析输入。
- `internal/storage/storage_test.go` 新增 chat persistence 回归：覆盖 `SaveChatMessage()` / `ListChatMessages()` 顺序读取、`DeleteSession()` 级联清理 chat messages，以及 `ExportAll()` / `ImportAll()` 对 `chat_messages` 的 round-trip。
- `internal/api/bindings/agent_test.go` 已改为验证纯 storage 行为：`GetChatHistory()` / `GetSessionContext()` 均直接读取 `chat_message`，`ClearSessionMemory()` 只清理持久化历史，`recordSessionInteraction()` 不再维护热缓存。

### 文档更新
- 更新 `AGENTS.md`，记录 `internal/agent/tools/` 与 `internal/agent/tools/builtin/` 的新目录职责、`Agent` 的 `executor/store` 注入模式，以及 bindings/test 装配时需显式 `SetStore(...)` 的约定。
- 更新 `AGENTS.md`，补充 storage-only chat context 设计：生产代码已移除 `SessionMemory`/LRU 运行时，Agent 对话历史与上下文统计统一来自 `chat_message`，`ClearSessionMemory()` 仅作为兼容命名保留并执行持久化历史清理。
- 更新 `AGENTS.md`，修正上下文弹窗与 chat history 回放约定：`ContextModal` 在打开时拉取 storage-backed `SessionContextStats`，而运行时只回放最近尾部窗口内的 `user` / `assistant` 历史消息。

### 代码改进
- 已将 `internal/agent/runtime.go` 提取为独立 `internal/agent/runtime/` 子包，新增 `runner.go`、`events.go`、`usage.go`、`truncate.go`、`helpers.go`、`retry.go`；原 `ReactRuntime` 重命名为 `runtime.Runner`，通过 `ExecuteToolFunc` 注入 `a.safeExecuteTool`，消除 `agent ↔ runtime` 循环依赖。
- `internal/agent/types.go` 现通过 type alias 继续对外暴露 `AgentEvent`、`AgentEventHandler`、`RuntimeResult`、`SafeToolResult`、`ToolExecutionResult`、`TruncateResult`，保持 `internal/api/bindings/agent.go` 等上层调用方无需改动。
- `internal/agent/agent.go`、`agent_dispatch.go` 及各工具实现已统一切换到 `runtime.Runner` / `runtime.ToolExecutionResult` / `runtime.SafeToolResult`；运行时 token usage 聚合与输出截断逻辑同步迁移到 runtime 子包。

### 测试更新
- `internal/agent/runtime_test.go` 已切换为直接构造 `runtime.NewRunner(...)` 并通过 `runtime.WithExecuteTool(agent.safeExecuteTool)` 覆盖既有最终答案提取回归，保持公开 Agent API 行为不变。

### 代码改进
- `internal/agent/llmtypes.LLMClient` 已切换为 stream-only 合同，删除 `Generate()`；新增 `llmtypes.CollectStream()` 负责将 `LLMStreamReader` chunk 聚合为完整 `LLMMessage`，并合并 `Content`、`ReasoningContent`、按 index 累积的 `ToolCalls` 与最后一个非空 `ResponseMeta`。
- `internal/agent/runtime.go` 的 `ReactRuntime.Run()` 现统一通过 `LLMClient.Stream()` + `llmtypes.CollectStream()` 获取完整 assistant turn，不再绕过流式链路走独立同步 completion 传输。
- `internal/agent/provider` 已按协议族拆分为 `internal/agent/provider/openai/`：当前由 `client.go`、`chat.go`、`stream.go` 构成 OpenAI protocol-family 实现，并将 `OpenAICompatibleClient`/`ProviderConfig`/`NewOpenAICompatibleClient` 重命名收敛为 `Client`/`Config`/`NewClient`；品牌 preset 薄封装已删除。
- `internal/agent/registry.go` 已切换到导入 `internal/agent/provider/openai`，默认 `openai-compatible` provider 通过新的 protocol-family client 构造。
- `internal/agent/agent.go`、`runtime_test.go`、`agent_test.go` 中的 mock/test LLM client 已删除 `Generate()` 并改为直接返回流 reader，保持全链路 stream-only。

### 测试更新
- provider 单元测试已从 `internal/agent/provider/openai_compatible_test.go` 迁移到 `internal/agent/provider/openai/client_test.go`，统一验证 OpenAI/Zhipu protocol-family 配置、stream 错误分类、`CollectStream()` 聚合行为与 `ListModels()`。
- 新增 `internal/agent/llmtypes/types_test.go`，覆盖 `CollectStream()` 的 chunk 聚合、tool call 按 index 拼接与错误透传。

### 文档更新
- 更新 `AGENTS.md`，记录 `internal/agent/provider/openai/` 的协议族目录结构与 `LLMClient` stream-only/`CollectStream()` 约定。

## [unreleased] - 2026-05-22

### 修复问题
- `internal/storage/export.go` 的 `ImportAll()` 现在会先验证顶层 `ActiveSession` 是否落在导入 payload 的 session ID 集合内；若顶层值缺失或失效，则按 `IsActive` 标记回退，再回退到最新 session，避免把 stale active 直接写入导入结果。
- `internal/storage/request.go` 的 `GetRequest()` 不再在空 `sessionID` 下悄悄回退到 active session；显式读/溯源路径已改为传入明确 session ID，`ExportHAR()` 改用内部按 ID 查找的私有 helper。
- `internal/storage/helpers.go` 的 `cloneRequest()` 现深拷贝 TLS 相关切片字段，避免证书链与扩展在克隆后发生别名共享。
- `internal/agent` 的 `get_request` / `diff_requests` 工具与请求快照/搜索结果元数据现统一使用当前分析会话或请求自身 `SessionID`，不再错误回退到 active recording session，修复“查看会话 ≠ 当前激活录制会话”时的上下文串线。

### 代码改进
- `internal/storage/session.go` 与 `internal/storage/request.go` 已切换为基于 `go-memdb` 的事务存储：`Storage` 现在仅保留 `db *memdb.MemDB` 作为状态，`session` / `request` 两张 memdb table 为唯一真相来源，公开 `*Storage` API 签名保持不变。
- Session 聚合字段 `Session.Requests` 与 `Session.HostGroups` 已从 runtime `Session` 中移除；`GetSession()` / `ListSessions()` 现在返回 metadata-only session，聚合视图统一通过新的 `SessionView` 与导入导出兼容 envelope 按 request table 动态派生。
- `SaveRequest` 现按 memdb `id` 唯一索引执行 upsert，并继续负责自动补全 active session、request id、创建/更新时间戳，满足代理双阶段落库行为。
- `internal/storage/search.go`、`har.go`、`export.go`、`provenance_query.go` 已移除最后一批 `d.mu` / `d.sessions` 兼容读路径，统一改为基于 `d.db.Txn(false)`、`ListSessions()`、`GetRequest()` 与 memdb-native request 查询执行读取；`ImportAll()` 现通过重建 memdb 并重新插入 session/request record 完成全量导入。
- `internal/storage/session.go` / `request.go` 已删除过渡期兼容层：`noopRWMutex`、`Storage.mu`、`Storage.sessions`、`rebuildSessionCacheTxn()`、`collectRequestCandidatesLocked()` 与 `findRequestInActiveSessionLocked()` 均已移除，storage 读写路径不再依赖 cache rebuild shim。
- `internal/storage` 现进一步删除 `activeMu` / `activeSession` 缓存，`GetActiveSession()`、`SaveRequest()`、`ListRequests()`、`ExportAll()` 等路径统一直接使用 memdb `is_active` 索引解析当前激活会话；写路径在同一事务内完成 active-session 解析，避免额外缓存与双重真相来源。
- `ImportAll()` 在导入数据缺少显式 `ActiveSession` 且各 session 也未标记 `IsActive` 时，现会与正常运行时一致回退激活最新 session，避免非空存储进入“无 active session”状态。
- `internal/storage` 本轮继续完成解耦：runtime `Session` 模型不再承载 `Requests/HostGroups` 派生字段，新增仅用于读取聚合视图与导入导出的内部 envelope；`sessionRecord` 也同步收缩为仅包含持久化所需的 session 元数据。
- `search.go` / `provenance_query.go` 已移除无意义的外层读事务与多余 wrapper，统一直接复用 txn-aware request 查询路径，减少样板逻辑并保留现有行为。
- `internal/storage/export.go` 的 `ExportAll()` 现从单个 read txn 构建导出快照，避免 `sessions[].is_active` 与顶层 `active_session` 在并发写入下来自不同快照。
- `internal/storage/helpers.go` / `session.go` 本轮继续做零行为变化收尾：删除未使用的 `cloneSession()`、内联单次使用的 `isSessionRecordActive()`，并去掉 `buildSessionTxn()` 未使用的 txn 参数，减少 storage 样板而不改变 memdb 读写语义。
- `internal/agent/agent_dispatch.go` 现将已解析的 `session_id` 一致传递给 `get_request` 与 `diff_requests`，避免在局部清理后重新绕回原始 `defaultSessionID`，保持分析会话优先级不变。

### 文档更新
- 更新 `AGENTS.md`，记录 `internal/storage` 的 session/request 主存储已迁移到 `go-memdb` 事务模型，`Storage` 仅保留 `db *memdb.MemDB`，active session 统一通过 `is_active` 索引直接查询，Session 聚合字段按需派生而非作为权威写入源。
- 更新 `AGENTS.md`，补充 `internal/storage/search.go` / `har.go` / `export.go` / `provenance_query.go` 也已全部切换为 memdb 原生读事务，后续不要再引入 `Locked` 兼容 helper 或 `d.sessions` 镜像缓存。

### 测试更新
- `internal/storage/storage_test.go` 新增/更新回归：`ImportAll()` 在顶层 `ActiveSession` 失效时会优先回退到嵌入的 `IsActive` session，再回退到最新 session；`GetRequest()` 现要求显式 session ID；`cloneRequest()` 的 TLS 深拷贝不再与原对象别名。
- `internal/storage/provenance_test.go` 新增回归，确认 provenance 查询在显式 session ID 下运行，不再依赖空 session 的 active fallback。
- `internal/agent/agent_test.go` 新增回归，覆盖 `get_request` / `diff_requests` / 搜索结果与初始请求快照在“分析会话 != active session”时仍使用正确的 capture session。
- `internal/agent/agent_test.go` 补充回归，确认 `executeTool` 在 `get_request` / `diff_requests` 上显式传入 `session_id` 时会优先使用解析后的会话参数，而不是落回 `defaultSessionID`。
- `CreateSession` / `SaveRequest` 现在在 memdb 克隆存储后将自动生成字段（ID/SessionID/IsActive/CreatedAt/UpdatedAt）回写到调用方原始对象，确保代理两阶段录制与测试的 `session.ID` / `req.ID` 可用。
- `ComposeRequest` / `ReplayRequest` 测试断言已修正为 upsert 语义（每 ID 恰好 1 条记录），与 memdb `Insert` 替换行为一致。
- `internal/storage/storage_test.go` 新增 active-session 默认解析覆盖：`SaveRequest()` / `ListRequests()` 在空 `SessionID` 下仍默认命中当前激活会话，`ImportAll()` 继续保留导入后的 active session 选择。
- 新增 `ImportAll()` 缺少 active 选择时的回归测试，覆盖“非空导入后自动激活最新 session”的稳定语义。
- 新增 runtime `GetSession()` 与派生 `GetSessionView()` 的分工测试，并验证导出 payload 仍保留历史 `sessions[].requests` 兼容形状。

### 新增功能
- `internal/storage/schema.go` 新增 go-memdb schema 定义、`sessionRecord` / `requestRecord` 类型与双向转换函数，支持 session 表 3 索引（id/is_active/created_at）与 request 表 6 索引（id/session_id/session_host/session_created_at/session_status_code/host）。

## [unreleased] - 2026-04-24

### 修复问题
- 修复前端 AI 面板在切换抓包会话时的对话串线问题：`gui/src/stores/agentStore.ts` 现将 streaming、analysis id、增量内容、agent events、工具/深度计数、related requests、provenance、pending queue 等完整瞬时状态一并纳入 `SessionSnapshot`，并允许在分析流式进行中切换可见会话。
- 修复 `gui/src/composables/useWailsEvents.ts` 对 `agent:analysis` 广播一律写入当前可见会话的错误；现已基于后端下发的 `session_id` 将 error / agent_event / agent_result / done / cancelled / legacy content 静默路由到对应会话快照，避免跨会话污染可见对话。

### 代码改进
- `agentStore` 新增 `routeEventToSession(sessionId, handler)` 以支持离屏会话的流式分析持续落盘到前端 session snapshot；当前可见会话仅在事件 `session_id` 匹配时才直接更新 UI 状态。

### 修复问题
- 修复 z.ai / GLM 在 Eino ADK `ExitTool` 路径下可能被误判为 `empty_final_answer` 的问题：当模型把最终答案写入 assistant 侧 `exit` tool call 的 `final_result` 参数、而 exit 工具 observation 本身为空时，`internal/agent/eino_events.go` 现会优先回收该参数作为真实最终答案，避免桌面 UI 错误显示 `Model did not return a final answer` 并落入保守 fallback。

### 测试更新
- `internal/agent/eino_events_test.go` 新增回归测试，覆盖 assistant 侧 `exit` tool call 仅通过 `final_result` 携带最终答案时，runtime 仍能正确产出 `FinalAnswer` 且不会触发 `empty_final_answer`。

### 文档更新
- 更新 `AGENTS.md`，补充 `internal/agent/eino_events.go` 在 `ExitTool` 路径下除 `Content / MultiContent / ReasoningContent` 外，还必须识别 assistant 侧 `exit` tool call 参数中的 `final_result`，以兼容 GLM / OpenAI-compatible 模型的终止答案形状。

## [unreleased] - 2026-04-17

### 代码改进
- `internal/api/bindings/session.go`、`request.go`、`agent.go`、`config.go`、`proxy.go`、`updater.go` 已移除对 `storage/proxy/modelsStore/settingsStore/context` 的 struct 字段注入，统一改为直接读取 `storage.Default`、`proxy.Default`、`config.DefaultModelsStore`、`config.DefaultSettingsStore` 与 `appctx.Ctx`；对应 runtime-only setter/helper 已删除，`app.go` 现负责在启动阶段一次性初始化这些全局单例。
- `internal/agent/Agent` 已移除内嵌 `*storage.Storage` 字段；`NewAgent` / `NewAgentFromProvider`、provider registry 与分析/工具检索路径统一改为直接使用全局 `storage.Default`，减少 Agent/registry/bindings/app 装配时的显式 store 透传。
- `internal/proxy` 现移除 `Proxy.storage` 字段并统一改为读取全局 `storage.Default`；`proxy.New()` 不再接收 `*storage.Storage` 参数，`app.go` 在初始化存储后负责设置 `storage.Default`。
- `RequestAPI.ReplayRequest` / `ComposeRequest` 不再在 bindings 层创建独立 `http.Client` 直连目标；现改为复用当前 `Proxy` 实例的 shared upstream client，确保请求重放与改写重放共享现有 External Proxy / bypass / transport 行为。
- HTTPS MITM 请求现会捕获原始客户端 `ClientHello` 首个 TLS record 到请求记录中；正常 MITM 上游转发与 `ReplayRequest` 在存在该数据时都会通过 `uTLS` 对上游连接做 best-effort `ClientHelloSpec` 复用，并在失败或缺失数据时安全回退到现有 shared upstream client。
- 拆分 `internal/config/config.go`（1308 行）为 5 个职责单一文件：`models.go`（类型定义 + ModelsConfig 方法）、`app_settings.go`（桌面/代理设置类型 + 生命周期函数）、`models_store.go`（ModelsStore + LoadModels + 持久化辅助）、`app_settings_store.go`（AppSettingsStore）、`normalize.go`（normalizeProviders / normalizeModels / mergeRuntimeModelsByProvider）。
- 删除所有老版本兼容代码：`legacyModelsConfig` 结构体、`migrateLegacyProviders()` 迁移函数、`legacyProviderAPIType()` 兼容函数。
- 简化 `LoadModels()`：直接反序列化为 `ModelsConfig`，移除从 `provider_api_keys` / `provider_base_urls` 旧格式的迁移路径。
- 简化 `normalizeProviders()`：将 `legacyProviderAPIType()` 的内联默认值逻辑直接嵌入，不再依赖独立兼容函数。

## [unreleased] - 2026-04-15

### 新增功能
- Session 列表每项新增绿色 `Active` 徽标，直观标识当前激活会话；右键菜单新增 `Set Active` 选项，支持直接切换激活会话（已激活会话该选项置灰不可点击）。
- AI Agent 消息历史现与抓包会话（capture session）关联：切换会话时自动保存当前对话快照并恢复目标会话的历史消息，Agent 记忆 session ID 直接使用抓包会话 ID，不再生成独立的合成 session ID。
- 新增 Eino ChatModel 适配层：`internal/agent/eino_model.go` 现可将现有 `llmcore.LLMClient`（如 `OpenAIClient`、`ZhipuClient`）包装为 Eino `model.ToolCallingChatModel`，支持 `Generate` / `Stream` / `WithTools`。
- 新增 `internal/agent/eino_agent.go` 与 `internal/agent/eino_events.go`：基于 Eino ReAct Agent 封装新的运行时与事件桥接层，可将 Eino 中间消息转换为 PacketMind 既有 `thought` / `action` / `observation` / `final` 事件流。

### 代码改进
- 模型配置收尾对齐：`ConfigAPI.GetAgentProviderKey()` 现在返回 `api_key`、`base_url` 与 `api_type`，前端 `ProviderInfo` 类型同步补齐 `api_type`，避免模型配置弹窗读取 provider 配置时出现前后端字段漂移。
- `gui/src/components/ModelConfigModal.vue` 收紧为当前仅暴露 `OpenAI Compatible` API 类型，`Connect` 流程改为同时要求 API Key 与 Base URL，并清理残留的 `Discover Models` / 可选 Base URL 文案，和新的运行时发现模型行为保持一致。
- `gui/src/components/ModelConfigModal.vue` 删除 provider 时不再使用浏览器原生 `confirm()`，改为应用内 `a-modal` 确认窗口，避免在 Wails 桌面环境中出现 `wails.localhost 显示` 的原生弹框体验。
- provider 列表删除能力改为完全由后端返回的 `deletable` 字段驱动；前端移除了对 `mock/openai/zhipu` 的硬编码删除限制，只按后端元数据决定是否显示删除按钮。
- 修复 `gui/src/components/ModelConfigModal.vue` 中 provider 删除成功后确认弹窗未关闭的问题：删除成功路径现在直接复位确认弹窗状态，避免被 `deleting` 期间的关闭保护误拦截。
- `gui/src/components/SessionList.vue` 的左键点击现仅切换查看/选择会话，不再隐式激活会话；会话激活动作保留在右键菜单的 `Activate Session` 中，避免浏览历史会话时误切换抓包写入目标。
- 新增前端独立的“当前查看会话”状态：`sessionStore` 现维护 `selectedSessionId/selectedSession`，Session 列表高亮与左键切换基于该状态，而 `Activated` 徽标继续仅反映后端 `is_active`，修复左键切换后列表无视觉反馈的问题。
- `gui/src/components/SessionList.vue` 现将“当前查看(Viewing)”与“当前激活(Activated)”做成两套明显不同的视觉语言：查看态为蓝色选中高亮与 `Viewing` 标记，激活态为绿色徽标/录制点；两者可同时出现，不再混淆。
- 修复 Eino ADK runtime 下 Agent 经常以 `empty_final_answer` 提前结束的问题：`internal/agent/eino_agent.go` 现启用 `adk.ExitTool`，`internal/agent/eino_events.go` 对 `event.Action.Exit` 产出的终止消息提升为真实 final answer，而不是错误走保守 fallback。
- 清理 `internal/agent/prompts/fallback_answer.txt` 中过期的 “increase tool budget” 文案，改为与当前 runtime 一致的提示。
- 恢复 `internal/agent/provider/openai_compatible.go` 的薄兼容构造函数 `NewOpenAIClient` / `NewZhipuClient`，让统一后的 OpenAI-compatible provider 继续兼容既有测试与调用习惯，而不重新复制 provider 实现。
- 进一步修复 `empty_final_answer`：`internal/agent/eino_events.go` 现在对普通终止 assistant 消息统一使用 `Content -> MultiContent 文本 -> ReasoningContent` 提取最终答案，并在 `ExitTool` 的 `final_result` 为空时回退到前一条 assistant 文本，避免模型把答案写在 assistant 内容里但 exit 工具返回为空时丢失最终答案。
- 修复 `internal/agent/eino_events.go` 中 `lastAssistantTerminalText` 被后续空 assistant/tool-call 回合覆盖的问题：现在仅在 assistant 消息确实带有非空终止文本时才更新缓存，避免多轮工具调用后 `ExitTool` 结果为空时把早先真实答案覆盖丢失。
- `internal/agent/eino_agent.go` 已将内部执行路径从旧 Eino ReAct/message-future 封装切换为单一 Eino ADK `ChatModelAgent + Runner` runtime；分析仍保持既有 `AgentRequest` / `AgentEvent` / `AgentResult` 合同、session memory 装配、系统提示词与取消语义。
- `internal/agent/eino_events.go` 新增 ADK `AgentEvent` 到 PacketMind `thought` / `action` / `observation` / `final` / `provider_retry` 事件桥接，并继续复用既有 observation JSON 元数据提取逻辑，保证 Wails `agent:analysis` 事件 envelope 无需改动。
- `internal/agent/eino_tools.go` 现为 ADK runtime 统一生成工具列表，直接复用现有 tool registry/schema、`safeExecuteTool()`、MCP 路由与原有 tool result JSON 结构；超时/panic 等 recoverable tool 错误会编码为兼容旧桥接逻辑的 JSON payload，而非抛出中断运行时的错误。
- `internal/agent/provider` 已将原 `openai.go` / `zhipu.go` 合并为单一 `openai_compatible.go`：通过 `ProviderFlavor` / `ProviderConfig` 统一控制默认 base URL、是否自动补 `/v1/`、默认模型、Zhipu thinking 注入、stream `reasoning_content` 提取、provider 错误标签与 `ListModels()` 返回的 provider 标识；`internal/agent/registry.go` 已改为按 flavor 构造 OpenAI 与 Zhipu 实例。
- `internal/agent/agent.go` 现作为唯一分析入口直接驱动单一 ADK runtime；旧的 `AgentOrchestrator` / `SpecialistAgent` / `collaboration` / `specialist_config` 实现及其测试文件已删除，后端不再维护多专家路由/接力执行链。
- `internal/api/bindings/agent.go` 的 Wails 分析入口已改为直接复用单一 `Agent.Analyze()` 运行时，不再在绑定层构造 `AgentOrchestrator` / `AnalyzeWithHandoff`；`Analyze`/`CancelAnalysis`/`GetChatHistory`/`GetSessionContext`/`ClearSessionMemory`/`ListModels`/`ListProviders`/`FetchModels` 的绑定签名保持不变，`agent:analysis` 事件顶层 envelope 与 `final_answer` / `token_usage` / `tokens_used` / `error` / `cancelled` 语义保持兼容。
- 绑定层现在在单 Agent 分析完成后直接将 `user`/`assistant` 往返写入既有 `SessionMemory`，继续保持会话缓存、聊天历史恢复与上下文统计行为；在无多专家协作结果时不再主动填充 `routing_*` / `handoff_*` / `specialist_*` 结果元数据。
- `internal/agent` 本轮进一步删除了单 ADK 迁移后的残留死代码：未使用的 `allEinoTools/buildEinoTools/buildEinoMCPTools` 路径、旧字段提取/格式化 helper、未使用 retry wrapper、无调用 provider helper、specialist-era error constructor，以及若干仅测试引用的旧兼容函数，降低了阅读成本并收缩了文件职责。
- Agent Eino Part 1 基础重构：删除 `internal/agent/llmcore/sse.go`、`internal/agent/sse.go`、`internal/agent/eino_convert.go`、`internal/agent/provider_compat.go`，并将 `llmcore/types.go` 收缩为仅保留 `ModelInfo` / `ModelLister`。
- `internal/agent/types.go`、`tool_registry.go`、`tools.go`、`mcp.go`、`agent.go`、`eino_agent.go`、`eino_model.go`、`eino_tools.go` 现改为直接以 Eino `schema.Message` / `schema.ToolInfo` / `schema.TokenUsage` / `model.ToolCallingChatModel` 为核心类型，不再保留自定义 LLM 协议别名层。
- `internal/agent/provider/openai.go`、`provider/zhipu.go`、`provider/adapter.go`、`provider/chat.go` 已改为 provider 直接实现 Eino `model.ToolCallingChatModel`，`Complete()`/流式返回也统一切换到 `schema.TokenUsage` 与 `schema.StreamReader[*schema.Message]`。
- 后端 `AgentAPI` 新增 `GetChatHistory(sessionID)` Wails 绑定，返回指定抓包会话的 Agent 对话历史（`ChatMessageDTO` 列表），前端切换会话时可从后端持久化存储恢复对话。
- 后端 `AgentAPI` 新增 `ClearSessionMemory(sessionID)` 绑定，用于统一清除内存 LRU 缓存，并复用于会话删除时的后端清理路径。
- `SessionAPI.DeleteSession` 现通过内部 `RegisterSessionDeleteHook(...)` helper 在 `app.go` 中注册删除清理回调，使删除抓包会话时同步清理对应的 Agent 内存缓存。
- `agentStore.ts` 新增 `SessionSnapshot` 内存缓存与 `bindSession()`/`switchSession()` 方法；切换会话时优先从前端缓存恢复，缓存未命中时回退到后端 `GetChatHistory` API 加载持久化记录。
- 移除 `ensureChatSessionId()` 中的合成 ID 生成逻辑，改为直接返回已绑定的抓包会话 ID。
- `App.vue` 的 `initializeActiveSession` 与 `handleSessionChange` 分别调用 `agentStore.bindSession()` / `agentStore.switchSession()` 以同步 Agent 上下文。
- 后端 `AgentAPI` 现通过 `NewAgentAPI(store, modelsStore)` 构造直接注入模型存储，并将 `SetContext` / `SetModelsStore` / `SetSessionDir` 从 Wails 导出面中移除，避免自动生成无用前端绑定；运行时上下文改由 `ConfigureAgentRuntime(...)` 在 `app.go` 内部注入。
- `GetChatHistory(sessionID)` 与 `GetSessionContext(sessionID)` 改为只在已有内存缓存存在时读取会话记忆，不再为纯读取请求创建空的 `SessionMemory` LRU 项；删除会话时直接复用 `ClearSessionMemory(sessionID)` 做统一清理。
- `SessionAPI` 的删除回调注册改为内部 helper `RegisterSessionDeleteHook(...)`，避免把仅供应用装配使用的 hook 方法暴露给 Wails 绑定。
- `Storage.DeleteSession(id)` 现在对缺失会话返回 `session not found`，与 `SessionAPI.DeleteSession` 的 `40001` 语义对齐。
- `internal/agent/eino_tools.go` 现仅保留单 ADK runtime 所需的 `buildADKTools()` 与共享 tool adapter，旧的非 ADK 聚合路径已移除。
- `go.mod` / `go.sum` 新增 `github.com/cloudwego/eino v0.8.9` 及其依赖，用于编译新的 Eino 适配层。
- `Agent.Analyze()` 已统一直接走单一 ADK runtime；旧 `ConversationRuntime` fallback 与相关兼容路径已删除。
- `EinoRuntime` 仅保留 `RunWithSchemaMessages()` 入口（接受原生 `[]*schema.Message`）；`Agent.Analyze()` 已切换到直接使用该入口，避免多余的消息转换与并行运行时心智负担。
- `SessionMemory` 新增 `ToSchemaMessages()` 方法，直接输出 `[]*schema.Message` 供 Eino 路径装配输入。

### 测试更新
- 修复 `internal/proxy/proxy_test.go`、`transport_test.go`、`internal/agent/agent_test.go`、`internal/api/bindings/agent_test.go`、`request_test.go` 对全局单例重构前旧构造签名与字段的测试引用；相关测试现统一在调用依赖全局的 Agent/Proxy/bindings 之前显式设置 `storage.Default`、`proxy.Default`、`config.DefaultModelsStore`，并改用无参 `New()` / `NewAgent()` / `New*API()` 构造，删除对已移除 `ConfigureAgentRuntime` 与 `Agent.store` / `Proxy.storage` 的依赖。
- 新增 `internal/api/bindings/agent_test.go`，覆盖 `GetChatHistory` 只读恢复不污染 LRU 缓存、`ConfigureAgentRuntime` 内部注入以及 `RegisterSessionDeleteHook` 删除清理 wiring；`internal/storage/storage_test.go` 同步覆盖缺失会话删除报错语义。
- 新增 `internal/agent/agent_test.go` 对单 ADK runtime `Agent.Analyze()` 路径的回归测试，验证 mock provider 下 `start/final` 事件与最终答案可通过新运行时正常产出。
- 修复 `internal/agent/provider/openai_test.go`、`provider/zhipu_test.go`、`memory_test.go`、`mcp_test.go` 对旧 llmcore Chat/Stream/Tool API 的引用，统一切换到 Eino `schema.Message`、`schema.TokenUsage`、`schema.StreamReader` 与 `schema.ToolInfo` 断言。
- `internal/agent/provider/openai_compatible_test.go` 现统一覆盖 OpenAI/Zhipu 两种 flavor 的构造、Generate、Stream、网络错误与 `ListModels()` 回归，原 `openai_test.go` / `zhipu_test.go` 已删除。

### 文档更新
- 更新 `AGENTS.md`：补充测试约定——凡单元测试直接构造依赖 package-level globals 的 `Agent` / `Proxy` / Wails bindings，必须先设置对应 `storage.Default`、`proxy.Default`、`config.DefaultModelsStore`、`appctx.Ctx`，并改用新的无参构造函数。
- 更新 `AGENTS.md`：记录 `internal/agent` 已移除 `Agent.store` 字段，运行期统一依赖 `internal/storage/global.go` 中由主进程初始化的 `storage.Default` 全局存储。
- 更新 `AGENTS.md`：补充当前模型配置弹窗仅暴露 `openai-compatible` API 类型，`GetAgentProviderKey()` 需返回 `api_key/base_url/api_type`，且 `Connect` 前必须填写 Base URL。
- 更新 `AGENTS.md`：前端桌面弹窗确认交互应优先使用应用内模态窗口，不要回退到浏览器原生 `confirm()`。
- 更新 `AGENTS.md`：provider 列表中诸如是否可删除之类的展示/交互能力必须由后端返回元数据驱动，前端不要硬编码特殊 provider 规则。
- 更新 `AGENTS.md`：对话框存在 loading 锁时，成功提交路径若需要立即关闭弹窗，不要复用会被 loading guard 拦截的 cancel helper，应直接复位弹窗状态。
- 更新 `AGENTS.md`：Session 列表左键仅用于切换查看/选中会话，激活会话必须通过右键菜单 `Activate Session` 显式触发。
- 更新 `AGENTS.md`：当前查看中的 session 与当前激活的 session 为两套状态；前端列表选中高亮不得继续复用后端 `is_active`。
- 更新 `AGENTS.md`：Session 列表需在视觉上明确区分“当前查看”和“当前激活用于录制”的 session，不能只靠单一高亮或单一 badge 表达两种状态。
- 更新 `AGENTS.md`：Eino ADK runtime 结束分析时，若 `event.Action.Exit` 携带终止消息，必须将其提升为最终答案；不能继续误判为 `empty_final_answer`。
- 更新 `AGENTS.md`：统一后的 OpenAI-compatible provider 如需继续兼容历史测试/调用方，应优先保留薄构造函数兼容层（如 `NewOpenAIClient` / `NewZhipuClient`），不要重新复制 provider 客户端实现。
- 更新 `AGENTS.md`：终止态 final answer 提取不得只依赖 `msg.Content`；普通 assistant 终止消息与 `ExitTool` 路径都应统一支持 `Content / MultiContent / ReasoningContent`，且当 exit 工具结果为空时要回退到前一条 assistant 文本。
- 更新 `AGENTS.md`：`lastAssistantTerminalText` 之类的终止文本缓存只能在读取到新的非空 assistant 终止文本时更新，不能被后续仅含 tool calls 的空消息覆盖。
- 更新 `AGENTS.md`：记录 `internal/agent/eino_agent.go` 已切换到单一 Eino ADK `ChatModelAgent + Runner` runtime，旧的 specialist/orchestrator/collaboration 栈已删除，`eino_events.go` / `eino_tools.go` 分别负责 ADK 事件桥接与工具执行桥接。
- 更新 `AGENTS.md`：记录 `internal/api/bindings/agent.go` 已切换为直接调用单一 `Agent.Analyze()` 后端路径，绑定层仅负责复用 `SessionMemory` 与透传既有 Wails event envelope；frontend contract 保持兼容，但在单 Agent 路径下 `specialist/handoff/routing` 元数据允许为空/省略。
- 更新 `AGENTS.md`：记录 Agent Eino Part 1 已移除自定义 llmcore 消息/工具/usage 协议层，provider 现在直接实现 `model.ToolCallingChatModel`，`llmcore` 仅保留 provider 共享错误/重试与模型发现类型。
- 更新 `AGENTS.md`：记录本轮 `internal/agent` 清理已删除死 helper、specialist-era 残留 error/export、非 ADK tool 聚合路径与无调用 provider helper，保持包结构围绕单 runtime、工具桥接、provider、memory 与 MCP 五类核心职责。
- 补充 `AGENTS.md`：Agent 会话历史只读恢复不应创建空记忆缓存；仅用于运行时装配的 helper（如 Wails context / 删除 hook 注入）不得直接暴露为 Wails 绑定方法。
- 补充 `AGENTS.md`：记录新增的 Eino ChatModel / Tool 适配层与 `llmcore`↔`schema` 转换文件位置，后续 Eino 集成应优先复用这些适配与转换工具。
- 补充 `AGENTS.md`：记录新增的 Eino ReAct runtime/event bridge 与 `Agent.Analyze()` 的 Eino-first、legacy-fallback 执行路径。
- 补充 `AGENTS.md`：记录 `SpecialistAgent.Analyze()` 已改为优先直连 `EinoRuntime`（复用 `SessionMemory.ToSchemaMessages()`、专家 tool whitelist 与 `SystemPromptBuilder`），失败时再回退到临时 `Agent` 包装路径。
- 补充 `AGENTS.md`：`internal/agent` 相关测试现应直接基于 Eino `schema` / `model` API 编写；provider 用 `Generate` / `Stream` / `WithTools`，SessionMemory 用 `ToSchemaMessages()`，MCP Tool 断言使用 `schema.ToolInfo{Name, Desc}`。

## [previous] - 2026-04-13

### 代码改进
- 将 `internal/storage/storage.go`（1388 行）按职责拆分为 6 个独立文件：`session.go`（Session CRUD + Storage 结构体）、`request.go`（Request CRUD + 过滤/排序）、`search.go`（FindInSession + 关键字搜索）、`provenance_query.go`（值溯源查询）、`export.go`（导入/导出）、`har.go`（HAR 类型与转换）、`helpers.go`（深拷贝/ID 生成/通用辅助）；原 `storage.go` 已删除。
- 移除冗余的 `search_requests_by_host` Agent 工具：删除 `ToolSearchRequestsByHost` 常量、`builtInTools` schema、`executeTool` 分发分支、`toolSearchRequestsByHost` 方法实现与 `Storage.GetHistoryRequests` 方法；内置工具从 13 个减少为 12 个。
- 移除 `Storage.GetHistoryRequests` 方法（功能已被 `search_all_fields` 替代）。

### 修复问题
- 完成桌面/代理配置源合并：移除 `configs/config.yaml` 与 `configs/desktop.json` 双配置并统一切换到 `configs/packetmind.json`，避免运行默认值、证书配置与运行时可变配置分裂。

### 代码改进
- `internal/config/config.go` 移除 Viper 与旧 `Config/ProxyConfig/CertConfig` 结构，新增 `CertSettings`、`DefaultPacketMindSettings`、`LoadPacketMindSettings`、`SavePacketMindSettings`、`MigrateLegacyConfigs`、`CloneForRuntime`，并保留 `DesktopSettingsStore.Update()` 原有外部代理密码保留逻辑不变。
- `internal/proxy` 改为完全依赖 `DesktopSettings`：`proxy.New()` 仅接收 `storage.Storage`，监听端口、MITM 开关与 CA 路径均从 `DesktopSettings` 读取；`internal/proxy/proxy_test.go` 同步更新到新构造方式。
- `internal/api/bindings/config.go` 与 `app.go` 去除对旧全局 `Config` 的依赖；`ConfigAPI.GetConfig()` 只返回 `desktop` + `agent` 配置，桌面配置脱敏结果新增 `cert` 节。
- 新增默认 `configs/packetmind.json`，并在启动阶段 best-effort 执行旧 `config.yaml` + `desktop.json` → `packetmind.json` 首次迁移。

### 文档更新
- 更新 `AGENTS.md` 与 `README.md`，同步记录统一配置文件改为 `configs/packetmind.json`、默认代理端口 `8888`，以及 Viper 已移除。

### 新增功能
- 新增 `internal/version/version.go` 统一版本源，通过 `-ldflags` 在构建时注入 Version/BuildTime/Commit；`app.go` 启动时打印版本信息。
- 新增 `internal/updater/updater.go` 自更新模块，基于 `go-selfupdate v1.5.2` 检查 GitHub Releases、比较当前版本并执行带进度回调的应用内二进制更新。
- 新增 `internal/api/bindings/updater.go` Wails `UpdaterAPI` 绑定，提供 `GetVersion()`、`CheckForUpdate()`、`PerformUpdate()`，并通过 `update:progress` / `update:done` 事件向前端广播更新进度与完成状态。
- 新增 `gui/src/composables/useUpdater.ts` 前端更新状态管理 composable，封装检查/下载/进度/完成状态与 Wails 事件订阅。
- 新增 `gui/src/components/UpdateModal.vue` Charles 风格更新弹窗，支持 checking/available/downloading/complete/error/up-to-date 六种状态。
- `gui/src/App.vue` 接入 UpdateModal 并将 About 弹窗版本号改为动态获取。
- 新增 `RequestAPI.FindInSession(opts)` Wails 绑定与 `Storage.FindInSession(opts)` 后端检索能力，支持前端在单个会话内按请求 URL/头/body、响应头/body、notes、error 执行 Find in Session。

### 修复问题
- 修复 Agent 工具与存储搜索/溯源链路无法读取 gzip/deflate/brotli 压缩 request/response body 的问题；现在 `previewBody`、artifact 提取与 body 搜索都会先按 `Content-Encoding` 解压，再执行文本预览、JSON/form 解析与关键字匹配。

### 代码改进
- 新增 `internal/storage/decode.go`，集中提供基于 `Content-Encoding` 的 gzip/deflate/brotli body 解压辅助函数，供 Agent 与 storage 共享，避免重复维护多套解码逻辑。
- `app.go` 现注入并注册 `UpdaterAPI`，在 Wails `startup()` 阶段传入运行时 context 以支持更新事件广播；新增 `internal/updater/updater_test.go` 覆盖构造、版本读取、更新检测 mock 与下载进度 reader 行为。
- `Makefile` 修复死引用 `./cmd/server`，改为 `wails build` + ldflags 注入；`wails.json` 和 `gui/package.json` 版本同步到 `1.0.13`。
- `.github/workflows/wails-build.yml` 升级 Go 1.25 + Wails v2.11.0，新增构建时 ldflags 版本注入，新增 `release` job 自动打包 `PacketMind_{version}_{os}_{arch}.{zip|tar.gz}` + SHA256 checksums 并创建 GitHub Release。
- `internal/storage/models.go` 新增 `FindInSessionOptions` / `FindInSessionMatch` 数据结构；`internal/storage/storage.go` 增加基于 literal/regex、大小写、whole-word 选项的会话内检索实现，并在 request/response body 搜索前统一复用 `DecodeBodyBytes(...)` 解压正文。

### 文档更新
- 更新 `AGENTS.md`，补充 storage 共享 body 解压能力与 Agent/storage 在 body 预览、artifact 提取、文本搜索前需先解压的约定。
- 更新 `AGENTS.md`，补充 `UpdaterAPI` 绑定、自更新模块目录、`update:progress` / `update:done` Wails 事件、`internal/version` 版本源与 ldflags 构建模式。
- 更新 `AGENTS.md`，补充 `RequestAPI.FindInSession` / `Storage.FindInSession` 会话内检索绑定与匹配选项约定。

## [v1.0.13] - 2026-04-11

### 新增功能
- `gui/src/components/ComposeModal.vue` 新增 Charles 风格 Compose 弹窗，支持从请求右键菜单预填 method/url/headers/body、编辑后发送，并在弹窗内查看响应状态/耗时/正文与复制响应。
- `gui/src/components/RequestList.vue` 新增 Charles 风格请求树右键菜单：请求叶子节点支持 Copy URL / cURL / Response Body、Save Response、Repeat、Agent Analyze、Delete；Host/Folder 节点支持 Copy Host、Expand All、Collapse All。
- `internal/api/bindings/request.go` 新增 `ComposeRequest(opts)` Wails 绑定，支持以自定义 method/url/headers/body 主动发送 HTTP 请求，并复用两阶段录制向前端发出 `request:new` / `request:complete`。

### 代码改进
- `internal/proxy` 请求录制改为 Charles 风格两阶段落库：新增 `saveRequestStart()` / `saveRequestComplete()` 与 `ensureActiveSession()`，代理在请求发出前先写入占位记录并触发 `onRequest`，响应完成后按同一 request id upsert 完整记录并触发新的 `onComplete` 回调。
- `internal/proxy/proxy.go`、`websocket.go`、`socks5.go` 全面切换到两阶段录制路径：HTTP、HTTPS MITM、SOCKS5-HTTP 与 WebSocket 成功链路先发 `request:new` 再发 `request:complete`；终止型错误路径、`recordMinimalError()` 与原始 SOCKS5 tunnel 仅保留 start 阶段事件。
- `app.go` 新增 `emitRequestComplete()` 并通过 `prox.SetOnComplete(...)` 广播 `request:complete` Wails 事件，保持既有 `request:new` 事件格式与 DTO 不变。
- `internal/api/bindings/request.go` 为 `ComposeRequest` 补充默认 active session 解析/创建、自定义请求落库、30 秒超时且禁跳转/跳过 TLS 校验的 HTTP 客户端执行路径，以及成功/失败统一的两阶段请求完成更新；失败场景会把 `502` 与错误文本写回记录。
- `gui/src/App.vue` 将请求树右键 `Compose` 动作接入顶层 modal 状态管理；`gui/src/api/wails.ts` 新增 `requestApi.compose()` 对 `RequestAPI.ComposeRequest` 绑定的前端封装。

### 代码改进
- `gui/src/stores/requestStore.ts` 新增 `updateRequest(updated)` 原地替换能力与 `isRequestPending(req)` helper，供实时请求在 `request:complete` 到达时无闪烁更新并统一判定“进行中”状态。
- `gui/src/composables/useWailsEvents.ts` 新增 `request:complete` Wails 事件订阅，前端收到完整响应后直接回写对应请求记录，保留既有 `request:new` 首次插入行为不变。
- `gui/src/components/RequestList.vue` 为进行中的请求补充 Charles 风格蓝色加载态：请求树节点显示 pulsing `LoadingOutlined` 协议图标、蓝色文字与纯 CSS 小型 spinner，完成后的请求继续沿用既有状态码/耗时展示。
- `gui/src/components/RequestList.vue` 右键菜单采用 `Teleport` 到 `body` 的桌面菜单模式，带视口边界修正、点击外部/滚动/Escape 自动关闭，以及基于 `requestApi.export` / `requestApi.replay` / `copyToClipboard` 的请求快捷操作。

### 文档更新
- 在 `AGENTS.md` 中补齐 `web`→`gui`、`AIAPI`→`AgentAPI`、`ai:analysis`→`agent:analysis`、`Database`→`Storage` 等命名迁移说明，并保持中文 UTF-8 文本可读。
- 刷新 `docs/main/` 与 `docs/modules/` 下当前态中文文档引用，统一 `AgentAPI`、`AgentAnalysis`、`storage.Storage`、`gui/src` 与 `/api/agent/*` 路径说明；历史/阶段记录中的旧称谓保持不动。
- 更新 `AGENTS.md`，补充 `RequestAPI.ComposeRequest` / `ReplayRequest` 等主动发起请求入口也必须遵循 `request:new` → `request:complete` 的两阶段录制约定，以及默认会话回退规则。

### 代码重构
- 将 `internal/storage/database.go` 重命名为 `storage.go`，`Database` 类型重命名为 `Storage`，`NewDatabase()` 重命名为 `NewStorage()`，所有相关变量/参数从 `db` 改为 `store`
- 将 `internal/api/bindings/ai.go` 重命名为 `agent.go`，`AIAPI` → `AgentAPI`，`NewAIAPI()` → `NewAgentAPI()`，`AIAnalysis` → `AgentAnalysis`，`AISettings` → `AgentSettings`，`UpdateAIConfig()` → `UpdateAgentConfig()`，`GetAIProviderKey()` → `GetAgentProviderKey()`
- 后端事件名从 `ai:analysis` 改为 `agent:analysis`，配置键从 `"ai"` 改为 `"agent"`
- 前端目录从 `web/` 重命名为 `gui/`，更新所有构建配置（wails.json、Makefile、CI、.gitignore）
- 前端 AI 命名统一收为 Agent：`aiStore.ts` → `agentStore.ts`，`AIPanel.vue` → `AgentPanel.vue`，`aiApi` → `agentApi`，`AIModel` → `AgentModel`，`AIAnalysis` → `AgentAnalysis`
- 通过 Wails bindings 重新生成确保前后端类型一致

## [1.0.39] - 2026-04-11

### 代码改进
- 将 Vue 3 前端 `gui/src/` 中残留的 AI 命名统一重命名为 Agent：`aiStore`/`AIPanel`/`AIAnalysis`/`AIModel`、Wails `aiApi` 访问层、`ai:analysis` 事件名以及相关 UI 文案已同步切换到 Agent 命名。
- 通过 `git mv` 将 `gui/src/stores/aiStore.ts`、`gui/src/components/AIPanel.vue`、`gui/src/components/AIPanel.module.css` 分别重命名为 `agentStore.ts`、`AgentPanel.vue`、`AgentPanel.module.css`，并更新前端组件、composable 与配置调用引用。

- 将 Go 后端中面向 PacketMind Agent 的 AI 命名统一重命名为 Agent：`AIAnalysis`/`AISettings`/`UpdateAISettings`、`AIAPI` 及相关配置绑定、运行时事件名 `ai:analysis` 与日志前缀已同步调整到 Agent 命名。
- 将 `internal/api/bindings/ai.go` 通过 `git mv` 重命名为 `internal/api/bindings/agent.go`，并同步更新 `app.go`、`internal/config`、`internal/storage`、`internal/agent` 中的后端注释、文档注释与消息文本。

### 测试更新
- 通过 `go build ./...` 验证后端命名重构后的全项目构建。

### 文档更新
- 更新 `AGENTS.md`，记录后端 Wails 绑定主类型/事件名已从 AI 命名切换为 Agent 命名。

## [1.0.38] - 2026-04-10

### 修复问题
- 修复 AI 面板模型选择器在 `default_model` 已失效或前端缓存了旧 model id 时可能落入无效选项的问题；`aiStore.loadModels()` 现在会优先选择有效的 active/current model，其次校验 `default_model`，最后稳定回退到可用分组/平铺模型中的首个有效模型。
- 修复 AI 面板顶部模型下拉只显示 provider 层级、难以正确选择具体模型的问题；AIPanel 现已改为自定义模型选择弹层，避免旧 `a-select` 在窄宽度下的显示异常。

### 代码改进
- `web/src/api/wails.ts` 与 `web/src/types/index.ts`：为 `ListModels()` 补齐强类型返回结构，正式声明 `grouped`、`active_model`、`active_provider`，移除前端对 `as any` 的依赖。
- `web/src/stores/aiStore.ts`：新增模型分组标准化与稳定前端 fallback grouping，确保后端未返回 `grouped` 或分组为空时模型选择器仍可按 provider 正常渲染。
- `internal/api/bindings/ai.go`：`ListModels()` 最小化补充返回当前 `active_model` / `active_provider`，供前端初始化时优先选择真实有效的活动模型。
- `web/src/components/AIPanel.vue`：将顶部模型选择器重构为自定义 popover picker，新增搜索框、provider 分组标题、选中 checkmark、设置入口与平铺 fallback 列表，使其更接近桌面工具风格的模型切换体验。

### 文档更新
- 更新 `AGENTS.md`，补充 `ListModels()` 当前返回 active/default/grouped 模型元数据以及前端初始化回退约定。

## [1.0.37] - 2026-04-10

### 新增功能
- 新增 `internal/agent/registry.go` provider 注册中心，统一描述 provider 元数据、Agent 构造与模型发现入口，并新增 `ListProviders` 绑定供前端读取 provider 状态。
- 前端多 provider 模型切换：`ModelConfigModal` 重构为左右分栏多 provider 管理界面，支持同时配置多个供应商的 API Key / Base URL 并快速切换模型。
- AI 面板模型选择器改为按 provider 分组显示（`a-select-optgroup`），切换即时生效并持久化。
- `aiStore` 新增 `providers`、`modelsGrouped`、`loadProviders`、`switchModel` 等能力，支持 provider 状态追踪与模型切换持久化。

### 代码改进
- `internal/agent/agent.go`：`NewAgentFromProvider` 改为通过 provider registry 解析工厂，移除硬编码 provider 分支，便于后续扩展更多后端 provider。
- `internal/config/config.go`：为 `ModelsStore` 新增 `GetProviders()` 与 `GetModelsGrouped()`，支持返回 provider 配置状态与按 provider 分组的模型列表。
- `internal/api/bindings/ai.go`：`ListModels` 新增 `grouped` 返回；`FetchModels` 改为复用 provider registry 的 `NewLister`；新增 `ListProviders()` 绑定输出 provider 元数据与配置状态。

### 测试更新
- 通过 `go test -v -race ./internal/agent/... ./internal/config/...` 验证 Agent/Config 模块回归。
- 通过 `go build ./...` 验证全项目构建。

### 文档更新
- 更新 `AGENTS.md`，补充 provider registry、provider 列表绑定与 grouped models 约定。

## [1.0.36] - 2026-04-10

### 代码改进
- 将 `internal/ai` 目录整体重命名为 `internal/agent`，并同步更新 `app.go`、`internal/api/bindings/ai.go` 及相关文档中的包路径引用。
- 将 `internal/agent/agent.go` 按职责拆分为 `agent_dispatch.go`、`provider_chat.go`、`provider_adapter.go`、`agent_types.go`，保留既有行为不变并降低单文件复杂度。
- 将 provider 相关能力进一步拆分为 `internal/agent/llmcore`（共享 LLM 协议类型、结构化错误、retry、SSE）与 `internal/agent/provider`（OpenAI/Zhipu 实现与 provider chat 适配），并通过根 `internal/agent` 的 type alias / 构造函数 facade 保持外部 API 与行为兼容。

### 测试更新
- 通过 `go build ./...` 验证重命名与拆分后的全项目构建。
- 通过 `go test -v -race ./internal/agent/...` 验证 Agent 模块回归。
- 通过 `go test ./internal/agent/...` 验证 `llmcore/provider` 重组后的 Agent/provider 回归；通过 `go build ./...` 验证全项目构建保持通过。

### 文档更新
- 更新 `AGENTS.md`、README 与 `docs/` 中关于 `internal/agent` 的目录与源码路径说明。
- 补充 `internal/agent/llmcore`、`internal/agent/provider` 与根 `agent` facade 的职责说明。

## [1.0.35] - 2026-04-09

### 新增功能
- AI 面板实时展示 provider 重试进度：当 AI 模型请求因 429/5xx 触发退避重试时，前端时间线显示 `🔄 正在重试第 N/3 次` 动画卡片。

### 代码改进
- `internal/ai/retry.go`：`retryProviderCall` 新增可选 `RetryNotifyFunc` 回调，在重试间隔前通知调用方当前重试状态；通过 `context.Value` 传递 notifier 避免侵入 `LLMClient` 接口。
- `internal/ai/runtime.go`：`ConversationRuntime.Run()` 在每次 LLM 调用前注入 retry notifier，重试时自动通过 `r.emit()` 发射 `provider_retry` 事件。
- `internal/ai/types.go`：`AgentEvent` 新增 `retry_attempt` / `retry_max` 字段。
- `internal/api/bindings/ai.go`：`emitAgentEvent` 透传 retry 字段到前端。
- 前端 `useWailsEvents.ts` / `aiStore.ts` / `AIPanel.vue`：新增 `provider_retry` 事件类型处理与带旋转动画的黄色重试卡片样式。

## [1.0.34] - 2026-04-09

### 修复问题
- 修复 AI provider/model 遇到瞬时失败时立即中断分析的问题；现在对 HTTP `429/500/502/503/504` 与瞬时网络错误执行有界退避重试，耗尽后返回结构化 retryable `AgentError`，非重试型 4xx 保持 fail-fast fatal 语义。

### 代码改进
- 新增 `internal/ai/retry.go`：集中实现 provider 重试、HTTP/网络错误分类、退避等待与结构化 provider error 类型。
- `internal/ai/agent.go`：将共享 `doProviderChat()` / `doProviderChatStream()` 接入统一 retry chokepoint，供 Agent `Chat` / `ChatStream` 共用。
- `internal/ai/openai.go` 与 `internal/ai/zhipu.go`：legacy `Complete()` / `Stream()` 路径接入同一套 retry 分类逻辑。
- `internal/ai/errors.go`：`AsAgentError()` 改为支持从 wrapped error chain 中提取 `AgentError`。

### 代码重构
- 将 `internal/proxy/proxy.go` 按功能拆分为多个文件以提高可维护性：
  - `internal/proxy/transport.go`（共享 transport、外部代理解析、上游连接/CONNECT/SOCKS5 上游拨号逻辑）
  - `internal/proxy/cert.go`（CA 加载/生成与按域签发证书逻辑）
  修改仅为代码组织调整，功能保持不变；已通过 `internal/proxy` 包回归测试。

### 代码改进 (补充)
- 移除 `internal/proxy/proxy.go` 中对已迁移 helper (`newSharedTransport` / `sharedHTTPClient`) 的过期注释，保持文件注释与实际代码位置一致。

### 修复问题（临时修复）
- 修复 `internal/proxy` 中因导入残留导致的编译错误：清理未使用的 import 并修复导入块不一致问题；已在 Windows 上运行 `go test ./internal/proxy` 并通过回归测试。

### 测试更新
- `internal/ai/openai_test.go`、`internal/ai/zhipu_test.go`：新增 429 重试成功、非重试 4xx fail-fast 场景覆盖。
- `internal/ai/errors_test.go`：新增 wrapped `AgentError` 提取测试。

## [1.0.33] - 2026-04-08

### 修复问题
- 修复 `wails dev` 在端口 3000 被占用时硬失败的问题：移除 `web/vite.config.ts` 中的 `strictPort: true`，允许 Vite 自动递增端口；`wails.json` 新增 `viteServerTimeout: 30` 给予前端启动足够等待时间。Playwright E2E 配置不受影响，仍固定使用端口 3000。

## [1.0.32] - 2026-04-07

### 修复问题
- 修复 HTTPS MITM 请求体在录制后未恢复导致上游转发收到空 body 的问题，并修正 MITM keep-alive 判断，避免 HTTP/1.1 长连接在首个请求后被错误关闭。
- 修复 CONNECT 在确认 hijack 能力前提前写入 200 响应的问题；现在仅在成功 hijack 原始连接后返回 `200 Connection Established`，并在失败时记录结构化错误。
- 修复 MITM 上游请求失败时仅落库不回包导致客户端悬挂的问题；现在会立即返回 `502 Bad Gateway` 并关闭连接。
- 修复 HTTP/MITM 响应转发未剥离 hop-by-hop 头的问题，避免代理错误转发 `Connection`、`Transfer-Encoding` 等仅限单跳头部。

### 新增功能
- 新增 `internal/ai/prompts/max_depth.txt`，用于 ReAct agent 达到最大推理深度时引导模型输出最终文本总结。

### 代码改进
- `internal/ai/runtime.go`：将最大推理深度行为从硬停止改为 soft-stop；最后一次允许迭代会注入 `max_depth` 提示并禁用 tools，让模型走既有无工具最终回答路径自然收敛。
- 保持 `agent.go` 中 `stopEarly` / `buildFallbackAnswer` 不变，继续服务于 `tool_budget_exhausted`、`repetition_limit_reached`、`empty_final_answer` 等提前停止场景。
- `internal/proxy/proxy.go`：为 MITM 路径复用单个 `http.Client`/`Transport`，避免每个请求重复创建连接池；MITM 上游超时提升到 60s，共享 HTTP client 总超时提升到 120s。
- `internal/proxy/proxy.go`：统一放宽代理服务器超时策略，将 `ReadTimeout` 提升到 60s，并关闭 `WriteTimeout` 以兼容长时间流式响应。

## [1.0.31] - 2026-04-05

### 新增功能
- 新增 `internal/permission` 包，提供基于 `path.Match` 的权限规则存储与匹配能力，支持按工具名/候选值执行 `allow`、`deny`、`ask` 决策，并采用“最后匹配规则优先”语义。

### 代码改进
- `internal/permission/permission.go`：新增 `Store`、`Rule`、`Action` 及基于 JSON 文件的原子持久化实现；缺失文件时自动初始化为空规则，损坏文件时记录 warning 并安全回退。
- `internal/permission/permission_test.go`：新增规则匹配、通配符、覆盖顺序、持久化加载、删除规则、缺失/损坏文件等回归测试。

## [1.0.30] - 2026-04-03

### 修复问题
- 修复 `Stop Proxy` / `Start Proxy` 按钮常态下显示灰白色背景的问题：改用 `type="text"` 避免 Ant `ant-btn-default` 全局样式覆盖，按钮现在常态下即可正确显示红色/绿色渐变背景。

- 修复上下文弹窗（ContextModal）整体窗口可滚动的问题：改为受限高度桌面弹窗，内容区固定在 `76vh` 内，仅内部列表/查看器滚动。
- 重构上下文弹窗布局为更适合阅读的双栏 inspector：顶部使用紧凑摘要卡片，底部改为左侧消息选择列表、右侧常驻原始结构查看器，替代原先不适合阅读的行内展开式原始消息框。
- 为上下文弹窗新增右下角拖拽尺寸手柄，改为由前端显式维护宽高状态并实时更新 `a-modal` 的 `width/body height`，在桌面环境下可稳定手动调整窗口尺寸。

## [1.0.29] - 2026-04-03

### 修复问题
- 移除抓包列表空状态中的硬编码代理提示文案（`Configure your browser to use proxy 127.0.0.1:8888`），避免监听端口变化后 UI 仍显示过期端口。
- 修复抓包列表面板在极窄宽度下顶部工具条换行错位的问题；`Total / Expand All / Collapse All` 现固定保持单行展示。

### 代码改进
- 顶部标题栏代理开关按钮改为语义化配色：`Start Proxy` 使用绿色背景，`Stop Proxy` 使用红色背景，并保留浅色桌面工具风格。
- `toggleProxy()` 启动成功提示改为读取实时监听端口，不再固定写死 `8888`。
- AI 面板标题由 `AI Analysis` 调整为 `AI AGENT`，与产品定位一致。

## [1.0.28] - 2026-04-03

### 新增功能
- AI 面板思考过程全面重构为 ReAct 循环时间线视图：
  - **ReAct 循环分组**：自动将连续的 thought/action/observation/error/decision 消息按逻辑步骤分组为带编号的循环，取代原先的扁平列表。
  - **垂直时间线**：每步以编号圆点 + 竖向连接线排列，步骤之间有清晰的时间线视觉。
  - **Tool Call 卡片**：Action + Observation 合并为单个可展开/折叠的工具卡片，显示工具名、状态指示器（✓ 成功 / ✗ 错误 / 蓝色脉冲进行中）、结构化参数、完整结果。
  - **思考动画**：Agent 工作中显示渐变扫光 "思考中..." 动画指示器。
  - **摘要药丸**：思考卡片折叠态显示每步摘要药丸（💭 思考 / 🔧 工具名 / 🎯 决策 / ❌ 错误），取代原先仅显示最后一步截断文本。
  - **Markdown 渲染**：思考内容和观测结果现在支持完整 Markdown 渲染（代码块、列表、强调等）。
  - **步骤耗时**：显示每个 ReAct 循环的执行耗时（如 "2.3s"）。
  - **轮次分隔线**：多轮对话之间添加带时间戳的水平分隔线。

### 修复问题
- 修复命令权限提升弹框（PermissionDialog）在浅色桌面主题下文字/标签/代码块几乎不可读的问题；将暗色硬编码色值对齐到项目浅灰工具色系。
- 修复 bash 工具执行后 observation 只显示 "bash exited 0" 而不包含 stdout/stderr 输出的问题；Summary 现在包含最多 500 字符 stdout 预览和 200 字符 stderr 预览。

### 代码改进
- 清理 Agent Direct 通道残余代码：删除 3 个无引用 prompt 文件（direct_chat_system、routing_classifier_with/without_context）、response_headers 路由决策功能（routerDecision 结构体、decideResponseHeaderMode* 方法、router_system/user prompt）及其 4 个测试用例。所有分析路径统一走 Agent 工具循环，不再有 direct 捷径分支。

## [1.0.27] - 2026-04-03

## [1.0.26] - 2026-04-03

### 修复问题
- 修复无目标自由问答（无 RequestID）时 Agent 系统提示词仍注入 traffic-search-only 工具指令的问题；现在 general/local 类问题使用不预设搜索导向的提示词，避免模型强制调用抓包搜索工具。
- 修复 specialist agent 克隆时未传播 `danger-full-access` 权限等级与 `permissionChecker` 回调的问题；现在 bash 等高权限工具在 specialist 路径中可正常通过权限校验。
- 调整 no-target 分析起始 trace 文案为中性描述，不再默认声明一定会先搜索抓包请求。

## [1.0.25] - 2026-04-02

### 新增功能
- Agent 新增快速分析工具 `analyze_encoding`，支持对输入值执行多层编码链识别与分层解码预览（如 base64/hex/url/json/gzip/zlib 组合）。
- Agent 新增请求对比工具 `diff_requests`，支持按 `meta`/`headers`/`body`/`cookies`/`query`/`timing` 分组做双请求差异比对。
- Agent 新增并行批量工具调用 `batch_execute`，支持在单次工具调用中并发执行多条子调用并聚合结果。

### 代码改进
- 新增 `internal/ai/tools_encoding.go`：实现 `encodingDetect(value string)` 与 `toolAnalyzeEncoding(...)`，输出结构化编码链、最终类型、预览与错误信息。
- 新增 `internal/ai/tools_diff.go`：实现 `toolDiffRequests(...)`，统一字段解析与差异输出结构。
- 新增 `internal/ai/tools_batch.go`：实现 `toolBatchExecute(...)` 与批量调用参数结构，支持并发执行与结果聚合。
- `internal/ai/tools_search.go`：完善 `search_all_fields` 返回结构，统一输出 `found_results`、`count`、`error_summary`。
- `internal/ai/tools.go`：补充 `ToolAnalyzeEncoding`、`ToolBatchExecute` 常量与 built-in tool schema，并同步 `diff_requests` / `search_all_fields` 定义。
- `internal/ai/agent.go`：`executeTool` 新增 `analyze_encoding` / `batch_execute` 分发，`allToolNames()` 同步纳入新增工具。
- `internal/ai/registry_test.go`：更新 built-in 工具清单与数量断言，覆盖新增快速分析工具注册。

### 测试更新
- 通过 `go build -buildvcs=false ./...` 验证构建。
- 通过 `go test -v -race ./internal/ai/...` 验证 AI 模块回归。
- 执行 `go test -v -race ./...` 覆盖全量回归。

## [1.0.24] - 2026-04-02

### 新增功能
- Agent 新增跨字段检索工具 `search_all_fields`，可在单次工具调用中同时搜索请求头、请求体、响应体、Cookie、Query 参数，快速定位指定值在抓包数据中的全部出现位置。

### 代码改进
- `internal/ai/tools.go`：新增 `ToolSearchAllFields` 常量与 built-in tool schema（`value` 必填，支持 `session_id` 与 `limit`）。
- `internal/ai/agent.go`：`executeTool` 新增 `ToolSearchAllFields` 分发分支，并将该工具加入 `allToolNames()` 以支持 unknown-tool 提示与模糊匹配。
- `internal/ai/tools_search.go`：补充跨字段搜索实现（按请求维度聚合匹配字段与预览），并返回结构化 `found_results` 与错误摘要。
- `internal/ai/specialist_config.go` 与 `configs/specialists.json`：将 `search_all_fields` 加入 `header`/`body` specialist 白名单，支持专家路径直接使用该工具。

### 测试更新
- `internal/ai/agent_test.go`：新增 `TestExecuteTool_SearchAllFields`，覆盖 headers/body/response/cookies/query 多字段命中与返回结构断言。
- `internal/ai/registry_test.go`：更新默认工具清单与 built-in 数量断言，确保 `search_all_fields` 已注册并可被默认注册表获取。

## [1.0.23] - 2026-04-02

### 新增功能
- Agent 工具结果截断升级为行感知截断服务：新增 `internal/ai/truncate.go`，替代原有基于字节的粗暴截断，当工具输出超过限制（2000 行 / 50000 字节）时将完整内容写入 `data/tool-output/` 并返回预览 + 文件路径提示。
- 新增 `CleanupOldTruncationFiles()` 自动清理超过 7 天的截断输出文件。

### 代码改进
- `internal/ai/truncate.go`：新增 `TruncateOutput()`、`CleanupOldTruncationFiles()`、`TruncateResult` 结构体与相关辅助函数（原子写入、工具名清理、行截断预览）。
- `internal/ai/runtime.go`：`ConversationRuntime.Run()` 中工具 observation 截断从 `truncateForModel` 切换为 `TruncateOutput`，保留原始 `truncateForModel` 供其他路径继续使用。

### 文档更新
- 更新 `AGENTS.md`：补充行感知截断服务与 `data/tool-output` 目录约定。

## [1.0.23] - 2026-04-02

### 新增功能
- AI 工具执行链路新增轻量 JSON Schema 参数校验：在 `executeTool()` 实际分发前，先按工具声明的 `parameters` 校验参数类型与约束，校验失败时返回结构化 `ok=false` 错误与自纠正提示，帮助模型在同轮推理中修正参数。

### 代码改进
- 新增 `internal/ai/schema_validate.go`：实现无外部依赖的 schema 校验能力，覆盖当前工具实际使用的 `type(string/integer)`、`required`、`enum`、`minimum`、`maximum` 约束。
- `internal/ai/agent.go`：`executeTool()` 在参数解析后新增 schema 校验分支，失败时返回结构化 observation（`error` + `hint`）而非继续执行工具。
- `internal/ai/agent.go`：新增 `getToolSchema(name)`，优先从 Agent 自定义 `ToolRegistry` 读取工具定义，缺省回退到 `DefaultToolRegistry()`，确保校验与运行时工具注册源一致。

### 文档更新
- 更新 `AGENTS.md`：补充 Agent 工具执行需在 `executeTool()` 分发前执行参数 schema 校验及失败返回约定。

## [1.0.22] - 2026-04-02

### 新增功能
- 代理监听升级为单端口统一入口：HTTP、HTTPS CONNECT 与 SOCKS5 现在共享同一个 TCP 监听端口，并通过首字节自动协议识别分发。
- Agent 工具分发新增 Invalid Tool sentinel：当模型调用不存在的工具时，`executeTool` 现在返回结构化 observation（包含 `available_tools` 与可选 `suggestion/hint`），不再直接以错误中断 ReAct 循环。

### 修复问题
- 修复双 listener 架构下 HTTP/HTTPS 与 SOCKS5 端口配置分裂导致的运行态不一致问题；现在三种协议统一走同一监听端口，避免热重载时双端口状态漂移。

### 代码改进
- `internal/proxy/proxy.go`：移除独立 `socks5Listener` 与 SOCKS5 独立 accept loop，新增 `serveUnified()` + `detectAndHandle()`，基于首字节 `0x05` 判定 SOCKS5，否则按 HTTP 路径处理。
- `internal/proxy/proxy.go`：新增 `singleConnListener` 适配单连接给 `http.Server.Serve(...)`，复用现有 `ServeHTTP`/CONNECT/MITM 处理链而不改动 HTTP/HTTPS 业务逻辑。
- `internal/proxy/proxy.go`：`Start/Stop/restartListenersIfNeeded` 重构为单 listener 生命周期管理，保留“预绑定成功后原子切换，旧 listener 锁外关闭”的热重载安全语义。
- `internal/proxy/proxy.go`：新增 `httpEnabled()` / `socks5Enabled()` helper，`effectiveHTTPPort` 重命名为 `effectivePort`，统一端口解析语义。
- `internal/config/config.go`：`DefaultDesktopSettingsFromConfig` 与 `DesktopSettings.normalize` 增加回兼容归一化，确保 `socks5_port`/`https_port` 运行态对齐 `http_port`（统一监听端口）。
- `internal/proxy/proxy_test.go`：测试辅助改为单端口构造（`newTestProxyForPort` / `newStartedTestProxy`），SOCKS5 热重载测试改为开关行为验证，端口校验用例改为 unified port 非法值校验。
- `internal/ai/agent.go`：`executeTool` 默认分支改为“先尝试 MCP，再返回结构化 unknown-tool observation”，并新增工具名模糊匹配（大小写精确、前缀匹配、Levenshtein 距离）与可用工具列表输出。

### 文档更新
- 记录统一端口 listener 架构变更、Invalid Tool sentinel 行为与测试调整说明。

## [1.0.21] - 2026-04-01

### 新增功能
- AI 智能路由加固：引入三阶段分类管线（确定性 Agent 预检 → 确定性 Direct 预检 → LLM 分类），通用知识问题（如"介绍一下美国"）不再被误路由到 Agent 工具分析模式。
- AI 面板上下文弹窗：AI Analysis 行新增圆形按钮，点击后打开"上下文"弹窗，展示当前会话的 session ID、消息数、短期记忆用量、模型、token 统计与上下文分配明细。
- AI 面板智能滚动：流式生成时不再强制自动滚动到底部；用户向上浏览时显示"跳到最新"浮动按钮，点击后恢复跟随。

### 修复问题
- 修复有 `request_id` 时所有查询均走 Agent 模式的问题；现在即使有选中请求，自由问答仍先过路由分类再决定走 direct 还是 agent。
- 修复 `deterministicRouteAgent` 测试用例与实际匹配模式不一致的问题。
- 修复上下文弹窗“原始消息”展开后只能看到拆散字段、无法按原始 message 结构阅读的问题；现在展开态改为接近 inspector 的 JSON 原始视图，并带行号展示。

### 代码改进
- `internal/api/bindings/ai.go`：新增 `deterministicRouteDirect()`（双语问候/通用知识/协议概念关键词预检）与 `deterministicRouteAgent()`（特定数据引用/请求搜索/字段分析关键词预检），分类失败默认走 DIRECT 安全降级。
- `internal/api/bindings/ai.go`：新增 `GetSessionContext()` Wails 绑定，返回 `SessionContextStats` 结构（session ID、消息数、记忆限制、模型、token 用量、缓存状态等）。
- `internal/api/bindings/ai.go`：路由分类 prompt 切换为英文并增加"When uncertain, choose DIRECT"引导。
- `internal/api/bindings/ai.go`：`Analyze()` 路由逻辑重构为：`Query != ""` 时统一走 `runDirectChat()`（内含三阶段分类），字段分析（无 Query）仍走 AGENT。
- `internal/api/bindings/ai_test.go`：新增 `TestDeterministicRouteDirect`、`TestDeterministicRouteAgent`、`TestParseRoutingClassifierResponse` 单元测试。
- `web/src/components/ContextModal.vue`：新增上下文弹窗组件，展示会话统计、token 明细、上下文分配条与原始消息列表。
- `web/src/components/ContextModal.vue`：原始消息展开态改为 raw viewer 风格；标题行显示 `role • msgId`，展开后以 JSON 代码查看器样式展示 message/meta/summary/parts 结构并附带行号。
- `web/src/components/AIPanel.vue`：新增 `isAutoFollowing` 滚动状态追踪、`handleScroll` 事件处理、浮动跳转按钮；Header 新增上下文圆形按钮。
- `web/src/stores/aiStore.ts`：新增 `sessionContext` 状态与 `fetchSessionContext()` action。
- `web/src/api/wails.ts`：新增 `getSessionContext()` API 方法。
- `web/src/types/index.ts`：新增 `SessionContextStats` 接口。

### 文档更新
- 更新 `AGENTS.md`：补充三阶段路由分类架构、上下文弹窗绑定、智能滚动 UX 约定。

## [1.0.20] - 2026-04-01

### 新增功能
- `web/src/components/AIPanel.vue`：新增分析完成后的 token usage 展示条，显示 input/output/total token 统计，支持流式完成后在 AI 回复区域直接查看本轮消耗。

### 修复问题
- 修复 AI 思考面板中 `agent_error` 事件缺少显式标签的问题，现可在思考时间线中清晰区分错误步骤。
- 修复专家信息未进入前端消息模型导致思考面板无法显示 specialist 的问题，现支持在思考步骤中展示 specialist 徽标。
- 修复 AI 思考面板依赖未维护的独立 trace 状态导致实际有推理步骤时仍可能不显示的问题；现在 trace 可见性直接从真实消息时间线派生。
- 修复 direct chat 场景分析完成后缺少 `tokens_used` 回退字段的问题；现在即使没有详细 `token_usage`，前后端也会保留总 token 回退值用于展示。
- 修复 direct chat 非流式完成路径丢弃 provider usage 的问题；当 OpenAI/Zhipu 返回 usage 时，`agent_result` 现在会携带真实 `token_usage` 与 `tokens_used`，不再固定为 0。
- 修复右侧 AI 面板同时渲染主时间线 trace 与底部折叠 trace 导致思考过程重复显示的问题；现在仅保留主时间线中的黄色思考区，并通过单一折叠开关控制显示。
- 修复 AI 面板思考区的折叠按钮与内容块割裂、观感像两个模块的问题；现在思考过程改为单一一体化卡片，默认折叠并保留最新思考预览。
- 修复 token usage 仅显示 `N/A`、对用户没有实际意义的问题；当 provider 仍未返回 usage 时，前端现在会基于最终回答文本给出明确标注的 estimated total token 回退值。

### 代码改进
- `web/src/stores/aiStore.ts`：`Message` 与 `addMessage` 选项新增 `specialistName` 字段；`addAgentEvent` 在 thought/action/observation/decision/warning/start 分支透传 `event.specialist_name`。
- `web/src/stores/aiStore.ts`：`agentTraceChain` 改为从 `messages` 派生的计算属性，`hasAgentTrace` 与思考面板显示状态现在基于真实 agent 时间线工作。
- `web/src/components/AIPanel.vue`：新增更显眼的 token summary block，并将 specialist 徽标同时渲染到主消息流与思考面板；思考面板角色映射补充 `agent_error`。
- `web/src/components/AIPanel.vue`：移除底部重复思考面板，改为在主消息流中对连续 agent trace 片段插入单一折叠开关；token 使用统计移动到底部输入区上方的固定 footer bar，避免长回答把统计条冲出可视区。
- `web/src/components/AIPanel.vue`：进一步将连续 trace 消息重构为单一 integrated thought card；收起态显示最新 trace 预览，展开态保留完整 thought/action/observation/error/decision 明细与 specialist 徽标。
- `web/src/stores/aiStore.ts`：trace 折叠状态默认改为收起，新分析不会再默认铺开整段思考内容。
- `internal/api/bindings/ai.go`：direct chat 的所有 `agent_result` 结束事件统一补充 `tokens_used`（无详细 usage 时固定回退为 0），对齐 agent path 结束事件契约。
- `internal/ai/openai.go` 与 `internal/ai/zhipu.go`：`Complete()` 签名调整为同时返回回复文本与 `*TokenUsage`，并在 provider 响应包含 usage 时映射输入/输出 token。
- `internal/api/bindings/ai.go`：direct chat 非流式 fallback 改为消费 `Complete()` usage，并在结束事件内补齐 `token_usage` 与 `tokens_used`（缺失 usage 时继续回退为 0）；流式路径保持现有安全回退行为。
- `internal/api/bindings/ai.go` 与 `internal/ai/types.go` / `internal/ai/zhipu.go`：direct chat 流式路径在 provider stream chunk 带 usage 时也会透传真实 `token_usage/tokens_used`；无 usage 时继续保留安全回退。
- `internal/ai/openai_test.go` 与 `internal/ai/zhipu_test.go`：同步 `Complete()` 新签名，补充 usage 缺失场景断言，保持 mock/错误路径测试覆盖不回退。
- `web/src/composables/useWailsEvents.ts`：`agent_result` 桥接时新增透传 `tokens_used` 到 `aiStore.setAgentResult(...)`。
- `web/src/stores/aiStore.ts`：新增 `tokensUsed` 标量状态，`setAgentResult` 在 `token_usage` 缺失时保留 `tokens_used` 回退值，并在 `startStreaming` / `analyzeWithQuery` / `clearMessages` 生命周期与 `tokenUsage` 同步重置。
- `web/src/types/index.ts`：`TokenUsage` 新增可选 `total_tokens` 字段，前端可在详细 usage 可用时优先消费后端总量。
- `web/src/components/AIPanel.vue`：token footer bar 在无 provider usage 时改为展示基于最终回答文本估算的 total，并显式标注 `estimated`，避免出现纯 `N/A` 摆设。

### 文档更新
- 更新 `AGENTS.md`：补充前端 AI 面板对 token usage 展示（含 estimated 回退）、specialist 徽标渲染、单一卡片式思考区与 trace 可见性派生方式的约定。

## [1.0.19] - 2026-04-01

### 新增功能
- 新增 `internal/ai/runtime.go`：引入可复用 `ConversationRuntime`，用于承载 Agent ReAct 多轮会话循环（LLM 调用、工具执行、权限校验、重复调用拦截、流式文本增量事件、token usage 累计）。

### 代码改进
- `internal/ai/runtime.go`：新增 `RuntimeResult`、`RuntimeOption` 及 `WithToolExecutor/WithPermissionLevel/WithMaxIterations/WithMaxToolCalls/WithSystemPrompt/WithEventHandler`，支持通过函数式配置构建 runtime。
- `internal/ai/agent.go`：`Analyze()` 重构为薄封装，保留请求校验、目标请求加载、response-header 路由决策与 trace/event 聚合；核心 ReAct 循环改为委托 `ConversationRuntime.Run()`。
- `internal/ai/agent.go`：保留 `safeExecuteTool()` 作为工具执行保护层，通过 runtime 注入 `WithToolExecutor(...)` 复用既有超时与 panic recovery 行为。
- `internal/ai/agent.go`：删除已抽离到 runtime 的权限与循环辅助逻辑（`authorize`、`permissionSufficient`、重复调用签名与 token usage 累计 helper），避免职责继续堆叠在 Agent 主文件。
- `web/src/composables/useWailsEvents.ts`：补齐 `text_delta` 事件内容映射（优先 `agent_event.content`，缺失时回退顶层 `content`），并在 `agent_result` 路径向 store 透传 `token_usage` 与 `tokens_used`。
- `web/src/stores/aiStore.ts`：新增 `tokenUsage` 状态；`addAgentEvent` 支持 `text_delta` 累加更新 `currentContent`；`setAgentResult` 保存 `token_usage`，并在 `startStreaming`/`analyzeWithQuery`/`clearMessages` 中重置，供 AI 面板按分析维度读取 token 统计。

### 文档更新
- 更新 `AGENTS.md`：补充 ReAct 运行时已拆分为 `ConversationRuntime`，并将工具权限校验约定同步到 runtime 侧。
- 更新 `AGENTS.md`：补充前端 Wails 事件桥接需透传 `text_delta` 内容与 `agent_result.token_usage/tokens_used` 的约定。
- 更新 `AGENTS.md`：补充 AI store 需对 `text_delta` 做增量内容拼接并维护 `tokenUsage` 状态的约定。

## [1.0.18] - 2026-04-01

### 新增功能
- Agent ReAct 在文本直答（无 tool calls）场景新增增量流式输出：分析过程中会逐段发送 `text_delta` 事件，前端可按 token/chunk 逐步渲染最终回答。

### 代码改进
- `internal/ai/types.go`：为 `LLMClient` 新增 `ChatStream(ctx, req)` 方法，保留既有 `Chat()` 阻塞调用不变。
- `internal/ai/agent.go`：新增基于 `LLMChatRequest` 的 provider chat-completions SSE 流式适配（`doProviderChatStream`），并为 `ZhipuClient` / `OpenAIClient` 实现 `ChatStream()`。
- `internal/ai/agent.go`：ReAct 循环在“无工具调用”的文本阶段优先尝试 `ChatStream()`，逐块透传 `text_delta`，流完成后组装最终答案；当流式不可用或出错时自动回退到已获取的阻塞 `Chat()` 结果。
- `internal/ai/agent.go`：保留工具调用阶段原有 `Chat()` 路径，确保 tool-call 步骤行为不变。

### 文档更新
- 更新 `AGENTS.md`：补充 Agent 文本直答步骤应优先使用 `ChatStream` + `text_delta` 事件渐进输出，并在 provider 不支持流式时回退阻塞 `Chat()` 的约定。

## [1.0.17] - 2026-04-01

### 新增功能
- Agent 工具执行链路新增权限控制基础能力：`Agent` 现可持有运行权限级别（`permissionLevel`），并支持通过 `SetPermissionLevel(...)` 在运行时配置。

### 代码改进
- `internal/ai/agent.go`：新增 `permissionSufficient(required, current PermissionLevel)` 权限层级比较辅助函数（`read-only < workspace-write < danger-full-access`）。
- `internal/ai/agent.go`：新增 `authorize(toolName string) error`，在工具执行前按注册表中的 `Function.PermissionLevel` 与 Agent 当前权限级别进行校验；当工具或权限信息缺失时采用安全默认（按 `read-only` 处理）。
- `internal/ai/agent.go`：在 ReAct 循环中于 `safeExecuteTool(...)` 前增加权限检查；权限不足时不执行工具，转为结构化 blocked observation（`ok=false, blocked=true, error=...`）。
- `internal/ai/agent.go`：`NewAgent(...)` 默认初始化 `permissionLevel=PermissionReadOnly`，保持现有只读工具链路行为兼容。

### 文档更新
- 更新 `AGENTS.md`：补充 Agent 工具执行前必须经过权限校验（`authorize`）与权限级别比较约定。

## [1.0.16] - 2026-04-01

### 修复问题
- 修复 Agent 工具失败事件缺少结构化错误元数据的问题；现在 `agent_event`（observation）与顶层 `error` 事件都会携带 `error_category`、`error_tool_name`、`error_timeout`、`error_recovered`，前端可准确区分超时、panic 恢复与普通失败。
- 修复 Agent provider chat 响应未解析 `usage` 字段的问题；现在会从 OpenAI/Zhipu chat completion 响应中提取真实 token 用量并透传到 Agent 结果。

### 代码改进
- `internal/ai/agent.go`：在 ReAct 工具执行循环中为失败 observation 事件补齐 `AgentEvent` 错误字段透传（含类别、工具名、超时、恢复标记），并保持工具结果 JSON 供模型自恢复。
- `internal/ai/agent.go`：新增 `providerUsage` 与 `providerChatResponse.Usage` 解析，将 provider usage 映射到 `LLMChatResponse.Usage`，并在响应头路由决策 + ReAct 循环中累计到 `AgentResult.TokenUsage`。
- `internal/api/bindings/ai.go`：`emitAgentEvent` 增加错误字段与 `specialist_name` 序列化；`emitError` 改为接收 `error` 并在可识别为 `AgentError` 时透传结构化错误元数据。
- `internal/api/bindings/ai.go`：`runAgentAnalysis` 保存分析记录时写入 `AIAnalysis.TokensUsed`，`emitAgentResult` 追加 `final_answer`、`token_usage` 与 `tokens_used` 字段（无 usage 时为 0）。
- `web/src/types/index.ts`：扩展 `AgentEvent` 类型，新增结构化错误字段与 `specialist_name` 字段定义。
- `web/src/composables/useWailsEvents.ts`：接收并转发 `agent_event/error` 的结构化错误字段；顶层错误消息追加可读 `Error Details`。
- `web/src/stores/aiStore.ts`：新增 `agent_error` 时间线消息类型；当 observation 带错误元数据时转为 `agent_error` 消息并保留错误字段。
- `web/src/components/AIPanel.vue`：新增 `agent_error` 渲染与样式，展示 error category/tool/timeout/recovered 细节。

### 文档更新
- 更新 `AGENTS.md`：补充 Agent 工具失败 observation 与绑定层 error 事件必须透传结构化错误元数据，并要求前端按 `agent_error` 进行差异化展示。

## [1.0.15] - 2026-04-01

### 修复问题
- 修复 `SpecialistAgent` 工具白名单仅过滤内置工具、遗漏 MCP 已注册工具的问题；现在专家工具集会同时纳入内置工具与 MCPManager 注册表中的工具，并按 `ToolWhitelist` 统一过滤。

### 代码改进
- `internal/ai/specialist.go`：`NewSpecialistAgent` 在构建工具列表时合并 `DefaultToolRegistry()` 与 `baseAgent.mcpManager.Registry()`；当白名单为空时放开全部内置+MCP 工具，当白名单存在时对两类工具统一按名称筛选。
- `internal/ai/specialist.go`：`SpecialistAgent.Analyze()` 创建临时 Agent 时显式透传 `mcpManager`，确保专家分析路径中的 MCP 工具执行委托保持可用。
- `internal/ai/prompt.go`：新增可组合的 `SystemPromptBuilder`（支持 `WithRole` / `WithInstructions` / `WithContext` / `WithSection` / `Build` / `Reset`），用于按段构建系统提示词并提供 nil-safe 链式调用。
- `internal/ai/agent.go`：`Analyze()` 改为使用 `SystemPromptBuilder` 组装系统提示词，保留 `agentSystemPrompt` 常量用于兼容旧引用并改为复用 `defaultAgentRoleText`。

### 文档更新
- 更新 `AGENTS.md`：补充 Specialist 工具白名单应同时覆盖内置工具与 MCP 注册工具的约定。
- 更新 `AGENTS.md`：补充 Agent 系统提示词应优先通过 `SystemPromptBuilder` 组合构建，`agentSystemPrompt` 仅用于兼容旧代码路径。

## [1.0.14] - 2026-04-01

### 代码改进
- `internal/ai/memory.go`：实现 SessionMemory 会话压缩能力，新增 `EstimateTokens()`（chars/4 估算）、`ShouldCompact(config)` 与 `CompactWithConfig(config)`，当消息估算 token 超过阈值时将旧消息压缩为单条 system 摘要并保留最近 N 条消息，且始终保留 context 键值。
- `SessionMemory.Compress()` 改为调用 `CompactWithConfig(DefaultCompactionConfig())`，统一走可配置压缩路径。
- `internal/ai/memory.go`：新增 `SessionMemorySnapshot`、`Snapshot()`、`SaveToPath(path)`、`LoadFromPath(path)`，支持带 `version/messages/context/limit` 的 JSON 持久化；保存采用“写临时文件 + rename”原子替换，并在读取不存在文件时返回 nil 以支持冷启动。
- `internal/api/bindings/ai.go`：`AIAPI` 新增 `sessionDir`（默认 `data/sessions`）与 `SetSessionDir(dir)`；`NewAIAPI` 启动时确保目录存在；`getOrCreateSessionMemory()` 新建内存前尝试从 `{sessionDir}/{sessionID}.json` 恢复；`runAgentAnalysis()` 完成后自动保存会话记忆到磁盘。

### 文档更新
- 更新 `AGENTS.md`：同步 SessionMemory 压缩能力已从占位 no-op 升级为基于阈值的摘要压缩实现。

## [1.0.13] - 2026-04-01

### 修复问题
- 修复 SpecialistAgent 在执行分析时未将 `SpecialistRole.Instructions` 注入系统提示词的问题；现在 specialist 角色指令会作为系统提示前缀参与推理。
- 修复 `internal/api/bindings/ai.go` 中会话记忆满容量时按 `map range` 随机淘汰的问题；现在改为基于最近访问时间执行 LRU 淘汰，确保优先移除最久未访问的会话记忆。

### 代码改进
- `internal/ai/agent.go`：为 `Agent` 增加 `systemPromptOverride` 字段，并在 `Analyze()` 中优先拼接该覆盖提示词后再使用基础 `agentSystemPrompt`。
- `internal/ai/specialist.go`：`SpecialistAgent.Analyze()` 创建临时 Agent 时写入 `systemPromptOverride = s.role.Instructions`，确保 header/body/security/general 专家角色指令生效。
- `internal/api/bindings/ai.go`：`AIAPI` 新增 `sessionAccessTimes map[string]time.Time`，并在 `NewAIAPI` 初始化。
- `internal/api/bindings/ai.go`：`getOrCreateSessionMemory()` 在命中与新建路径统一更新时间戳，容量达到上限时按最旧访问时间淘汰并同时清理 `sessionMemories`/`sessionAccessTimes`，保证并发访问下一致性。

## [1.0.12] - 2026-03-31

### 新增功能
- 新增最小可用的 WebSocket 抓包展示能力：HTTP、HTTPS MITM 与 SOCKS5-HTTP 场景下识别 WebSocket 握手，并将双向消息字节流作为 `websocket_frames` 记录到请求详情中。
- 请求详情新增可用的 `WebSocket` 顶部标签页，当请求为 WebSocket 且存在帧数据时，按方向/类型/大小/时间/消息展示捕获结果。

### 修复问题
- 修复右侧 AI 面板中“右键字段分析”和“底部继续追问”使用不同 `session_id` 导致上下文割裂的问题；现在同一面板内所有 AI 交互统一写入一个稳定 chat session。
- 修复 AI 思考区仅展示占位文本而非模型真实 reasoning 的问题；智谱 GLM 支持思考模式的模型现在会解析并流式展示官方 `reasoning_content`。
- 修复 AI 自由问答在追问场景下不会带上前序问答上下文的问题；现在底部聊天框会复用稳定 chat session，并把会话内最近问答传入模型，支持连续追问。
- 修复 AI 自由问答只在完成后一次性显示结果的问题；现在 provider 支持流式时会增量推送内容到前端，边生成边显示。
- 修复 AI 自由问答看不到思考/动作过程的问题；现在 AI 面板支持展示可折叠的思考过程区块。
- 修复 AI 自由问答模式前端已允许发送、但后端仍沿用 request-aware Agent 路径导致报错 `request_id is required` 的问题；现在无选中请求时会走 direct free-chat 路径。
- 修复 AI 分析栏底部输入区在存在选中请求时仍偷偷注入 request-aware tracing 逻辑、导致普通问题触发 `trace_value_flow` 等工具调用的问题；现在该输入区始终为自由对话入口。
- 修复 AI 模型配置弹窗中动态刷新后的可用模型只停留在前端临时状态、点击 OK 后不会写回本地 `configs/models.json` 且分析栏仍使用旧模型列表的问题。
- 修复 AI 模型配置弹窗重新打开时无法复用本地已保存 provider API Key / Base URL 的问题；认证过的 provider 配置现在会保存在本地并在下次配置时自动复用。
- 修复 `Proxy -> SSL Proxying Settings` 中关闭 `Enable SSL Proxying` 后，代理仍继续执行 HTTPS MITM 解密抓包的问题；现在 SSL Proxying 开关会真实控制 HTTPS 解密能力。
- 修复请求详情中请求侧在无 Query String / Cookies / Text / Hex 内容时仍强制显示空标签的问题。
- 修复请求详情中响应侧在无 Text / Hex 内容时仍强制显示空标签的问题。
- 修复请求树中 WebSocket 请求只能依赖 scheme 推断、无法根据真实抓包结果稳定显示 WebSocket 图标的问题。

### 代码改进
- `internal/storage/models.go` 与 `internal/api/bindings/dto.go`：新增 `is_websocket`、`websocket_frames` 与对应 DTO/传输序列化。
- `internal/proxy/proxy.go`：为 HTTP、HTTPS MITM、SOCKS5-HTTP 增加 WebSocket 握手识别、握手转发和双向帧捕获；同时补齐普通 HTTP 请求的 Cookie 落库。
- `internal/proxy/proxy.go`：HTTPS MITM 运行态判定改为同时受 `listener.mitm_enabled` 与 `ssl_proxying.enabled` 控制，避免 UI 的 SSL Proxying 开关失效。
- `internal/config/config.go` 与 `internal/api/bindings/config.go`：新增动态模型合并持久化能力，`UpdateAIConfig` 可在保存活动模型前把动态发现的 provider 模型写入 `models.json`，并新增按 provider 读取本地已保存 API Key 的绑定接口。
- `web/src/components/ModelConfigModal.vue` 与 `web/src/api/wails.ts`：模型配置弹窗现在会复用本地已保存 provider 配置，并在保存时一并持久化动态发现模型；分析栏刷新后可立即使用新模型列表与活动模型。
- `web/src/components/AIPanel.vue` 与 `web/src/stores/aiStore.ts`：AI 面板底部输入区现在始终走自由对话路径；request-aware 分析保留给显式字段/请求分析入口。
- `internal/api/bindings/ai.go`：当 `query` 存在但 `request_id` 为空时，改为走 direct free-chat completion 路径，而不是继续进入 request-aware tracing agent。
- `internal/api/bindings/ai.go`、`internal/ai/types.go` 与 `internal/ai/zhipu.go`：自由问答路径现在支持复用 session memory、发出 thought 事件，并在智谱 GLM 支持思考模式的模型上解析/流式转发官方 `reasoning_content`。
- `web/src/stores/aiStore.ts`：右侧 AI 面板的字段分析与自由聊天现在统一复用同一个 `chatSessionId`；清空面板时也会重置该会话，避免旧记忆残留。
- `web/src/stores/aiStore.ts` 与 `web/src/components/AIPanel.vue`：底部 AI 聊天引入稳定 chat session，支持连续追问上下文与可折叠的 thought/action/observation 展示。
- `web/src/components/RequestDetail.vue`：将顶部标签改为按请求能力动态显示，并实现 WebSocket 帧表格视图。
- `web/src/components/RequestSection.vue` 与 `web/src/components/ResponseSection.vue`：改为按实际数据动态生成底部内容标签，保留现有 Tree/HTML/Image/JSON Text 逻辑。
- `web/src/components/RequestList.vue` 与 `web/src/types/index.ts`：新增 `is_websocket` / `websocket_frames` 前端类型与图标判定支持。

## [1.0.11] - 2026-03-30

### 新增功能
- 后端 DTO 层支持 Brotli (br) 内容编码解压，使用 `github.com/andybalholm/brotli` 库
- 后端 DTO 层支持 charset 感知的文本转码：从 Content-Type 提取 charset 参数，通过 `golang.org/x/text/encoding/htmlindex` 将 GBK/GB2312/ISO-8859-1 等非 UTF-8 文本自动转码为 UTF-8 再传输给前端
- 修复 base64 后备行为：当解压成功但文本仍不可读时，base64 编码使用解压后的解码字节而非原始压缩字节

### 代码改进
- `internal/api/bindings/dto.go` 新增 `decodeBrotli`、`extractCharset`、`transcodeToUTF8` 函数
- `internal/api/bindings/dto_test.go` 新增 Brotli HTML/JSON、GBK/GB2312/ISO-8859-1 转码、不支持的 charset 后备、二进制安全、Brotli+GBK 组合等测试用例
- 新增依赖 `github.com/andybalholm/brotli v1.2.0`

## [1.0.10] - 2026-03-30

### 新增功能
- 响应内容视图新增 HTML 预览标签（HTML），使用 sandboxed iframe 安全渲染解码后的 HTML 内容
- 响应内容视图新增 JSON Text 标签（JSON Text），专门展示格式化后的 JSON；原有 Text 标签恢复为展示未经美化的原始解码文本
- 新增字符集感知的前端解码后备支持（解析 Content-Type 中的 charset）

### 代码改进
- web/src/components/ResponseSection.vue 新增 HTML 与 JSON Text 标签页及沙箱 iframe 渲染
- web/src/components/bodyTreeUtils.ts 新增 isHTMLContentType 与 extractCharset 判断，支持可选 charset 传参，将原始解码与 JSON 美化拆分为 decodeBodyAsync 和 formatBodyForTextAsync

## [1.0.9] - 2026-03-30

### 修复问题
- 修复 `internal/proxy/proxy.go` 中 13 处仅打印日志但未落库存储的失败路径，覆盖访问拒绝、CONNECT hijack 失败、MITM 握手/读包/上游失败、SOCKS5 握手/请求失败与 SOCKS5-HTTP 转发失败等场景，避免失败连接静默丢失。

### 代码改进
- `internal/proxy/proxy.go` 新增 `recordMinimalError(...)` 辅助方法，统一最小错误请求记录的构造与保存，减少重复代码。
- 为 `handleHTTP` 已有的 2 条错误记录路径（`NewRequestWithContext` 失败、`client.Do` 失败）补齐 `Request.Error` 字段。
- 为 MITM 请求上游失败路径补齐错误落库（`StatusCode=502`、`Error`、`Duration`），保证与其他失败路径行为一致。

### 文档更新
- 更新 `AGENTS.md`，补充代理错误路径的最小化落库约定与 `recordMinimalError` 复用要求。

## [1.0.8] - 2026-03-30

### 新增功能
- 优化顶部 Proxy 菜单：移除与顶部状态栏重复的 Recording 切换项；当外部代理启用时，`External Proxy Settings...` 菜单项本身会显示勾选状态，避免新增冗余子菜单项。

### 代码改进
- `web/src/components/ApplicationMenu.vue`：移除 `proxy-recording` 菜单项与 `recordingEnabled` prop，保留 `externalProxyEnabled` 状态透传，并直接将 `External Proxy Settings...` 渲染为可勾选菜单项。
- `web/src/App.vue`：向 ApplicationMenu 透传外部代理开关状态，移除多余的 `proxy:toggle-external-proxy` 菜单 action 分支。

## [1.0.7] - 2026-03-26

### 新增功能
- 优化请求树视图设计：当文件夹下仅包含单一请求时，自动将其与请求节点合并为直接叶子节点（类似 Charles 的单请求合并），减少无意义的多层折叠。
- 强化异常请求的视觉反馈：针对失败/Aborted 的请求（status=0），应用红色停止图标及置灰文本，使其在一大堆请求中像 Charles 一样极具辨识度。

### 代码改进
- `web/src/components/RequestList.vue`：增强 `compressSingleChildFolders` 逻辑，支持将单请求子节点直接上卷压缩为带完整路径标签的叶子节点，并正确保留节点的选中和高亮 Key 行为。
- `web/src/components/RequestList.vue`：引入 `CloseCircleFilled` 图标并添加 `statusFailed` 和 `treeRowFailed` 等 CSS 样式，完善异常请求的渲染。

## [1.0.6] - 2026-03-26

### 新增功能
- 优化请求树视图设计（Charles 风格）：不再将 Query String 拆分为独立的子文件夹节点，而是直接将其附加到对应的请求叶子节点路径上，减少树层级并提升信息密度。
- 强化请求树节点图标体系：新增 html、js、css、json、image、font 等资源的专属图标与颜色区分，替代原先单一的文件图标。

### 代码改进
- `web/src/components/RequestList.vue` 与 `web/src/stores/requestStore.ts`：修改 `splitPathSegments`，查询参数不再生成独立 segment 而是附加在末尾。
- `web/src/components/RequestList.vue`：引入并应用更多 Ant Design Vue 的文件类型图标，细化 `getRequestIconType`，根据 Content-Type 和扩展名进行精准图标匹配。
- 修复 `web/src/components/bodyTreeUtils.ts` 中的 TypeScript 编译报错问题（重复定义及 `ArrayBufferLike` 泛型匹配）。

## [1.0.5] - 2026-03-26

### 修复问题
- 修复抓包详情响应/请求 Text 视图中对 gzip/deflate/br 压缩格式或 Base64 编码的文本响应展示为 Base64 字符串的问题，现已使用 DecompressionStream 异步解压并呈现可读原文。

### 代码改进
- `web/src/components/bodyTreeUtils.ts`：将 `decodeBody` 重构为异步方法 `decodeBodyAsync`，基于 `DecompressionStream` 新增对 `Content-Encoding` 压缩流的解压支持（gzip、deflate、br等）
- `web/src/components/RequestSection.vue` 与 `web/src/components/ResponseSection.vue`：引入 `watch` 处理异步 Body 解析结果，支持异步解析后自动更新界面状态

## [1.0.4] - 2026-03-26

### 新增功能
- 代理监听新增运行时热重载能力：桌面设置更新 `http_port` / `https_port` / `socks5_port` 后，无需重启应用即可切换到新端口

### 修复问题
- 修复代理监听端口修改后仅写入配置但运行监听仍停留旧端口的问题
- 修复 HTTP/HTTPS 共用单 listener 架构下端口可被设置为不一致但运行行为不确定的问题；现在启用两者时会强制校验同端口
- 修复响应体在已知文本类型（`text/html`、`application/javascript` 等）下仍显示 Base64 编码文本而非原始内容的问题

### 代码改进
- `internal/proxy/proxy.go`：
  - `ApplyDesktopSettings` 改为返回错误并在运行态触发 listener 热切换，失败时回滚运行态 desktop 设置
  - `Start` 改为基于 desktop runtime settings 预绑定 listener，HTTP/HTTPS 使用 `Serve(listener)`，SOCKS5 改为可复用的 `serveSOCKS5(listener, port)`
  - 新增 `httpListener` 字段与 `restartListenersIfNeeded()`，实现新 listener 预绑定、锁内原子交换、锁外优雅关闭旧 server/listener
  - `Stop` 调整为先摘除运行态引用再在锁外 shutdown/close，避免与 handler 读锁竞争风险
- `internal/api/bindings/config.go`：`UpdateDesktopSettings` 调整为先 apply proxy 再持久化；持久化失败时 best-effort 回滚 proxy 运行态设置，并向前端返回可读错误
- 新增/扩展 `internal/proxy/proxy_test.go` 热重载回归测试：HTTP 端口切换、端口冲突回滚、慢请求切换期间 drain、SOCKS5 端口切换/冲突、HTTP+HTTPS 同端口校验
- `web/src/components/bodyTreeUtils.ts`：优化文本类型响应的 Base64 解码逻辑，对已知文本类型（`text/*`、`application/javascript` 等）直接尝试解码，不再依赖启发式字节检查

### 文档更新
- 更新 `AGENTS.md`：补充代理端口热重载约定、HTTP/HTTPS 同 listener 端口一致性约束、listener 热切换锁外 shutdown 约束，以及 `ConfigAPI.UpdateDesktopSettings` 的 apply-then-persist/失败回滚约定

## [1.0.3] - 2026-03-20

### 文档更新
- 基于当前代码实现重写 `README.md`，移除旧的 Gin / React / SQLite 架构描述，改为 Wails + Vue 3 + In-Memory + Agent-only 现状说明
- 更新 `docs/PROJECT_OUTLINE.md`、`docs/main/架构设计.md`、`docs/main/技术选型.md`，统一当前桌面运行时、绑定层、内存存储与 AI 编排架构描述
- 更新 `docs/modules/代理模块.md`、`存储模块.md`、`AI分析模块.md`、`前端UI模块.md`、`API模块.md`，移除与现状不符的旧设计内容并同步当前模块边界

## [1.0.2] - 2026-03-20

### 新增功能
- 完成 Charles 风格顶部菜单的二级子菜单交互，当前已支持 `File -> Open Recent`、`View -> Focused Hosts`、`Proxy -> Windows/macOS Proxy` 与 `Tools -> Auto Save` 等 flyout 子菜单

### 修复问题
- 修复 `web/src/App.vue` 中 `Proxy -> Web Interface Settings...` 错误打开通用 Proxy 设置的问题；现在会打开专用 `web-interface` 设置弹窗

### 修复问题
- 修复 `web/src/components/DesktopSettingsModal.vue` 在首屏挂载阶段对 Vue props 直接执行 `structuredClone` 导致的 `DataCloneError`，避免桌面端启动后出现灰白空壳界面
- 修复 `web/src/components/AIPanel.vue` 漏导入 `StopOutlined` 图标导致的 Vue unresolved component warning

### 代码改进
- 扩展 `web/src/components/ApplicationMenu.vue` 支持基于 `children` 的二级子菜单模型，并接入 recent sessions、focused hosts、系统代理与 auto save 等动态菜单数据
- 扩展 `web/src/App.vue` 与 `web/src/components/RequestList.vue` 的菜单动作路由，补齐结构/顺序视图切换、状态栏显隐、主机聚焦、About 弹窗、Web Interface 设置入口等菜单行为

### 文档更新
- 更新 `AGENTS.md`，补充桌面设置弹窗初始化需先转为普通对象再克隆的前端约定
- 更新 `AGENTS.md`，补充顶部菜单二级子菜单与 Web Interface 设置入口约定

## [1.0.1] - 2026-03-19

### 新增功能
- 桌面无边框标题栏新增 Charles 风格顶部应用菜单栏，提供 `File / Edit / View / Proxy / Tools / Window / Help` 父菜单结构

### 修复问题
- 修复请求详情 Response Text 视图在文本响应场景下直接显示 Base64 串的问题；现在会优先按文本 MIME 类型解码并显示原文

- 请求详情 Response 区域在图片响应（`image/*`）场景下新增 `Image` 标签，可直接预览响应图片

### 代码改进
- 新增 `web/src/components/ApplicationMenu.vue`，在现有 Wails frameless 标题栏中实现 Charles 风格菜单交互与下拉层
- `web/src/App.vue` 接入顶部菜单，并将当前已存在能力映射到菜单动作：新建 Session、清空请求、导出选中请求（cURL/HAR 复制到剪贴板）、刷新请求、展开/折叠请求树、代理启停、请求重放、窗口控制与 About
- `web/src/components/RequestList.vue` 新增基于 `packetmind:request-list-command` 的外部展开/折叠命令监听，使顶部 View 菜单可复用现有请求树能力

### 文档更新
- 更新 `AGENTS.md`，补充顶部应用菜单与响应正文显示约定

## [1.0.0] - 2026-03-18

### 重大变更 (Breaking Changes)
- **Wails-only 架构**: 项目已从双模式 (Gin HTTP + Wails) 迁移为纯 Wails 桌面应用
  - 移除了 Gin HTTP 服务器 (`cmd/server/main.go`)
  - 移除了 Gin HTTP handlers (`internal/api/handlers/`)
  - 移除了 WebSocket Hub (`internal/api/ws/`)
  - 前端现在直接使用 Wails bindings 替代 HTTP API
  - 前端现在使用 Wails Events 替代 WebSocket

### 新增功能
- 添加了有效的 Windows 应用图标 (`build/windows/icon.ico`) - 蓝底白字 "PM" 设计

### 代码改进
- 前端 API 层统一为 Wails bindings (`web/src/api/wails.ts`)
- 前端事件处理统一为 Wails Events (`web/src/composables/useWailsEvents.ts`)
- 移除了 Vite proxy 配置（不再需要代理到后端）
- 新增 Wails-facing DTO 层（`internal/api/bindings/dto.go`），将绑定返回中的 `time.Time` 转为字符串时间字段，消除 `wails dev` 的 `Not found: time.Time` 绑定生成警告
- `app.go` 移除对 `App` 自身的 Wails 绑定暴露，仅绑定真正面向前端的 API；避免内部事件桥接方法把 `storage.Request` 再次带入绑定生成
- `app.go` 新增有效的 `AssetServer` 配置并嵌入 `web/dist`，修复运行时 `AssetServer options invalid` 错误
- 修复 `ModelConfigModal.vue` 等前端残留导入/响应处理，彻底对齐 Wails-only API 形态
- 启用 Wails 无边框窗口（`Frameless: true`），并在 `App.vue` 中实现基于 `--wails-draggable` 的自定义标题栏拖拽区域
- 新增前端窗口控制按钮（最小化 / 最大化 / 关闭）与最大化状态切换；Windows/Linux 使用自定义按钮，macOS 保留原生 traffic lights
- 统一前端对 Wails runtime 的动态加载方式，消除 runtime 模块静态/动态混用导致的构建 warning
- 打磨无边框桌面壳层体验：优化标题栏与窗口按钮 hover/active 反馈，补充最大化态视觉处理，并新增右下角 resize grip 视觉提示（仍使用 Wails/OS 原生边缘缩放）
- 进一步将右上角窗口按钮调整为更接近 Windows 原生 caption buttons 的比例、线宽、对齐与 hover/active 反馈，减少 Web 组件感
- 将无边框窗口右上角三个 caption icons 统一为同一 10×10 几何坐标系下的纯 CSS 图标，修复大小不一致与基线不齐问题
- 重写 `web/src/api/wails.ts`，改为直接调用 Wails 生成的 `wailsjs/go/bindings/*` 接口并做轻量响应归一化，修复运行时误报 `Wails binding not available`
- 启动窗口改为优先按主屏幕尺寸动态计算大小和居中位置；若启动阶段无法获取屏幕信息，则回退为更大的专业工作窗口尺寸并居中，避免默认窗口过小
- 进一步降低整体前端 Web 感：统一零圆角/低圆角控件、更强分隔线、更高信息密度、更克制的灰阶桌面工具壳层风格
- 请求详情区（尤其 Contents/Headers/Overview/SSL 的 key-value 区域）改为随中间面板宽度自适应：宽时双列，窄时自动切换为上下堆叠，避免值被挤到最右侧难以阅读

### 移除功能
- 移除独立的 Gin HTTP 服务器入口
- 移除 WebSocket 实时通信（由 Wails Events 替代）
- 移除 HTTP API 层 (`web/src/api/index.ts`)
- 移除 WebSocket composable (`web/src/composables/useWebSocket.ts`)

---

## [0.9.0] - 2026-03-16

### 新增功能
- 完成 Wails v2 桌面端迁移基础架构
  - 新增 `wails.json` 配置文件
  - 新增 `app.go` 主入口，支持 Wails 应用启动
  - 新增 `build/` 资源目录（appicon.png, windows/icon.ico, darwin/icon.icns）
  - 更新 `go.mod` 添加 Wails v2.9.0 依赖

### Go Bindings 层
- 新增 `internal/api/bindings/types.go` - 统一响应结构 `SessionResponse`
- 新增 `internal/api/bindings/session.go` - Session CRUD 绑定
- 新增 `internal/api/bindings/request.go` - Request 列表/导出/重放绑定
- 新增 `internal/api/bindings/config.go` - 配置读取/AI 配置更新绑定
- 新增 `internal/api/bindings/proxy.go` - 代理状态/启动/停止绑定
- 新增 `internal/api/bindings/ai.go` - AI 分析（支持流式事件）绑定

### 前端迁移
- 新增 `web/src/composables/useWailsEvents.ts` - Wails 事件处理（替代 WebSocket）
  - 订阅 `request:new` 事件 → requestStore.addRequest
  - 订阅 `ai:analysis` 事件 → aiStore 处理 agent_event/agent_result/done
- 新增 `web/src/api/wails.ts` - Wails API 封装层
  - 封装 sessionApi、requestApi、aiApi、configApi、proxyApi
  - 与现有 `api/index.ts` 接口兼容
- 新增 `web/src/utils/wails.ts` - Wails 运行时工具函数
  - `copyToClipboard` - 使用 `ClipboardSetText` 替代 `navigator.clipboard`
  - `getWindowSize` - 使用 `WindowGetSize` 替代 `window.innerWidth`
  - `onWindowResize` - 使用 Wails 事件替代 `window.addEventListener('resize')`

### CI/CD
- 新增 `.github/workflows/wails-build.yml` - GitHub Actions 构建工作流
  - 支持 Windows/macOS/Linux 三平台构建
  - 自动发布到 GitHub Releases

### 架构变更
- 事件系统迁移：WebSocket → Wails Events
  - `new_request` → `request:new`
  - `ai_analysis` → `ai:analysis`
- API 调用迁移：axios HTTP → Wails Go bindings IPC

## [0.8.1] - 2026-03-16

### 新增功能
- 新增 Wails Session 绑定：`internal/api/bindings/session.go`
  - 提供 `ListSessions`、`GetSession`、`CreateSession`、`UpdateSession`、`DeleteSession`、`ActivateSession` 方法
  - 新增 `CreateSessionRequest`、`UpdateSessionRequest`、`PaginatedSessions` 类型
- 新增 Wails Request 绑定：`internal/api/bindings/request.go`
  - 提供 `ListRequests`、`GetRequest`、`DeleteRequest`、`ClearRequests`、`ExportRequest`、`ReplayRequest` 方法
  - 新增 `RequestListOptions`、`PaginatedRequests`、`ReplayRequestOptions`、`ReplayResult` 类型
- 新增共享响应类型文件：`internal/api/bindings/types.go`
  - 定义统一响应结构 `SessionResponse`（`code` / `message` / `data`）

### 代码改进
- Session/Request 绑定对齐既有 Gin handlers 的错误码与响应契约（成功 `code=0`；错误码沿用 `40001`、`40002`、`50001`）
- Request 绑定补齐 HAR/cURL 导出与请求重放能力，并沿用现有重放结果结构（`request_id`、`status_code`、`duration_ms`、`response_size`、`response_header`）

### 文档更新
- 更新 `AGENTS.md`：补充 Wails 绑定层 Session/Request 的契约对齐约定

## [0.8.1] - 2026-03-16

### 新增功能
- 新增 Wails v2 项目基础骨架：根目录 `wails.json`、`app.go` 与 `build/` 资源目录，作为 Gin+Vue 向桌面端迁移的第一步。
- 新增 Wails AI 绑定 `internal/api/bindings/ai.go`，提供 `Analyze`、`CancelAnalysis`、`ListModels`、`FetchModels` 前端调用入口。
- 新增 AI 绑定请求/响应结构 `AnalyzeRequest`、`AnalyzeResponse`、`FetchModelsRequest`，对齐现有 HTTP AI API 语义。

### 代码改进
- `app.go` 新增 Wails 启动入口，完成配置加载、内存存储初始化、代理启动与 `request:new` 事件广播（`EmitRequest`）。
- 根目录新增 `wails.json`，前端命令链路保持指向现有 `web/`（不改动原 SPA 目录）。
- 新增 `build/appicon.png`、`build/windows/icon.ico`、`build/darwin/icon.icns` 占位说明文件，后续可替换为真实图标资源。
- `go.mod` 引入 `github.com/wailsapp/wails/v2 v2.9.0` 并完成模块整理。
- `internal/api/bindings/ai.go` 新增基于 Wails Events 的流式广播封装（统一事件名 `ai:analysis`），支持 `agent_event` / `agent_result` / 错误与取消事件。
- AI 绑定复用 AgentOrchestrator + SessionMemory + 可取消上下文机制，`Analyze` 改为异步 goroutine 执行，避免阻塞主线程。
- 修复 `internal/ai/openai.go` 中 `Complete()` 的异常占位返回，恢复 OpenAI 正常响应解析与错误处理，确保工程可编译通过。

### 文档更新
- 更新 `AGENTS.md`：补充 Wails 基础入口与构建资源目录约定。
- 更新 `AGENTS.md`：补充 Wails 绑定目录结构与 AI 绑定事件约定。

## [0.8.0] - 2026-03-16

### 代码改进
- 新增 Wails 后端绑定 `internal/api/bindings/config.go`：提供 `ConfigAPI`（`GetConfig`/`UpdateAIConfig`/`GetCertInfo`/`DownloadCert`）并复用 `SessionResponse` 统一返回结构
- `ConfigAPI.GetConfig` 仅返回脱敏配置（不暴露 API Key 与 SOCKS5 密码）；`DownloadCert` 改为返回 Base64 编码证书内容，便于桌面端直接落盘
- 新增 Wails 后端绑定 `internal/api/bindings/proxy.go`：提供 `ProxyAPI`（`Status`/`Start`/`Stop`）并复用与 HTTP handler 一致的错误码语义（`50003`）

### 文档更新
- 新增 `docs/migration/wails-migration.md`：完整的 Wails v2 迁移指南文档 (~2800 行)
  - Executive Summary：迁移动机、收益评估、工时估算
  - Architecture Comparison：当前 Gin+SPA vs 目标 Wails 架构对比 (ASCII 图)
  - Prerequisites：Go/Wails/平台依赖、安装验证命令
  - Step-by-Step Migration Guide：
    - 项目初始化 (wails.json, app.go)
    - Go Bindings：Session/Request/AI/Config/Proxy 完整绑定代码示例
    - WebSocket → Events 迁移：事件名映射、后端广播、前端订阅
    - Frontend Store 适配：API 封装层、Store 导入更新
    - Browser API 替换：Clipboard/Window/Document/localStorage
  - Build Configuration：完整 wails.json、Vite 配置、跨平台构建命令、GitHub Actions CI/CD
  - Testing Strategy：Go 绑定测试、前端测试、手动测试清单
  - Known Limitations and Workarounds：WebSocket 移除、代理端口、文件下载等
  - Timeline and Milestones：3 周迁移计划、周里程碑、风险缓解

## [0.7.8] - 2026-03-13

### 新增功能
- 新增动态模型发现接口 `POST /api/ai/models/fetch`，支持从 Zhipu/OpenAI API 宽时拉取可用模型列表
- 新增 `ModelLister` 接口与 `ModelInfo` 类型，统一 Provider 模型发现契约
- 新增 `ZhipuClient.ListModels()` 与 `OpenAIClient.ListModels()` 方法实现
- 前端 `ModelConfigModal` 点击 "Initialize Models" 按钮时调用 `/api/ai/models/fetch` 动态获取模型列表

### 修复问题
- 修复 `SessionList.vue` 右键菜单点击后立即消失的问题（添加 `event.stopPropagation()`）

### 代码改进
- 前端新增 `fetchDynamicModels()` API 函数
- 删除重复的 `internal/ai/provider.go`（类型已统一维护在 `types.go`）
- 清理 `internal/ai/zhipu.go` 中的重复 `ListModels` 与损坏的 `Stream` 方法实现

### 测试更新
- 新增 `TestZhipuClient_ListModels` 单元测试
- 新增 `TestOpenAIClient_ListModels` 单元测试
- 新增 `TestAIHandler_FetchModels_MockProvider` 处理器测试
- 新增 Playwright E2E 测试骨架（session-context-menu.spec.ts、model-fetch.spec.ts）

### 文档更新
- 更新 `AGENTS.md`：补充 `ModelLister` 接口与动态模型发现端点说明

## [0.7.6] - 2026-03-13

### 代码改进
- 调整前端 `web/src/components/AIPanel.vue` 与 `web/src/stores/aiStore.ts` 的 AI 时间线渲染逻辑：Agent reasoning / action / observation / decision 改为直接并入主消息流按到达顺序展示
- AI 分析完成后，`Related Requests` 与 `Provenance Chain` 不再作为结论下方的独立收尾区块出现，而是作为时间线消息插入在最终 assistant 结论之前，减少视觉突兀感
- `agent_related` 与 `agent_provenance` 时间线项保留请求跳转能力，并复用既有 provenance 展示信息
- `web/src/stores/requestStore.ts` 新增 `ensureRequestLoaded()` 与 `selectRequestById()`，使 AI 面板中点击 related requests / provenance request id 时，即使目标请求不在当前已加载列表中，也能先按 id 拉取详情再切换中间详情视图
- 改进 `web/src/components/AIPanel.vue` 中 `Related Requests` 与 `Provenance Chain` 请求 ID 的样式：
  - 使用等宽字体（monospace）增强可识别性
  - 添加渐变背景与边框，形成更明显的按钮/导航项外观
  - 悬停时添加高亮效果（背景、边框），提升可点击感知

### 文档更新
- 更新 `AGENTS.md`，补充 AI 面板中 reasoning / related requests / provenance chain 应并入同一条流式时间线展示的约定
- 更新 `AGENTS.md`，补充 AI 面板请求跳转需保证目标请求详情已载入 store 的约定

## [0.7.5] - 2026-03-13

### 代码改进
- `internal/ai/interfaces.go` 新增 `AnalyzerInterface`（`Analyze` / `StreamAnalyze`），统一抽象 legacy 分析能力契约
- `internal/ai/interfaces.go` 新增编译期断言 `var _ AnalyzerInterface = (*Analyzer)(nil)`，确保 `Analyzer` 持续满足接口约束

### 文档更新
- 更新 `AGENTS.md`，补充 Analyzer 与 Agent 并存现状、当前不可直接替换结论，以及未来下线 Analyzer 的推荐迁移路径

## [0.7.4] - 2026-03-13

### 代码改进
- 新增 `internal/ai/utils.go`，集中维护 Agent/Analyzer 共享辅助函数：参数解析、字符串/内容处理、HTTP 预览、JSON 序列化、工具切片克隆、格式化与字段提取
- `internal/ai/agent.go` 移除已下沉到 `utils.go` 的通用 helper 定义，仅保留 Agent 核心流程与特定逻辑
- `internal/ai/analyzer.go` 移除格式化与字段提取 helper 定义，统一复用 `utils.go` 实现
- `internal/ai/orchestrator.go` 移除重复的 `containsAnyFold` 实现，改为复用共享工具函数

### 文档更新
- 更新 `AGENTS.md`，补充 AI 共享辅助函数应统一维护在 `internal/ai/utils.go` 的约定

## [0.7.3] - 2026-03-13

### 代码改进
- 新增 `internal/ai/analyzer_test.go`，为 Analyzer 非 Agent 模式补充单元测试覆盖：`NewAnalyzer`、`Analyze`、`StreamAnalyze`、`collectFieldHistory`、`extractFieldValue`、`buildPrompt`、`mockResponse`
- 新增参数提取辅助函数测试：`extractQueryParam`、`extractCookie`、`extractFromBody`，覆盖 JSON/Form 提取、缺失字段、空输入、异常输入等边界场景
- 新增 Analyzer 错误路径测试：`Analyze` 异步错误输出（storage nil / request missing）、`StreamAnalyze` 同步错误返回（storage nil / request missing）

### 文档更新
- 更新 `AGENTS.md` 测试状态说明，补充 `internal/ai/analyzer_test.go` 的覆盖范围

## [0.7.2] - 2026-03-13

### 新增功能
- 新增 provider 层单元测试：`internal/ai/zhipu_test.go` 与 `internal/ai/openai_test.go`，覆盖构造函数、`Complete`、`Chat` 与（Zhipu）`Stream` 的核心行为

### 代码改进
- `internal/ai/zhipu_test.go` 使用 `httptest.Server` 模拟 Zhipu API 与 SSE 流，覆盖成功响应、HTTP 非 200、API error 对象、无 choices、非法 JSON 与网络失败场景
- `internal/ai/openai_test.go` 使用 `httptest.Server` 模拟 OpenAI API，覆盖 `Complete`/`Chat` 的成功与错误路径，并补充 `Stream` 方法未实现时的兼容性校验测试
- `internal/ai/tools.go` 新增内置工具定义集中管理，统一维护工具名称常量与 `builtInTools()`，降低工具名称硬编码与分散维护成本
- `internal/ai/registry.go` 移除内置工具定义实现，仅保留 ToolRegistry 与默认注册逻辑，通过 `builtInTools()` 装载默认工具
- `internal/ai/agent.go` 工具分发分支改为复用工具名称常量，避免注册名与执行分支字符串漂移

### 文档更新
- 更新 `AGENTS.md` 测试现状说明，补充 AI provider 测试覆盖范围
- 更新 `AGENTS.md`，补充内置工具定义集中维护在 `internal/ai/tools.go` 的约定

## [0.7.1] - 2026-03-13

### 新增功能
- AI Agent 阶段三新增最小 MCP 集成：`internal/ai/mcp.go` 提供 `MCPClient` 接口、`MCPToolAdapter` 适配器与 `MCPManager` 管理器，支持将 MCP 服务端的工具注册到 ToolRegistry

### 代码改进
- 新增 `internal/ai/interfaces.go`，集中维护 Agent 对存储层能力断言所需的接口（`requestByIDGetter`、`headerSearcher`、`bodySearcher`、`responseBodySearcher`、`provenanceSourceFinder`、`provenanceUsageFinder`、`provenanceTracer`）
- `internal/ai/agent.go` 移除内联存储能力接口定义，改为复用 `interfaces.go` 中的同名接口，保持原有方法签名与断言行为不变
- `internal/ai/mcp.go` 新增 `MCPToolDefinition` / `MCPToolResult` / `MCPContentBlock` 数据结构，用于表示 MCP 工具定义与执行结果
- `internal/ai/mcp.go` 新增 `MCPClient` 接口：`ListTools(ctx)` / `CallTool(ctx, name, args)` 两个方法，满足最小 MCP 工具发现与调用需求
- `internal/ai/mcp.go` 新增 `MCPToolAdapter`：支持工具名前缀命名（避免冲突）、工具发现缓存、ToolRegistry 注册、委托执行
- `internal/ai/mcp.go` 新增 `MCPManager`：支持多 MCP 适配器管理、统一工具注册、按前缀路由工具调用
- `internal/ai/mcp_test.go` 新增 `mockMCPClient` 与完整测试覆盖：工具发现、前缀注册、委托执行、错误处理、nil 安全
- `internal/ai/agent.go` 新增 `SetMCPManager()/GetMCPManager()` 并在未知工具分支尝试委托给 MCP 管理器执行，使 MCP 工具进入真实 Agent 执行路径
- `internal/api/handlers/ai.go` 当前 agent 流程已切换为 `AnalyzeWithHandoff()`，并在 `agent_result` 中附加 `handoff_count` / `specialist_chain` / `final_specialist` 等阶段三可选元数据
- `internal/api/handlers/ai_test.go` 新增 `TestAIHandler_StageThreeCollaborationMetadata`，验证协作元数据映射兼容性
- `internal/ai/types.go` 新增统一导出类型定义，集中管理 `LLMMessage`、`Tool*`、`LLMChat*`、`Agent*` 等共享结构，减少 `agent.go` 中类型与逻辑耦合
- `internal/ai/agent.go` 移除重复导出类型定义并将 provider 请求消息结构改为直接复用 `LLMMessage`，保持字段与 JSON 标签兼容

### 文档更新
- 更新 `docs/modules/AI分析模块阶段进度记录.md`，记录阶段三完成情况、验证结果与当前边界
- 更新 `AGENTS.md`，同步阶段三 MCP 集成架构说明与使用约定
- 更新 `AGENTS.md`，补充 AI 存储能力断言接口统一维护位置（`internal/ai/interfaces.go`）
- 更新 `AGENTS.md`，补充 AI 公共消息/请求结果类型已抽离至 `internal/ai/types.go` 的约定

## [0.7.0] - 2026-03-13

### 新增功能
- AI Agent 阶段三新增最小可用多专家协作机制：specialist 可通过确定性规则请求 handoff 到其他 specialist
- 新增 `HandoffRequest` 类型，允许 specialist 在分析结果中触发到其他专家的委托请求
- 新增 `CollaborationPlan` 与 `CollaborationResult` 类型，记录协作链路与元数据

### 代码改进
- 新增 `internal/ai/collaboration.go` 与 `internal/ai/collaboration_test.go`，实现：
  - `AnalyzeWithHandoff()` 方法支持带 handoff 的分析流程
  - 确定性 handoff 规则（基于关键词匹配，非 LLM 驱动）
  - 有界深度限制（默认最多 3 次 handoff）
  - 循环防护（已访问的 specialist 不会被重复调用）
  - 简单结果合并与元数据记录
- 新增 `DefaultHandoffRules()` 提供默认 handoff 规则：header→security、body→security、security→body/header
- `CheckHandoffNeeded()` 基于结果内容与已访问集合判断是否需要 handoff
- `CanHandoff()` / `GetValidHandoffTargets()` 辅助函数用于查询合法 handoff 目标
- 现有 `Analyze()` 方法保持不变，handoff 为可选增强能力

### 文档更新
- 更新 `AGENTS.md`，记录阶段三最小协作机制的架构约定与使用方式

## [0.6.9] - 2026-03-13

### 新增功能
- AI Agent 阶段二新增结构化错误模型：`AgentError` 支持 `retryable / recoverable / fatal` 分类，以及 tool/specialist 相关上下文

### 代码改进
- `internal/ai/agent.go` 新增 `safeExecuteTool()`，为 tool 执行提供超时与 panic recovery，并将错误回填为结构化 observation 供模型继续推理
- `internal/ai/specialist.go` 为 agent 事件附加 `specialist_name` 元数据，增强 orchestrator/specialist 层的流式可观测性
- `internal/ai/orchestrator.go` 新增 specialist 失败时 fallback 到 `general` specialist 的恢复逻辑，并在 `OrchestratorResult` 中保留 `FallbackFrom` / `FallbackError`
- `internal/ai/errors_test.go` 新增阶段二错误模型测试，覆盖错误分类、timeout、panic、wrap/unwarp 行为
- `internal/api/handlers/ai.go` 新增 agent-mode goroutine panic recovery，在 panic 时广播带 `done=true` 的错误消息，避免静默失败
- `internal/api/handlers/ai.go` 新增 additive orchestrator metadata 透传：`agent_result` 现包含 `specialist_name`、`routing_reason`、`routing_confidence` 可选字段，同时顶层 `data` 对象新增 `specialist_name` 字段
- `internal/api/handlers/ai_test.go` 新增 `TestAIHandler_OrchestratorResultMetadata`、`TestAIHandler_AdditiveMetadataFields`、`TestAIHandler_PanicRecoveryBroadcast` 测试用例，验证路由行为、元数据透传与 panic 恢复

### 文档更新
- 更新 `docs/modules/AI分析模块阶段进度记录.md`，记录阶段二完成情况、验证结果与阶段三建议
- 更新 `AGENTS.md`，同步阶段二关于结构化错误、tool 执行保护与可选 specialist 元数据的约定

## [0.6.8] - 2026-03-13

### 新增功能
- AI 模块新增 SessionMemory 会话记忆基础实现：支持有界短期记忆存储、线程安全操作、上下文键值对存储、GetRecent/Clear/ToLLMMessages 等核心方法
- AI Agent 阶段一新增最小可用多专家骨架：`AgentOrchestrator` 可基于确定性规则在 `header/body/security/general` specialist 间路由，并共享会话记忆

### 代码改进
- 新增 `internal/ai/memory.go` 与 `internal/ai/memory_test.go`，实现 bounded、thread-safe、nil-safe 的 SessionMemory
- `internal/ai/agent.go` 新增 `memory` 字段与 `SetMemory/GetMemory` 方法，向后兼容地支持可选会话记忆集成
- 新增 `internal/ai/registry.go` 与 `internal/ai/registry_test.go`，将内置工具迁移到可复用的 `ToolRegistry`，支持默认注册表、克隆、移除与并发访问
- 新增 `internal/ai/specialist.go`、`internal/ai/orchestrator.go` 及对应测试，提供 tool whitelist、确定性路由、共享 SessionMemory 与阶段一 handoff 基础设施
- `internal/api/handlers/ai.go` 的 agent 模式分析已切换为通过 `AgentOrchestrator` 执行，同时复用按 `SessionID` 维护的进程内会话记忆而不改变现有 HTTP / WebSocket 返回结构

### 文档更新
- 新增 `docs/modules/AI分析模块阶段进度记录.md`，记录 AI Agent 改进阶段一完成情况、验证结果与后续阶段建议
- 更新 `AGENTS.md`，同步阶段一 AI Agent 架构现状（SessionMemory / ToolRegistry / SpecialistAgent / AgentOrchestrator）

## [0.6.6] - 2026-03-12

### 新增功能
- 请求详情 Contents 视图中的 Request/Response Body 新增条件化 `Tree` 标签页：当正文可解析为 JSON 或 XML 时展示层级树，支持对象/数组、XML 元素/属性/文本节点的紧凑浏览
- AI Agent 新增基于结构化参数提取的 provenance tracing：可从目标请求字段回溯同会话更早响应中的候选来源，并输出带置信度的传播链路

### 修复问题
- 修复抓包列表关键字过滤与实时 WebSocket 新请求不一致的问题：新到达请求现在会遵循当前 `session/host/method/search` 过滤条件，避免污染已过滤视图
- 修复抓包列表搜索框依赖 `InputSearch` 提交事件导致过滤更新不及时的问题，改为输入变化时实时同步过滤
- 修复抓包列表 `Search host/path` 实际会命中更宽泛 URL 字段而展示无关 host 的问题，搜索语义现统一收窄为仅匹配 Host/Path

### 代码改进
- `web/src/stores/requestStore.ts` 新增统一过滤匹配逻辑与实时请求节点高亮状态管理（含定时清理），并在清空请求时释放高亮计时器
- `web/src/components/RequestList.vue` 为 Host/路径/请求节点接入 Charles 风格短时高亮动画，保持现有树层级与选中交互不变
- 抽离并复用正文解码与格式化逻辑到 `web/src/components/bodyTreeUtils.ts`，统一 Text/Tree 视图对 Base64 文本体、JSON 美化与 XML/JSON 自动识别行为
- 新增 `web/src/components/BodyTreeNodeView.vue` 递归树节点组件，保持现有 Charles 风格高密度灰阶视觉，不影响 Text/Hex/Raw 原有交互
- 新增 `internal/storage/provenance.go` 与 `internal/storage/provenance_test.go`，实现 query/header/cookie/json/form 的结构化参数提取、候选来源检索与传播打分
- `internal/ai/agent.go` 新增 `find_prior_response_sources`、`find_later_request_usages`、`trace_value_flow` 三个 provenance 工具，增强 A 响应值 → B 请求值 的跨请求来源分析能力

## [0.6.5] - 2026-03-12

### 修复问题
- 优化 AI Agent 对响应头字段的分析路径，改为由模型先做 direct/trace 路由判断，避免在显然无需跨请求追踪的场景下误进入多轮工具调用
- 修复 Agent 分析请求在 `agent_mode` 下未充分利用 `target_field` / `target_location` / `target_value` 上下文的问题，减少 trivial header 分析长时间无响应现象
- 修复前端 AI 面板在分析进行中缺少明确运行信号的问题，支持停止当前分析并显示排队中的后续提问
- 修复前端 Agent reasoning 面板偶发仅显示容器但不显示推理步骤内容的问题，改为使用稳定的 trace/message ID，并在分析完成后默认折叠保留推理链
- 修复窄视口或长对话内容下 AI 面板底部输入框被消息区域挤出可视区的问题，改为由消息区内部滚动并保持输入区固定在底部

### 代码改进
- `internal/api/handlers/ai.go` 为 Agent 自动构造更聚焦的分析 query，在未提供自定义 query 时优先使用右键分析的字段上下文
- `internal/ai/agent.go` 新增响应头路由决策阶段，由模型输出 `direct` / `trace` 决策、置信度与追踪提示，减少基于字段名的硬编码
- `internal/api/handlers/ai.go` 新增 Agent / Legacy 分析分流、分析取消能力与上下文生命周期管理，前端可通过新接口停止当前分析
- `web/src/stores/aiStore.ts` 与 `web/src/components/AIPanel.vue` 新增 AI 思考过程展示、路由决策展示、分析中停止按钮与输入排队能力
- `web/src/stores/aiStore.ts` 新增稳定序列 ID 与结束时自动折叠 trace 的状态控制，避免 trace 列表因 key 冲突导致渲染异常
- `web/src/components/AIPanel.vue` 调整列式 flex 布局约束，移除固定宽度并补充 `min-height: 0` / `flex-shrink` 控制，确保底部输入区稳定可见
- 新增 `internal/ai/agent_test.go` 与 `internal/api/handlers/ai_test.go` 覆盖响应头路由决策与 Agent query 构造逻辑

## [0.6.4] - 2026-03-12

### 新增功能
- 前端 AI 面板完整集成 Agent 功能：
  - 新增 Agent/Simple 模式切换开关
  - Agent 模式下实时展示推理过程（Thought → Action → Observation 循环）
  - 显示当前推理深度和工具调用次数
  - 展示关联的 Request IDs
  - 支持点击关联请求 ID 跳转（待实现）

### 代码改进
- 重构 `aiStore.ts`，修复代码重复问题，新增 Agent 状态管理
- 修复 `useWebSocket.ts` Agent 事件解析逻辑，正确处理 `agent_event` 字段
- `AIPanel.vue` 新增 Agent 推理链可视化组件

## [0.6.3] - 2026-03-12

### 新增功能
- 新增 `internal/ai/agent.go`，提供基于 ReAct 的请求溯源 Agent，支持多轮工具调用、深度/预算限制、重复调用拦截与最终结论输出
- Agent 新增 `get_request`、按 Host/Header/请求体/响应体检索等工具定义，并统一兼容 Zhipu / OpenAI 的 function calling 请求格式
- Agent 支持通过回调流式输出 thought/action/observation/final 事件，便于后续接入 WebSocket 广播与前端推理链展示

### 代码改进
- 为 `ZhipuClient` 与 `OpenAIClient` 补充统一的 `LLMClient` 对话实现，复用同一套 provider 适配层处理 tool calls 与最终回答
- Agent 结果结构补充 trace chain、工具调用计数、提前停止原因与关联 request_id 聚合，提升可观测性与调试能力

### 文档更新
- `AGENTS.md` 补充 Agent 推理链、统一 LLMClient 封装与流式进度回调约定

## [0.6.2] - 2026-03-12

### 修复问题
- **关键修复**: AI分析API调用因context被取消而失败的问题
  - 原因: HTTP handler使用`c.Request.Context()`传递给后台goroutine，HTTP响应发送后context被取消导致AI API调用中断
  - 修复: 改用`context.Background()`确保AI调用在HTTP响应后继续运行
- AI分析功能调试日志增强：
  - 前端 WebSocket 消息接收日志（`useWebSocket.ts`）
  - 前端 AI Store 分析请求日志（`aiStore.ts`）
  - 后端分析请求处理日志（`ai.go`）

### 文档更新
- 明确 AI 分析功能需要配置 API Key 才能获得真实 AI 分析结果
- 无 API Key 时会返回模拟分析结果（模板内容）

## [0.6.1] - 2026-03-11

### 新增功能
- 请求详情 Overview 视图补齐 Charles 风格基础信息区，新增 URL、状态、协议、连接、SSL、请求/响应尺寸等总览字段
- Overview 中新增 Timing 区块，展示请求开始/结束、响应开始/结束、总耗时、连接耗时、TLS 握手、请求/响应耗时与延迟等时间信息
- 请求详情 SSL 视图补齐 Charles 风格分组信息，新增 Protocol、Session Resumed、Cipher Suite、ALPN、Server Certificates、Extensions 等折叠区块
- HTTPS 请求现会额外记录并暴露 SNI、ALPN、会话复用、Curve、OCSP/SCT 存在状态、服务端证书链与证书扩展摘要，供 SSL 视图直接展示
- 请求详情 Summary 视图补齐 Charles 风格表格摘要，新增 Resource、Host、Code、Mime Type、Header、Body、Time 列以及 Total / Grand Total / Duration 汇总行
- 抓包列表改为 Charles 风格树形结构：按 Host 分组、按路径逐层折叠、查询串作为叶子节点展示，并为 HTTP/HTTPS/WebSocket/Protobuf/SOCKS 等记录显示不同图标
- Overview 视图改进：
  - 改为单列布局，每个字段单独占一行显示
  - Connection、Timing、Size 等有子字段的区块支持折叠/展开
  - 默认折叠状态，点击标题可切换展开状态
  - 使用 Ant Design 图标（RightOutlined/DownOutlined）作为折叠指示器

### 修复问题
- 修复 SOCKS5 代理在协议探测后回退原始隧道时丢失已 Peek 的首包字节，避免 HTTPS 请求因 TLS ClientHello 被截断而无法收到响应
- 修复 SOCKS5 代理访问 HTTPS 时仅记录 CONNECT 的问题，TLS 流量现在会进入 MITM 解密并记录真实的 HTTPS 请求方法与响应状态

### 代码改进
- 优化 `handleSOCKS5HTTP()` 的上游响应读取方式，复用单个 `bufio.Reader` 以减少长连接场景下的缓冲错位风险
- SOCKS5 的 TLS 分支复用现有 HTTPS MITM 逻辑，保证 SOCKS5 与 HTTP 代理对 HTTPS 流量的记录行为一致

### 文档更新
- 同步更新 `AGENTS.md` 中的测试状态，记录 SOCKS5 回归测试已覆盖 HTTPS 首包转发与 MITM 记录场景
- 同步记录请求详情 Overview/Timing 视图已支持 Charles 风格总览信息展示
- 同步记录请求详情 SSL 视图已支持 Charles 风格分组与证书链展示
- 同步记录请求详情 Summary 视图已支持 Charles 风格摘要表格展示
- 同步记录抓包列表已切换为 Charles 风格层级树形展示

## [0.6.0] - 2026-03-10

### 新增功能

#### 请求详情布局重构 (Charles 风格)
- 请求详情区域改为更接近 Charles 的桌面式结构
- 顶部新增请求摘要栏与导航标签（Overview / Contents / SSL / WebSocket / Summary / Chart / Notes）
- Contents 视图采用上下双窗格：上半部分展示请求，下半部分展示响应
- 请求/响应面板均改为紧凑型 key/value 展示，弱化表格感，贴近 Charles 的信息密度
- 请求底部标签栏支持：Headers、Query String、Cookies、Text、Hex、Raw
- 响应底部标签栏支持：Headers、Text、Hex、Raw
- 上下窗格之间支持拖拽调整高度，并持久化到 localStorage
- 所有关键字段、正文、Raw/Hex 内容仍支持右键 AI 分析
- 新增组件：
  - `RequestSection.vue` - Charles 风格请求内容窗格
  - `ResponseSection.vue` - Charles 风格响应内容窗格

#### 右键菜单 AI 分析
- 在请求详情页面右键点击任意参数可触发 AI 分析
- 支持右键分析的元素：
  - Request Headers / Response Headers
  - Request Body / Response Body
  - Cookies
- 右键菜单功能：
  - "AI Analyze" - 将参数发送到 AI 面板进行分析
  - "Copy Value" - 复制参数值到剪贴板
- 新增组件：
  - `ContextMenu.vue` - 右键菜单组件
- aiStore 新增方法：
  - `analyzeParameter()` - 分析指定参数
  - `analyzeWithQuery()` - 自定义查询分析

#### 域名折叠显示
- 请求列表按域名(Host)分组折叠显示，类似 Charles Proxy
- 支持展开/折叠单个域名组
- 支持一键全部展开/折叠
- 显示每个域名下的请求数量
- 按时间倒序排列域名组和请求
- 抓包栏工具条移除 Method 筛选，简化顶部布局

#### 可调整宽度面板
- 三栏布局（抓包列表、请求详情、AI对话）支持鼠标拖拽调整宽度
- 宽度自动持久化到 localStorage
- 设置最小/最大宽度限制防止布局崩溃
- 拖拽时禁用文本选择，优化用户体验

#### 前端桌面工具化视觉调整
- 整体界面进一步降低 Web 感，向桌面抓包工具视觉靠拢
- 主框架、抓包栏、详情栏、AI 面板统一改为更紧凑的灰阶工具风格
- 收紧头部高度、按钮尺寸、标签栏高度与间距，增强信息密度
- 降低圆角、弱化卡片感，强化分隔线、边框与面板结构感

#### SOCKS5 HTTP 流量解析
- SOCKS5 代理现在可以解析明文 HTTP 流量
- 正确显示实际的 HTTP 方法（GET、POST、PUT、DELETE 等）而非 CONNECT
- 捕获完整的请求/响应 Headers
- 捕获请求/响应 Body（最大 5MB）
- 捕获 Cookies
- 自动检测协议类型（HTTP/TLS/其他）
- 新增函数：
  - `looksLikeHTTP()` - 检测 HTTP 请求
  - `isTLSHandshake()` - 检测 TLS 握手
  - `handleSOCKS5HTTP()` - 解析 HTTP 流量
  - `limitedWriter` - 限制捕获大小的写入器

#### AI 助手 Agent 化
- AI 助手面板支持模型切换功能
- 模型配置从 `config.yaml` 移至独立 JSON 文件 (`configs/models.json`)
- 前端下拉菜单选择不同 AI 模型
- 支持多模型预置（GLM-4系列、GPT-4系列、Mock演示）

### 架构变更

#### AI 配置管理重构
- 新增 `configs/models.json` 配置文件，包含：
  - 预置模型列表（id、名称、provider、描述、max_tokens）
  - 默认模型设置
  - Provider API Key 配置
- `internal/config/config.go` 新增：
  - `ModelConfig` 结构体
  - `ModelsConfig` 结构体
  - `LoadModels()` 加载函数
  - 辅助方法：`GetModelByID()`, `GetModelsByProvider()`, `GetAPIKey()`
- `internal/ai/analyzer.go` 新增 `GetAPIKey()` 方法

### API 变更

#### 新增接口
- `GET /api/ai/models` - 获取可用模型列表

#### 修改接口
- `POST /api/ai/analyze` - 新增 `model_id` 参数支持动态模型选择

### 前端变更

- `AIPanel.vue` - 新增模型选择下拉菜单和模型信息展示
- `aiStore.ts` - 新增模型状态管理（models、selectedModelId、loadModels）
- `api/index.ts` - 新增 `aiApi.listModels()` 和 `fetchModels()` 方法
- `types/index.ts` - 新增 `AIModel` 类型定义

### 文件变更

| 文件/目录 | 变更类型 | 说明 |
|-----------|----------|------|
| `configs/models.json` | 新增 | AI 模型配置文件 |
| `internal/config/config.go` | 修改 | 添加模型配置加载 |
| `internal/storage/models.go` | 修改 | AnalysisRequest 添加 model_id |
| `internal/ai/analyzer.go` | 修改 | 添加 GetAPIKey 方法 |
| `internal/api/handlers/ai.go` | 修改 | 支持动态模型选择 |
| `cmd/server/main.go` | 修改 | 加载模型配置 |
| `web/src/components/AIPanel.vue` | 修改 | 添加模型选择器 |
| `web/src/stores/aiStore.ts` | 修改 | 模型状态管理 |
| `web/src/api/index.ts` | 修改 | 新增模型 API |
| `web/src/types/index.ts` | 修改 | 新增 AIModel 类型 |
| `internal/proxy/proxy.go` | 修改 | 添加 SOCKS5 HTTP 解析 |
| `web/src/components/RequestDetail.vue` | 修改 | Body base64 解码 |
| `web/src/App.vue` | 修改 | 实现可调整宽度的三栏布局 |
| `web/src/components/RequestList.vue` | 修改 | 实现域名折叠功能 |
| `web/src/components/ContextMenu.vue` | 新增 | 右键菜单组件 |
| `web/src/components/HeadersView.vue` | 修改 | 添加右键分析支持 |
| `web/src/components/RequestDetail.vue` | 修改 | 添加 Body/Cookies 右键分析 |
| `web/src/components/AIPanel.vue` | 修改 | 支持自定义查询分析 |
| `web/src/stores/aiStore.ts` | 修改 | 添加分析方法 |

## [0.5.0] - 2026-03-10

### 架构变更

#### 存储模块重构 (SQLite → In-Memory)
- 移除 SQLite/GORM 依赖，改用纯内存存储（类似 Charles Proxy）
- 使用 `sync.RWMutex` 实现线程安全
- 添加多级索引（requestsBySession, requestsByHost, analysesByRequest）优化查询性能
- 新增数据导入/导出功能：
  - `ExportAll() ([]byte, error)` - 导出所有数据为 JSON
  - `ImportAll(data []byte) error` - 从 JSON 导入数据
  - `ExportHAR(requestID string) ([]byte, error)` - 导出单个请求为 HAR 格式
- **影响**: 数据不再持久化到磁盘，重启后数据清空（可通过导入恢复）

#### 前端框架迁移 (React → Vue 3)
- 框架从 React 18 迁移到 Vue 3 + TypeScript
- UI 库从 Ant Design React 迁移到 Ant Design Vue 4.x
- 状态管理从 Zustand 迁移到 Pinia
- 构建工具保持 Vite（重新配置）
- 新增 Vue composables 替代 React hooks

### 新增功能
- 存储模块新增内存数据导入/导出能力
- 新增内存二级索引以提升查询与清理性能

### 代码改进
- `internal/storage/database.go` 完全重写为线程安全的内存实现
- `go.mod` 移除 GORM 和 SQLite 依赖
- 前端组件完全重写为 Vue 3 Composition API

### 文档更新
- `AGENTS.md` 同步更新存储介质说明为 In-Memory
- `AGENTS.md` 更新前端技术栈为 Vue 3

### 文件变更

| 文件/目录 | 变更类型 | 说明 |
|-----------|----------|------|
| `internal/storage/database.go` | 重写 | SQLite → In-Memory |
| `go.mod` | 修改 | 移除 gorm/sqlite 依赖 |
| `web/package.json` | 重写 | React → Vue 3 依赖 |
| `web/vite.config.ts` | 修改 | Vue 配置 |
| `web/tsconfig.json` | 修改 | Vue TypeScript 配置 |
| `web/src/*.tsx` | 删除 | React 组件 |
| `web/src/*.vue` | 新增 | Vue 组件 |
| `web/src/stores/*.ts` | 重写 | Zustand → Pinia |
| `web/src/composables/*.ts` | 新增 | Vue composables |

## [0.4.0] - 2026-03-10

### 文档更新

#### AGENTS.md 开发指南
- 创建 AI 编码代理开发指南文档
- 包含完整的 Build/Lint/Test 命令说明
- 详细代码风格指南（导入、命名、错误处理、结构体设计等）
- 测试约定和架构模式说明
- 常见开发任务指引
- **重要**: 每次代码改动必须更新 AGENTS.md 和 CHANGELOG.md

### 文档规范

#### 变更记录要求
- 所有代码变更必须在 CHANGELOG.md 中记录
- 架构/约定变更需同步更新 AGENTS.md
- 使用标准格式：版本号、日期、分类（新增/修复/改进/文档）

### 文件变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `AGENTS.md` | 新增 | AI 代理开发指南 |
| `CHANGELOG.md` | 修改 | 添加文档更新记录 |

---

## [0.3.0] - 2026-02-28

### 新增功能

#### SOCKS5代理 (`internal/proxy/proxy.go`)
- 完整SOCKS5协议实现（RFC 1928）
- 支持无认证模式
- 支持用户名/密码认证（RFC 1929）
- 支持IPv4/IPv6/域名地址类型
- 默认端口：1080（可在 `configs/config.yaml` 中修改）
- 配置项：
  ```yaml
  proxy:
    socks5:
      enabled: true
      port: 1080
      username: ""  # 可选
      password: ""  # 可选
  ```

#### 配置管理API (`internal/api/handlers/config.go`)
- `GET /api/config` - 获取当前配置（安全返回，不含敏感信息）
- `PUT /api/config/ai` - 更新AI配置（provider、api_key、model等）
- `GET /api/config/cert` - 下载CA证书（用于安装到系统/浏览器）
- `GET /api/config/cert/info` - 获取证书信息（路径、大小、是否存在）

#### 请求重放功能 (`internal/api/handlers/handlers.go`)
- `POST /api/requests/:id/replay` - 重放指定请求
- 支持自定义Headers和Body
- 自动保存重放结果为新请求记录
- 返回重放结果（状态码、耗时、响应大小）

#### 前端更新
- `web/src/api/index.ts` - 新增 `requestApi.replay()` 和 `configApi` 方法
- `web/src/components/RequestDetail.tsx` - 添加"重放"和"cURL导出"按钮
- `web/src/components/RequestDetail.module.css` - 新增按钮样式

### 文件变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/proxy/proxy.go` | 修改 | 添加SOCKS5代理支持 |
| `internal/api/handlers/config.go` | 新增 | 配置管理Handler |
| `internal/api/handlers/handlers.go` | 修改 | 添加请求重放功能 |
| `cmd/server/main.go` | 修改 | 注册配置API路由 |
| `web/src/api/index.ts` | 修改 | 新增API调用方法 |
| `web/src/components/RequestDetail.tsx` | 修改 | 添加重放/导出按钮 |
| `web/src/components/RequestDetail.module.css` | 修改 | 按钮样式 |

### API接口汇总

```
# 配置管理
GET    /api/config           获取配置
PUT    /api/config/ai        更新AI配置
GET    /api/config/cert      下载CA证书
GET    /api/config/cert/info 证书信息

# 请求管理（新增）
POST   /api/requests/:id/replay  重放请求
```

---

## [0.2.0] - 2026-02-27

### 已实现功能

#### 代理服务
- [x] HTTP代理（端口8888）
- [x] HTTPS代理（MITM解密）
- [x] SSL/TLS解密（动态生成证书）
- [x] CA证书自动生成和管理

#### 请求管理
- [x] 请求列表展示（分页、排序）
- [x] 请求详情查看（Headers、Body、Response、Cookies）
- [x] 基础过滤（域名、方法、状态码、搜索）
- [x] 请求导出（HAR、cURL格式）
- [x] 请求删除/清空

#### AI分析
- [x] 右键"AI分析"链接触发
- [x] 参数分析（推断生成方式）
- [x] 上下文分析（历史请求对比）
- [x] 流式输出（WebSocket）
- [x] 支持智谱GLM-4和OpenAI

#### WebSocket
- [x] 实时请求通知
- [x] AI分析流式推送

---

## 待开发功能

### P1 - 高优先级
- [ ] 设置页面UI（代理配置、AI配置）
- [ ] 右键菜单（文档要求的交互方式）
- [ ] 请求对比功能
- [ ] 请求拦截修改

### P2 - 中优先级
- [ ] 上游代理支持
- [ ] 自定义Prompt模板
- [ ] 暗色模式
- [ ] 请求搜索增强（正则表达式）

### P3 - 低优先级
- [ ] 插件系统
- [ ] 脚本支持
- [ ] 导出更多格式（Postman、Insomnia）

---

## 开发环境

- Go 1.21+
- Node.js 18+
- React 18
- Ant Design 5.x

## 启动命令

```bash
# 后端
go mod tidy
make dev

# 前端
cd web
npm install
npm run dev
```

## 配置文件

配置文件位于 `configs/config.yaml`，主要配置项：

```yaml
server:
  host: 127.0.0.1
  port: 8080

proxy:
  http:
    enabled: true
    port: 8888
  https:
    enabled: true
    port: 8888
    mitm: true
  socks5:
    enabled: true
    port: 1080

ai:
  provider: zhipu  # zhipu | openai
  api_key: ""
  model: glm-4
```
## [0.6.6] - 2026-03-12

### 新增功能
- 请求详情 Contents 视图中的 Request/Response Body 新增条件化 `Tree` 标签页：当正文可解析为 JSON 或 XML 时展示层级树，支持对象/数组、XML 元素/属性/文本节点的紧凑浏览

### 代码改进
- 抽离并复用正文解码与格式化逻辑到 `web/src/components/bodyTreeUtils.ts`，统一 Text/Tree 视图对 Base64 文本体、JSON 美化与 XML/JSON 自动识别行为
- 新增 `web/src/components/BodyTreeNodeView.vue` 递归树节点组件，保持现有 Charles 风格高密度灰阶视觉，不影响 Text/Hex/Raw 原有交互
