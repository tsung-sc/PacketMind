# AGENTS.md - PacketMind 开发指南

> 本文档为 AI 编码代理提供代码库上下文和开发规范。
> **每次代码改动后，必须更新本文档和 CHANGELOG.md**

---

## 项目概览

**PacketMind** 是一个 AI 驱动的智能抓包分析工具，使用 Go 1.21+ 后端 + Vue 3 前端，通过 Wails v2 打包为桌面应用。

- **模块**: `github.com/packetmind/packetmind`
- **桌面框架**: Wails v2
- **存储**: In-Memory（进程内存，带多索引，支持导入/导出）
- **AI 集成**: 智谱 GLM-4 (主力), OpenAI (备选)
- **前端**: Vue 3 + TypeScript + Ant Design Vue + Pinia
- **入口点**: `app.go` (Wails 应用主入口)

---

## Build/Lint/Test 命令

### 桌面应用 (Wails)

```bash
# 开发运行
wails dev                 # 启动 Wails 开发模式 (热重载)

# 构建
wails build               # 构建生产版本

# 生成绑定
wails generate module     # 重新生成前端绑定
```

### 后端 (Go)

```bash
# 测试
make test                # 运行所有测试 (go test -v -race ./...)
go test -v -race ./...   # 直接运行所有测试
go test -v -race ./internal/proxy  # 运行单个包的测试

# 代码检查
make lint                # 运行 golangci-lint
make fmt                 # 格式化代码 (go fmt ./...)

# 清理
make clean               # 删除 bin/ 和 data/ 目录
```

### 前端 (Vue 3)

```bash
cd gui

# 开发 (在 wails dev 中自动启动)
npm run dev              # 启动开发服务器 (Vite)

# 构建
npm run build            # TypeScript 编译 + Vite 构建

# 预览
npm run preview          # 预览生产构建
```

---

## 架构

### Wails 绑定

前端通过 Wails bindings 调用后端方法：

```typescript
// 前端调用示例
import { sessionApi, requestApi, agentApi } from '@/api/wails'

const response = await sessionApi.list()
console.log(response.code, response.data)
```

后端绑定位于 `internal/api/bindings/`:
- `SessionAPI` - 会话管理
- `RequestAPI` - 请求管理
- `AgentAPI` - Agent 分析
- `ConfigAPI` - 配置管理
- `ProxyAPI` - 代理控制
- `UpdaterAPI` - 应用自更新

### Wails 事件系统

前端通过 Wails Events 接收后端广播：

```typescript
// 前端订阅示例
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

EventsOn('request:new', (data) => {
  console.log('New request:', data)
})

EventsOn('agent:analysis', (data) => {
  console.log('Agent event:', data)
})
```

后端通过 `runtime.EventsEmit` 广播事件：
- `request:new` - 新请求到达
- `agent:analysis` - Agent 分析事件 (agent_event, agent_result, done)
- `update:progress` - 自更新下载进度
- `update:done` - 自更新完成事件

---

## 代码风格指南

### 导入组织

遵循 Go 标准导入顺序，使用空行分隔：

```go
import (
	// 1. 标准库
	"context"
	"fmt"
	"net/http"
	
	// 2. 第三方库
	"github.com/gin-gonic/gin"
	
	// 3. 项目内部包
	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/storage"
)
```

### 命名约定

#### 包名
- 使用简短、小写、单个单词
- 例: `proxy`, `storage`, `agent`, `handlers`

#### 类型命名
- **PascalCase** 用于导出类型
- **camelCase** 用于私有类型
- 接口使用 `-er` 后缀 (如 `Analyzer`, `Storage`)

```go
// 导出
type Request struct { ... }
type Storage struct { ... }

// 私有
type zhipuRequest struct { ... }
```

#### 函数/方法命名
- 构造函数: `New<Type>()`
- Getter: 不使用 `Get` 前缀，直接用字段名
- Setter: `Set<Field>()`

```go
// 构造函数
func NewStorage(dsn string) (*Storage, error)
func NewAnalyzer(store *storage.Storage, provider string, ...) *Analyzer

// 方法
func (s *Storage) GetRequest(id string) (*Request, error)
func (p *Proxy) SetOnRequest(fn func(*storage.Request))
```

#### 常量
- 分组相关常量
- 使用 `iota` 枚举

```go
const (
	socks5Version  = 5
	socks5NoAuth   = 0
	socks5Connect  = 1
)
```

### 错误处理

#### 错误返回
- 总是检查错误
- 使用 `fmt.Errorf` 包装错误并添加上下文
- 不要吞掉错误

```go
// 正确
if err := db.SaveRequest(record); err != nil {
	return fmt.Errorf("failed to save request: %w", err)
}

// 错误
if err != nil {
	return err  // 缺少上下文
}
```

#### 日志记录
- 使用 `fmt.Printf` 或 `log.Printf` 记录关键操作
- 包含组件前缀: `[Proxy]`, `[Server]`, `[SOCKS5]`

```go
fmt.Printf("[Proxy] Starting HTTP/HTTPS proxy on %s\n", addr)
log.Printf("Warning: Failed to load config: %v", err)
```

### 结构体设计

#### 数据模型
- 使用 JSON 标签
- 主键: `ID string`
- 时间戳: `CreatedAt time.Time`

```go
type Request struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	
	Method      string `json:"method"`
	Host        string `json:"host"`
	StatusCode  int    `json:"status_code"`
	
	Headers Headers `json:"headers"`
	Body    []byte  `json:"body"`
}
```

#### 自定义类型
- 为复杂字段实现 `Value()` 和 `Scan()` 方法

```go
type Headers map[string][]string

func (h Headers) Value() (driver.Value, error) {
	return json.Marshal(h)
}

func (h *Headers) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), h)
}
```

### API 处理器

#### 标准响应格式
```go
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 成功响应
c.JSON(http.StatusOK, Response{
	Code: 0,
	Data: result,
})

// 错误响应
c.JSON(http.StatusInternalServerError, Response{
	Code:    50001,
	Message: err.Error(),
})
```

#### 分页参数
```go
page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
```

### 配置管理

#### 配置结构
桌面/代理运行配置统一持久化到 `configs/packetmind.json`；AI 可变配置继续单独持久化到 `configs/models.json`

```go
type DesktopSettings struct {
	Proxy  DesktopProxySettings  `json:"proxy"`
	Tools  DesktopToolSettings   `json:"tools"`
	Window DesktopWindowSettings `json:"window"`
	Cert   CertSettings          `json:"cert"`
}

type ModelsConfig struct {
	Models            []ModelConfig     `json:"models"`
	DefaultModel      string            `json:"default_model"`
	ProviderAPIKeys   map[string]string `json:"provider_api_keys"`
	ProviderBaseURLs  map[string]string `json:"provider_base_urls,omitempty"`
	ActiveProvider    string            `json:"active_provider,omitempty"`
	ActiveModel       string            `json:"active_model,omitempty"`
	ActiveMaxTokens   int               `json:"active_max_tokens,omitempty"`
	ActiveTemperature float64           `json:"active_temperature,omitempty"`
}
```

`DefaultPacketMindSettings()` / `LoadPacketMindSettings()` / `SavePacketMindSettings()` 负责 `packetmind.json` 的默认值、读取与原子写回；`MigrateLegacyConfigs()` 会在首次启动时 best-effort 将旧 `config.yaml` + `desktop.json` 合并迁移到 `packetmind.json`

### 并发模式

#### Goroutine 错误处理
```go
go func() {
	if err := prox.Start(ctx); err != nil {
		log.Printf("Failed to start proxy: %v", err)
	}
}()
```

#### Channel 使用
```go
// 带缓冲的 channel
output := make(chan StreamChunk, 100)

// 在 goroutine 中关闭
go func() {
	defer close(output)
	// ... 发送数据
}()
```

### 注释规范

#### 导出函数必须有注释
```go
// NewStorage 创建新的数据库连接
// dsn: 数据库连接字符串 (SQLite 文件路径)
func NewStorage(dsn string) (*Storage, error) { ... }
```

#### 复杂逻辑添加行内注释
```go
// SOCKS5 握手: 读取客户端方法列表
buf := make([]byte, 257)
n, err := conn.Read(buf)
```

---

## 测试约定

**当前状态**: 已添加后端回归测试，当前包含 `internal/proxy/proxy_test.go`（覆盖 SOCKS5 HTTPS 首包转发与 MITM 请求记录）、provider 单元测试 `internal/agent/provider/openai/client_test.go`（统一覆盖 OpenAI/Zhipu preset 构造函数、stream-only chat/错误处理、`CollectStream()` 聚合与 `ListModels()` 场景），以及 `internal/storage/storage_test.go`（会话激活/更新/删除行为 + chat message 持久化/导入导出）与 `internal/config/config_test.go`（AI 可变配置持久化到 models.json）

- `internal/agent` 当前自定义 `llmtypes` provider / runtime / MCP 测试应保持 stream-only 合同：provider 仅走 `Stream()` / `WithTools()`，需要完整 assistant turn 的调用方统一通过 `llmtypes.CollectStream()` 聚合 chunks；生产态 chat context 已改为直接从 storage `chat_message` 恢复 `[]*LLMMessage`，MCP / tool 适配测试维持现有公共结构断言

### 测试文件命名
- 文件名: `<filename>_test.go`
- 测试函数: `func Test<FunctionName>(t *testing.T)`

### 测试模式
```go
func TestProxyHandleHTTP(t *testing.T) {
	// Arrange
	proxy := NewTestProxy()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	// Act
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)
	
	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
```

---

## 架构模式

### 依赖注入
所有组件通过构造函数注入依赖:

```go
// main.go
store, _ := storage.NewStorage()
prox := proxy.New(cfg.Proxy, cfg.Cert, store)
modelsStore := config.NewModelsStore("./configs", modelsCfg)
handler := handlers.NewHandler(store, prox)
```

### 事件驱动
使用回调函数解耦模块:

```go
prox.SetOnRequest(func(req *storage.Request) {
	hub.BroadcastAll(map[string]interface{}{
		"type": "new_request",
		"data": req,
	})
})
```

### 流式处理
AI Agent 分析通过 WebSocket 逐步广播 `agent_event` / `agent_result`，前端按时间线流式展示。

### Agent 推理链
- `internal/agent/agent.go` 提供请求级分析编排（请求校验、prompt 装配、trace/event 聚合），并作为唯一分析入口直接驱动单一 llmtypes/runtime 运行时
- `internal/agent/Agent` 当前显式持有 `store *storage.Storage` 与 `executor *tools.Executor`；运行期请求/检索/溯源访问统一通过 `SetStore(store)` 注入的 storage 执行，不要再回退到父包直接读取 `storage.Default`
- `internal/agent/llmcore` 现为 Agent/provider 共享叶子包：仅保留 `ModelInfo`、`ModelLister`、`AgentError` / provider retry 与 provider HTTP 错误类型；不再承载自定义消息/tool/usage/SSE 协议类型，避免 `agent -> provider -> agent` 循环依赖
- `internal/agent/provider/openai/` 现承载 OpenAI API wire-protocol family 客户端（含官方 OpenAI、Zhipu 与任意 OpenAI-compatible endpoint）；目录语义是“协议族”而非厂商名，后续同协议 provider 应继续放在该子目录下扩展，而不是回退到根 `provider/` 扁平文件布局
- `internal/agent/provider/openai/` 当前由 `client.go` / `chat.go` / `stream.go` 构成该协议族实现：`Client` 仅暴露 stream-only `Stream()` / `WithTools()` / `ListModels()`；品牌 preset 薄封装已删除，provider 差异统一通过 `Config` 控制 base URL 归一化、stream reasoning 提取与 provider 标签
- `internal/agent/llmtypes.LLMClient` 现为 stream-only 接口；需要完整 assistant turn 的运行时（如 `internal/agent/runtime.Runner`）必须通过 `llmtypes.CollectStream()` 聚合 `LLMStreamReader`，不要重新引入 `Generate()` 或旁路的同步 completion 传输层
- `internal/agent/runtime/` 现承载独立运行时子包（`runner.go` / `events.go` / `usage.go` / `helpers.go` / `retry.go`）：父包 `agent` 只能通过函数注入 `safeExecuteTool` 与 type alias 复用其 `AgentEvent` / `RuntimeResult` / `ToolExecutionResult` / `SafeToolResult`，禁止让 runtime 反向导入 `internal/agent`
- `internal/agent/tools/` 现承载与父包解耦的工具基础设施：`catalog.go` 管 tool 常量/schema clone helper，`args.go` 管参数解析，`validate.go` 管 schema 校验，`executor.go` 管 builtin/MCP 分发与 safe execute，`suggest.go` 管未知 tool 模糊匹配；该子包可依赖 `runtime` / `llmtypes` / `storage`，但不得反向导入父包 `agent`
- `internal/agent/tools/builtin/` 现承载所有内置工具实现（`request/search/provenance/diff/encoding/batch/shell/helpers/register`）；内置工具必须通过显式注入的 `*storage.Storage` 执行，不得继续直接读取 `storage.Default` 或依赖父包 `Agent` 方法承载真实实现；在恢复明确审批流之前，不要把 `bash` 注册回模型可见的 builtin schema/handler 集合
- `internal/agent/Agent` 本轮已恢复显式 `store *storage.Storage` 与 `executor *tools.Executor` 字段：`SetStore(store)` 负责注册 builtin tools，`SetMCPManager(manager)` 负责装配带 MCP schema 的 executor，`Analyze()` / `runtime.Runner` 只通过 `executor.SafeExecute` 执行工具；后续不要再恢复父包 giant switch 或把工具实现塞回 `agent.go`
- `internal/agent/agent.go` 中面向 `llmtypes` 的 ReAct 循环现通过 `internal/agent/runtime.Runner` 构造，并以 `runtime.WithExecuteTool(a.safeExecuteTool)` 注入工具执行；不要再把 `*Agent` 塞回运行时结构体中形成循环依赖
- `Agent.Analyze()` 当前直接装配基于 `llmtypes` 的 runtime 输入；生产态 chat context 统一从 storage `chat_message` 恢复最近尾部窗口内的 `user` / `assistant` turn，不再经过 `SessionMemory` / schema-message 转换层，也不要把持久化的 `system` / `tool` 记录直接回放给模型
- `internal/agent` 本轮已删除重构后遗留的死代码：旧的 dispatcher/memory/compat helper 与无调用残留应继续优先删除，避免重新引入兼容壳层
- 生产态 Agent chat context 已完全切换为 storage-only：对话历史统一持久化到 `chat_message`，分析前由 `Agent.Analyze()` 直接读取最近尾部窗口内的 `user` / `assistant` turn；不要重新引入 `SessionMemory`、LRU、预热或磁盘快照缓存层
- `AgentAPI.GetChatHistory(sessionID)` / `GetSessionContext(sessionID)` 必须直接读取 storage `chat_message`，读取历史时不得创建任何临时内存会话对象或额外缓存
- `internal/agent/tools/catalog.go` 集中维护内置工具 schema 与工具名常量；新增/调整内置工具时应优先在该 catalog 中维护，并在 executor/builtin 分发侧复用常量避免字符串漂移
- Agent 内置工具已支持 `search_all_fields` 跨字段检索；实现位于 `internal/agent/tools/builtin/search.go`，新增同类搜索能力时应优先复用当前 builtin handler 聚合模式
- `internal/storage/decode.go` 提供共享 `DecodeBodyBytes()`/`GetContentEncoding()` 解压辅助能力；凡 Agent body 预览、storage body 文本搜索、provenance body artifact 提取等读取 request/response body 明文语义的路径，都必须先基于 `Content-Encoding` 对 gzip/deflate/brotli 内容解压，再做文本判断、关键字匹配或 JSON/form 解析
- `internal/storage/models.go` 中 `FindInSessionOptions` / `FindInSessionMatch` 用于会话内全文检索；`internal/storage/storage.go` 的 `FindInSession()` 当前支持 request URL、request headers/body、response headers/body、notes、error 范围检索，并支持 literal/regex、大小写与 whole-word 匹配；凡检索 request/response body 时必须先复用 `DecodeBodyBytes()` 解压正文
- `internal/storage` 当前已完成 session/request 的 memdb 收尾：`Storage` 仅保留 `db *memdb.MemDB`，不再维护 `d.mu` / `d.sessions` 镜像缓存，也不再保留 `activeMu` / `activeSession` 一类 active-session cache 或任何 `*Locked` 兼容 shim；`search.go`、`har.go`、`export.go`、`provenance_query.go` 的读取必须统一走 `d.db.Txn(false)`、`ListSessions()`、`GetRequest()` 或 memdb-native request 查询
- `internal/storage` 现新增第三张 memdb table `chat_message`：用于按 capture `session_id` 持久化 Agent 对话历史（user/assistant turn），索引至少包含 `id`、`session_id` 与 `session_created_at`；删除 session 时必须与 request 一样在同一事务内级联删除对应 chat messages
- `internal/storage/request.go` 的 `GetRequest(sessionID, id)` 现要求显式传入 session ID，不再在空 session 下回退到 active session；显式读/溯源路径应直接传递已知 session ID，`ExportHAR()` 这类真正 sessionless 的内部场景应使用私有按 ID 查找 helper，而不要放宽公开 API
- `internal/storage/helpers.go` 的 `cloneRequest()` 必须深拷贝 TLS 相关切片字段（包括 `TLSServerCertificates` / `TLSServerExtensions` 及证书内嵌切片字段），避免返回值与 memdb row 共享底层切片造成别名写穿
- Session 与 Request 分别落在独立 memdb table；runtime `Session` 仅承载 session 元数据，`GetSession()` / `ListSessions()` 应保持 metadata-only 返回；派生的 `requests/host_groups` 统一收敛到 `SessionView`（`GetSessionView()` / `ListSessionViews()`）与导入导出 envelope，在需要聚合视图时按 request table 动态派生；新增 storage 读路径不要再把派生字段塞回 `Session` / `sessionRecord`，也不要恢复 `collectRequestCandidates` 一类仅吞错误的薄 wrapper
- `ImportAll()` 在导入 payload 未提供 `ActiveSession` 且各 session 也未标记 `IsActive` 时，必须回退选择最新 session 作为 active，保持与 `CreateSession()` / `DeleteSession()` 一致的“非空存储最多且至少一个 active session”语义
- Agent 内置工具现分为两类：基础分析工具（`get_request` / `search_*` / `find_prior_response_sources` / `find_later_request_usages` / `trace_value_flow` / `analyze_encoding` / `diff_requests` / `search_all_fields` / `batch_execute` / `bash`）与认知探索工具（`summarize_session` / `classify_requests` / `trace_flow_sequence` / `test_hypothesis`）
- 认知探索工具位于 `internal/agent/tools/builtin/summarize.go`、`classify.go`、`flow.go`、`hypothesis.go` + `hypothesis_parser.go`；新增同类能力时应优先沿用“独立工具文件 + catalog.go schema + exports.go 导出 + mcp/builtin.go 注册”的统一接入模式
- `summarize_session` 遍历 `store.ListRequests()` 按 host 聚合，生成 status/content-type 分布、关键端点、auth/signed 检测与 critical_observations；输出控制在 ~3000 字符
- `classify_requests` 使用轻量启发式（path/header/content-type/status/body 关键词）将请求分类为 auth_entry、token_issuance、signed_request 等 10 类，并生成 auto_focus_suggestion 引导下一步
- `trace_flow_sequence` 按时序扫描请求，检测 cookie_flow（Set-Cookie → Cookie）、token_flow（response token → Bearer）、value_reuse、redirect_chain，并输出 gaps_and_anomalies
- `test_hypothesis` 内置轻量表达式解析器（`hypothesis_parser.go`），支持 EXTRACT(body|header|query|response_body|response_header|cookie, path)、CONCAT、CONCAT_WITH、LOWER/UPPER、MD5/SHA1/SHA256、HMAC_SHA256、BASE64、URLENCODE、字面量；逐请求计算并与 actual 值比对，返回 match rate 与 alternative_hypotheses
- `analyze_encoding` 应优先输出结构化编码链与分层预览（包含最终类型与错误摘要）；`search_all_fields` 结果应保持 `found_results` / `count` / `error_summary` 结构；`batch_execute` 应默认并发聚合子调用结果且不改变既有单工具实现
- AI 模块已引入 `internal/agent/prompt.go` 的 `SystemPromptBuilder` 用于组合系统提示词；`Analyze()` 路径应优先通过 builder 叠加基础角色与动态上下文，不要重新引入 specialist/orchestrator prompt 分支
- `internal/api/bindings/agent.go` 的 Wails agent 分析入口当前应直接调用单一 `Agent.Analyze()` 运行时；绑定层只负责把完成后的 `user`/`assistant` 往返持久化到 storage `chat_message`，不要恢复内存会话复用逻辑
- AI Agent 阶段二已引入 `AgentError`：当前用于标注 tool 级错误的 `retryable / recoverable / fatal` 分类与附加上下文，新增错误类型时应优先复用该结构
- AI Agent 阶段二已为 tool 执行引入超时与 panic recovery 包装：应优先通过 `safeExecuteTool()` 返回结构化 observation，而不是把底层 panic/timeout 直接冒泡为进程级故障
- Agent `executeTool()` 在 unknown tool 分支应优先返回结构化 sentinel observation（`ok=false` + `error` + `available_tools` + 可选 `suggestion/hint`）而非直接返回 error，以便模型在同一轮循环内自纠正工具名并继续推理
- AI Agent 阶段三已引入最小 MCP 集成：`internal/agent/mcp.go` 提供 `MCPClient` 接口、`MCPToolAdapter` 适配器与 `MCPManager` 管理器
- AI 模块已将 Agent/Provider 共享导出类型集中到 `internal/agent/types.go`（如 `LLMMessage`、`Tool*`、`LLMChat*`、`Agent*`）；新增或调整这些公共结构时应优先在该文件维护，避免在 `agent.go` 重复定义
- Agent / storage 的能力边界当前通过显式 `*storage.Storage` 注入和 `tools/builtin` handler 分层表达；后续若需要新增抽象，优先保持在最小必要范围内，不要为测试或兼容性恢复已删除的 `interfaces.go`
- 产品运行链路已切换为 Agent-only：`internal/api/bindings/agent.go` 不再保留 legacy `Analyzer` 执行分支，`POST /api/agent/analyze` 语义对应的分析入口统一走单一 `Agent.Analyze()`
- AI 可变配置（provider/model/api_key/base_url/max_tokens/temperature）统一以 `configs/models.json` 为持久化源，前端通过 `PUT /api/config/agent` 更新活动模型设置
- 模型配置弹窗中通过 provider 动态发现到的模型在用户确认保存后也必须回写到 `configs/models.json` 的 `models` 列表；AI 分析栏模型下拉应始终以该本地持久化列表为准，而不是依赖前端临时缓存
- 会话管理支持多会话 + 单 active：后端必须保证任一时刻最多一个 `is_active=true`，并且代理抓包仅写入当前 active session
- 代理监听端口（HTTP/HTTPS/SOCKS5）支持运行时热重载：`ApplyDesktopSettings` 在代理运行中可触发 listener 预绑定 + 原子切换；端口冲突时必须保持旧 listener 继续服务
- HTTPS MITM 解密能力当前应同时受 `Proxy.Listener.MITMEnabled` 与 `Proxy.SSLProxying.Enabled` 控制；当 `Enable SSL Proxying` 关闭时，CONNECT/TLS 流量必须回落为原始 tunnel，不应继续做证书置换与明文抓包
- HTTP/HTTPS/SOCKS5 统一共享同一个入站 listener 与端口（`ProxyConfig.Port`）；协议分发由连接首字节自动判定（`0x05` 走 SOCKS5，其余走 HTTP/HTTPS），`SOCKS5Enabled` / `HTTPEnabled` / `HTTPSEnabled` 仅控制协议可用性
- 桌面设置中监听端口字段为 `ProxyListenerSettings.Port`（JSON key `"port"`），前端通过 `desktopSettings.proxy.listener.port` 读写
- `ProxyConfig` 的 `Port` 字段为统一监听端口，`HTTPProxyConfig`/`HTTPSProxyConfig`/`SOCKS5ProxyConfig` 不再包含各自独立的 Port 字段
- 监听器热切换时禁止在持有 `Proxy.mu` 的情况下执行 `http.Server.Shutdown()`；应先完成 listener/server 引用交换，再在锁外优雅关闭旧实例避免死锁
- Agent 工具执行在进入 builtin/MCP handler 前必须执行参数 schema 校验：优先复用 `internal/agent/tools/validate.go` 的轻量校验（当前覆盖 `type(string/integer)`、`required`、`enum`、`minimum/maximum`）；校验失败应返回结构化 `ok=false` 错误 observation（含可纠正 hint），而不是继续调用具体工具实现
- AI 模块的工具结果截断能力当前位于 `internal/agent/runtime/truncate.go`：当工具输出超过 `TruncateMaxLines` / `TruncateMaxBytes` 时将完整内容原子写入 `data/tool-output/`，返回预览 + 文件路径提示；后续不要重新引入父包级 `internal/agent/truncate.go`
- `CleanupOldTruncationFiles()` 清理 `TruncateOutputDir` 中超过 `TruncateRetention`(7 天) 的文件，应在应用启动或周期性维护中调用
- 截断输出文件采用临时文件 + rename 原子写入，工具名经 `sanitizeToolName` 清理后用于文件名前缀
- MCP 集成采用窄接口设计：`MCPClient` 仅包含 `ListTools(ctx)` 与 `CallTool(ctx, name, args)` 两个方法，满足最小工具发现与调用需求
- `MCPToolAdapter` 支持工具名前缀命名（格式 `{prefix}_{original_name}`），避免不同 MCP 服务端与内置工具之间的名称冲突
- `MCPManager` 支持多 MCP 适配器的生命周期管理、统一 schema 暴露与按前缀路由工具调用
- MCP 工具执行结果通过 `MCPToolResult` 结构返回，支持多内容块（text/image/resource）与错误标记
- MCP 集成测试友好：通过 `mockMCPClient` 可在不依赖真实 MCP 服务端的情况下验证工具发现、注册与执行逻辑
- 后续扩展 MCP 资源/提示/客户端生命周期管理时，应在现有 `MCPClient` 接口基础上增量扩展，保持向后兼容
- `internal/agent/mcp_client.go` 的 `NewStdioMCPClient` 是生产态 stdio MCP 客户端构造函数：内部用 `mark3labs/mcp-go/client.NewStdioMCPClient` 启动子进程，并立即调用 `Initialize` 完成协议握手（`ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION`）；握手失败时关闭子进程并返回错误，调用方不应重试同一 command
- MCP 服务器配置位于 `AppSettings.MCP.Servers`（类型 `[]MCPServerConfig`），由 `configs/packetmind.json` 的 `mcp.servers` 数组持久化；每条配置包含 `name/command/args/env/enabled` 五个字段；`enabled: false` 的条目在 `app.go` `initMCPManager()` 中跳过，不启动子进程
- `app.go` 的 `initMCPManager(ctx, mcpCfg)` 遍历已启用服务器，依次 `NewStdioMCPClient` → `manager.AddAdapter` → `manager.RegisterAll`；任一步骤失败时 best-effort 记录日志并继续下一条，不中断应用启动；所有适配器均失败时返回 `nil`，Agent 正常以无 MCP 模式运行
- Agent 结果需包含最终结论、trace chain、工具调用次数与提前停止原因，供 WebSocket/UI 展示推理过程；`Agent.Analyze()` 不应再手动注入合成 `start` trace/event，事件流应直接反映 runtime 真实产出的 thought/action/observation/final 等事件
- Agent provider chat 响应的 `usage` 字段需解析并映射到 `LLMChatResponse.Usage`；`Analyze()` 需跨路由决策与 ReAct 循环累计到 `AgentResult.TokenUsage`，绑定层 `agent_result` 事件与 `AgentAnalysis.TokensUsed` 应同步输出总 token（无 usage 时为 0）
- AI provider 调用链当前统一通过 `internal/agent/retry.go` 执行有界重试：共享 `doProviderChat()` / `doProviderChatStream()` 以及 legacy `Complete()` / `Stream()` 路径都应对 HTTP `429/500/502/503/504` 与瞬时网络错误做退避重试；重试耗尽后返回 `CategoryRetryable` 的 `AgentError`，非重试型 4xx 应尽早返回 `CategoryFatal`
- `retryProviderCall()` 支持可选 `RetryNotifyFunc` 回调，在重试间隔前通知调用方当前重试进度；`ConversationRuntime.Run()` 通过 `context.Value` 注入 notifier，在每次 LLM 调用的重试事件中自动发射 `provider_retry` 类型的 `AgentEvent`（含 `retry_attempt` / `retry_max` 字段），前端据此展示实时重试状态
- 前端 AI 面板在 agent trace 时间线中渲染 `provider_retry` 事件为黄色旋转动画卡片，显示 `🔄 正在重试第 N/M 次 (Provider operation)` 格式文案；`useWailsEvents.ts` 与 `agentStore.ts` 透传 `retry_attempt` / `retry_max` 字段
- Agent ReAct 在无 tool calls 的文本直答步骤应优先走 `LLMClient.ChatStream(ctx, req)` 并通过 `text_delta` 事件增量上报内容；若 provider 不支持或流式失败，必须回退到阻塞 `Chat()` 结果，且工具调用步骤保持阻塞 `Chat()` 语义不变
- 前端 `gui/src/composables/useWailsEvents.ts` 在桥接 `agent:analysis` 时必须将 `text_delta` 作为标准 `agent_event` 透传，并确保其 `content` 优先取 `agent_event.content`（缺失时回退顶层 `content`）；`agent_result` 分支需透传 `token_usage` 与 `tokens_used` 给 Agent store；`gui/src/stores/agentStore.ts` 需在 `text_delta` 事件中增量拼接 `currentContent`，并同时维护 `tokenUsage`（明细）与 `tokensUsed`（总量回退）在按次分析中的重置与结果落盘状态
- 前端 `gui/src/stores/agentStore.ts` 的消息模型需保留 `specialistName`（映射自 `agent_event.specialist_name`），供 AI 面板在思考时间线中展示专家来源徽标
- 前端 AI 面板中的 trace 可见性应直接从真实消息时间线派生（如 `agent_thought` / `agent_action` / `agent_observation` / `agent_error` / `agent_decision`），不要依赖未维护的独立 trace 缓存状态
- Agent 流式进度通过回调上报，业务层如需广播给前端，应复用现有 WebSocket Hub
- 对 `response_headers` 目标的分析应优先经过一个无工具的路由决策阶段，由模型判断当前问题适合 `direct` 直答还是 `trace` 深入追踪；路由不确定时默认进入追踪
- 对“某值从哪里来的”这类跨请求问题，应优先使用结构化 provenance tracing：先从目标请求提取 query/header/cookie/json/form 参数，再回溯同会话更早响应中的 header/set-cookie/json 值，并结合时间顺序、会话、Host 与字段语义做置信度排序
- 前端 AI 面板需要展示 Agent 的思考/决策/工具调用过程；分析进行中输入框仍可录入新问题，但应进入排队态，并通过停止按钮允许用户中断当前分析
- 前端 AI 面板底部输入区应始终支持自由问答，不应因当前存在选中请求而自动切换为 request-aware tracing 模式；针对具体请求/字段的上下文分析应通过显式请求分析入口触发
- 前端 AI 面板中的自由问答应复用稳定 chat session，以便后续追问继承前序问答上下文；provider 支持流式时应尽量增量展示回复内容，并允许用户折叠/展开 thought/action/observation 过程
- 右侧 AI 面板中的显式字段/请求分析入口与底部自由问答输入区应共享同一个稳定 chat session；同一面板内的后续追问必须能继承前序字段分析结论，而不是落入独立 memory
- AI Agent 对话的 `chatSessionId` 现直接绑定到抓包会话 ID（capture session ID）；`agentStore.bindSession(id)` 在应用初始化时将 agent 上下文绑定到当前抓包会话，`agentStore.switchSession(id)` 在用户切换会话时保存当前消息快照并恢复目标会话历史；前端 `SessionSnapshot` 内存缓存按 session ID 键控 messages/sequence/tokenUsage
- `internal/agent` 的请求读取、diff、搜索结果与初始请求快照必须优先使用当前分析会话 `SessionID`（或请求自身 `SessionID`）而不是 `GetActiveSession()`；“正在查看/分析的会话”与“当前激活用于录制的会话”是两套状态，工具层不得再把两者混淆
- 前端 `gui/src/stores/agentStore.ts` 的 `SessionSnapshot` 必须同时持久化该会话的完整瞬时分析状态（如 `streaming`、`analysisId`、`currentContent`、`agentEvents`、`currentToolCalls/currentDepth`、`relatedRequestIds`、`traceExpanded`、`provenanceChain`、`pendingQueue`），以支持在流式分析进行中切换抓包会话而不丢失各自的 AI 面板上下文
- `AgentAPI.GetChatHistory(sessionID)` 返回 `ChatMessageDTO[]`（role/content/created_at），并且必须直接从 storage `chat_message` 表读取；生产代码不得再为读取历史创建任何 `SessionMemory` / LRU cache 项
- `AgentAPI.ClearSessionMemory(sessionID)` 仅为兼容既有 Wails 方法名保留；当前语义是清除指定会话在 storage `chat_message` 中的持久化对话历史。`storage.DeleteSession()` 已在事务中级联删除 chat messages，`app.go` 不需要再额外挂删除清理回调
- `AgentAPI.recordSessionInteraction(...)` 只负责把 user/assistant turn 持久化到 storage `chat_message` 表；bindings 不再维护会话级热缓存或预热逻辑
- `agent:analysis` 事件 envelope 需继续保持前端兼容：顶层 `analysis_id` / `done` / `content` / `agent_event` / `agent_result` / `error` / `cancelled` 结构不可破坏；多专家/协作路径已移除；`specialist_name` 字段保留 `omitempty` 以兼容旧前端缓存，则 `specialist_name`、`routing_reason`、`routing_confidence`、`handoff_count`、`specialist_chain`、`final_specialist` 可为空或省略
- 前端 `gui/src/composables/useWailsEvents.ts` 处理 `agent:analysis` 时，必须优先依据后端 payload 中的 `session_id` 调用 `agentStore.routeEventToSession(...)` 路由事件；非当前可见会话的流式事件应静默写入对应 session snapshot，而不是丢弃或污染当前面板
- 前端 AI 面板中的 Agent reasoning / action / observation / decision、related requests 与 provenance chain 应尽量并入同一条时间线按出现顺序流式展示，避免在最终结论输出后再把支撑性上下文突兀地插到消息底部
- 前端 AI 面板在单次分析完成后应展示 token usage（至少 input/output/total），并且不应在同一条回复下重复渲染多份统计条
- 直连自由问答（direct chat）与 agent_result 结束事件应始终携带 `tokens_used` 标量回退值；当详细 `token_usage` 缺失时，前端仍需持久化并展示总 token 消耗
- 直连自由问答在非流式 completion 路径中应优先透传 provider usage：`Complete()` 返回 usage 时，绑定层 `agent_result` 必须同时输出 `token_usage` 与 `tokens_used`；流式路径在无法稳定提取 usage 时可继续使用 `tokens_used=0` 安全回退
- 当前消息主时间线已展示 agent 步骤时，应确保 trace 元素（思考/工具/观测）在主时间线中可见
- 当前消息主时间线已展示 agent 思考步骤时，不应再额外渲染第二套重复的思考面板；若需要折叠/展开，应直接作用于主时间线中的连续 trace 片段
- 前端 AI 面板中的思考过程在视觉上应保持为单一、一体化的 thought card；默认态应收起并保留最近一条 trace 的预览，而不是把“显示/隐藏思考过程”按钮与正文拆成两个割裂模块
- token usage 展示应优先放在 AI 面板底部、输入区附近等稳定可视区域；不要仅绑定在长回复正文底部导致统计信息被滚出视窗
- 当 provider 未返回详细 usage 且界面仍需给出成本感知反馈时，前端 AI 面板可基于最终回答文本长度展示明确标注为 `estimated` 的 total token 回退值；不要只显示无信息量的 `N/A`
- Agent 工具执行失败的 observation 事件需透传结构化错误元数据（`error_category`、`error_tool_name`、`error_timeout`、`error_recovered`）；Wails 绑定层 `agent:analysis` 的 `agent_event` 与 `error` 事件都应保留这些字段，前端需据此区分通用 observation 与 `agent_error` 展示
- 前端 AI 面板中的 Agent trace/message 列表项需使用稳定唯一 ID，避免因重复 key 导致推理步骤内容渲染丢失
- 前端 AI 面板中来自 related requests / provenance chain 的请求跳转操作，必须确保目标请求详情已载入 store；若当前列表未包含该请求，也应按 request id 拉取后切换中间详情视图
- 前端 AI 面板应保证底部输入区在窄面板与长消息场景下始终可见；消息列表负责内部滚动，不允许因 flex 收缩约束缺失把输入区挤出可视区域
- 前端弹窗/表单若需要基于 props 构造可编辑草稿，禁止直接对 Vue props/代理对象执行 `structuredClone`；应先转为普通对象（如 `toRaw` 后再序列化）以避免首屏挂载阶段触发 `DataCloneError`
- 桌面前端中的确认/删除交互应优先使用应用内 modal/dialog 组件，不要使用浏览器原生 `confirm()`，以避免 Wails 环境下出现系统样式弹框破坏桌面一致性
- 若 modal/dialog 在 `loading` 态下阻止用户主动取消，成功提交后的关闭逻辑不得复用同一个受 guard 限制的 cancel handler；应直接在成功路径复位弹窗可见性与相关草稿状态，避免“操作成功但弹窗不消失”

### 模型配置管理

- 运行时模型列表以 `configs/models.json` 为本地持久化源；Wails 绑定层 `AgentAPI.ListModels()` 返回 `models`、`grouped`、`default_model`、`active_model`、`active_provider`，其中 grouped 的 `provider_name` 必须使用 provider 显示名（如 `Ark`）而不是 provider ID（如 `zhipu`）；前端模型选择器初始化必须优先采用有效 `active_model`，其次仅在 `default_model` 真实存在于返回列表时才回退使用，否则稳定回退到首个可用 grouped/flat model，禁止继续保留失效的旧 model id
- `ModelsConfig` / `ModelsStore` 不得硬编码 `zhipu` 作为 active provider 默认值；当 active provider 为空时应回退到第一个已配置 provider，provider 列表为空时保持为空并让调用链按无 provider 处理
- Wails 绑定层 `AgentAPI` 应提供 `ListProviders()`，通过 `ModelsStore.GetProviders()` 返回 provider 元数据、是否已配置 API Key/Base URL、模型数量与 active 状态；`ListModels()` 需同时返回扁平 `models` 与按 provider 分组的 `grouped` 数据
- provider 列表中的交互能力（如是否允许删除）必须通过后端 `ListProviders()` 返回的元数据字段下发，前端禁止对特定 provider id（如 `mock/openai/zhipu`）做硬编码特判
- 手动模型维护通过 `ConfigAPI.UpsertModel()` / `DeleteModel()` 写入 `configs/models.json`，`ModelConfigModal.vue` 不应再导入或调用已移除的 `fetchDynamicModels` / `FetchModels` 动态发现路径；配置弹窗中的模型初始化/刷新只能使用当前 Wails 暴露的 `AgentAPI.ListModels()` 与 `ConfigAPI.UpdateAgentConfig()`
- `ModelConfigModal.vue` 应允许用户手动添加/编辑/删除模型条目（Model ID、Display Name、Context、Output），并在同一弹窗保存当前 provider、API Key、Base URL、active model、max tokens 与 temperature；模型 CRUD 后必须刷新 `AgentAPI.ListModels()` / `ListProviders()` 本地状态；弹窗不应再提供 `Refresh Configured Models` 这类动态发现/刷新按钮
- `ModelConfigModal.vue` 的 Channel / Model 操作按钮应贴近 PacketMind 桌面 inspector 风格：低圆角、灰阶边框、紧凑尺寸、列表内 inline actions，避免 Ant Design 默认蓝色/link 按钮造成 Web 风格突兀
- `ModelConfigModal.vue` 应允许用户手动新建、编辑、删除 LLM channel/provider，新增 provider 统一使用当前后端支持的 `openai-compatible` API type，并通过 `ConfigAPI.UpsertAgentProvider()` / `DeleteAgentProvider()` 持久化 provider ID、显示名、API Key 与 Base URL；删除能力仍以后端 `ProviderInfo.deletable` 为准
- `ModelConfigModal.vue` 的 API Type 字段应使用 select/combobox 呈现；当前只有 `openai-compatible` 一个选项，但 UI 需保留后续新增 API 类型的扩展位置
- `ConfigAPI.GetAgentProviderKey(provider)` 必须返回 `api_key`、`base_url`、`api_type` 三个字段，供前端模型配置弹窗恢复 provider 草稿；不要只返回 API Key。
- 模型配置现采用 per-provider 存储：`configs/models.json` 中 `Models` 字段已从 `ModelsConfig` 顶层移除，改为每个 `ProviderConfig` 持有自己的 `Models map[string]ModelConfig`。每个 LLM channel 的模型列表独立管理，编辑一个 provider 的模型不会影响其他 provider。
- `ConfigAPI.UpsertModel(provider, id, model)` / `DeleteModel(provider, id)` 现在需要显式传入 provider ID；`ModelsStore.UpsertModel` / `DeleteModel` 内部按 provider 路由到对应 `Providers[i].Models` map。
- `ModelsConfig.GetModelByID(id)` 遍历所有 provider 搜索，`GetModelByProviderAndID(provider, id)` 只搜索指定 provider。
- `AgentAPI.ListModels()` 返回的 `grouped` 数据中每个 provider 组只包含该 provider 自身的模型列表，不再聚合全局模型池。
- `ModelConfigModal.vue` 中模型 CRUD 操作（添加/编辑/删除模型）自动关联当前选中的 provider（`selectedProviderId`），不会跨 provider 误操作。
- 后端 `normalize()` 初始化每个 provider 的 `Models` map（如为 nil 则分配空 map），确保序列化后 models.json 中每个 provider 始终有 `models` 字段。

---

## 项目结构约定

```
internal/
├── api/
│   └── bindings/     # Wails 前端绑定
├── proxy/            # 代理核心逻辑
├── storage/          # 内存存储（go-memdb 多索引）
│   ├── session.go    # Storage 结构体 + Session CRUD
│   ├── request.go    # Request CRUD + 过滤/排序
│   ├── schema.go     # memdb schema + record 类型 + 转换函数
│   ├── global.go     # storage.Default 全局单例
│   ├── search.go     # FindInSession + 关键字搜索
│   ├── provenance_query.go # 值溯源查询
│   ├── export.go     # JSON 导入/导出
│   ├── har.go        # HAR 类型与转换
│   ├── helpers.go    # 深拷贝/ID 生成/通用辅助
│   ├── models.go     # 数据模型定义
│   ├── decode.go     # body 解码（gzip/deflate/brotli）
│   └── provenance.go # 溯源类型与 artifact 提取
├── agent/            # AI 分析引擎（根 facade + runtime/llmcore/provider 子包）
├── updater/          # 基于 go-selfupdate 的应用自更新
└── config/           # 配置加载

app.go                # Wails 主入口

gui/                   # Vue 3 前端
├── src/
│   ├── components/   # Vue 组件
│   ├── stores/       # Pinia 状态管理
│   ├── composables/  # Vue composables
│   └── api/          # Wails API 客户端
configs/               # 配置文件（packetmind.json / models.json）
docs/                  # 文档
```

### Wails 绑定约定

- `internal/api/bindings/` 中的导出 API 方法应统一返回 `SessionResponse`，保持与现有前端响应处理兼容
- 绑定层方法签名不得依赖 `gin.Context`，应通过显式参数传入请求数据
- 配置读取接口必须默认脱敏，禁止返回 `api_key`、代理密码等敏感字段
- Session/Request 绑定需对齐响应码语义（成功 `code=0`；错误码保持 `40001`、`40002`、`50001`）
- Session/Request 绑定方法需保持值返回模式（`SessionResponse` 非指针），避免改变前端调用约定
- `ConfigAPI.UpdateDesktopSettings` 在涉及运行时代理设置时应先调用 proxy 应用（可失败），再持久化到 `packetmind.json`；持久化失败时需 best-effort 回滚 proxy 运行态设置
- 若绑定返回结构包含 `time.Time` 或嵌套包含 `time.Time` 的 storage/model 类型，必须先转换为 Wails-facing DTO（字符串时间字段），避免 `wails dev` 生成绑定时出现 `Not found: time.Time` 警告
- 不要把仅用于内部事件桥接的应用方法暴露到 `Bind` 列表；否则其参数/返回类型也会进入 Wails 绑定生成范围
- 仅用于应用装配/运行时注入的 helper（如 Wails context 注入、删除回调注册、内部 sessionDir 配置）应优先做成 package-level/internal helper，而不是直接挂在导出 API 类型上；否则 `wails generate module` 会生成多余甚至不安全的前端绑定

### Wails 基础工程约定

- 根目录 `wails.json` 为 Wails 构建配置入口；`frontend:dir` 指向 `gui/`
- 根目录 `app.go` 为 Wails 主入口，负责应用生命周期（startup/shutdown）、代理启动以及通过内部事件桥接向前端广播 `request:new`
- 请求列表前端当前依赖双阶段事件更新：`request:new` 用于插入初始占位请求，`request:complete` 用于按 request id 原地替换同一条记录；新增请求事件时不得破坏这一增量更新语义
- `internal/api/bindings/request.go` 中的主动请求发送入口（如 `ReplayRequest`、`ComposeRequest`）也必须遵循同样的双阶段录制约定：发送前先落库并发出 `request:new`，完成或失败后按同一 request id 更新记录并发出 `request:complete`；失败请求需保留 `502` + `Error` 文本，且在无 active session 时创建默认会话后再记录
- `internal/api/bindings/request.go` 当前还提供 `FindInSession(opts storage.FindInSessionOptions)` Wails 绑定，供前端按会话执行全文检索；返回值应保持 `SessionResponse{code,data}` 结构兼容，匹配结果避免直接暴露 `time.Time`，优先返回字符串时间字段与基础请求元信息
- 前端 `gui/src/components/RequestList.vue` 的请求叶子节点右键菜单当前包含 `Compose` 动作；`gui/src/App.vue` 负责承接 `composeRequest` 事件并打开 `ComposeModal.vue`，该弹窗需预填 method/url/headers/body，发送时通过 `requestApi.compose()` 调用 `RequestAPI.ComposeRequest`，并依赖后端两阶段 `request:new` / `request:complete` 事件驱动请求列表更新
- `internal/proxy` 请求录制当前采用两阶段落库：`saveRequestStart()` 在上游调用前保存初始请求快照并触发 `onRequest/request:new`，`saveRequestComplete()` 在响应捕获完成后按同一 request id upsert 完整记录并触发 `onComplete/request:complete`；终止型错误与原始 SOCKS5 tunnel 仅允许走 start 阶段，不得再额外发送 complete 事件
- `internal/proxy` 不再在 `Proxy` 实例上持有 `*storage.Storage`；代理录制路径统一通过全局 `storage.Default` 访问存储，`app.go` 需在创建 storage 后先设置 `storage.Default` 再构造 `proxy.New()`
- `internal/api/bindings` 当前已切换为全局单例装配：`SessionAPI/RequestAPI/AgentAPI/ConfigAPI/ProxyAPI` 构造函数均不再接收 store/proxy/models/settings 参数，运行时统一直接读取 `storage.Default`、`proxy.Default`、`config.DefaultModelsStore`、`config.DefaultSettingsStore`
- `internal/api/bindings/agent.go` 与 `updater.go` 的事件广播上下文统一直接读取 `internal/appctx.Ctx`；`ConfigureAgentRuntime`、`ConfigAPI.SetModelsStore`、`ConfigAPI.SetSettingsStore`、`UpdaterAPI.SetContext` 等仅用于注入全局依赖的 helper 已删除，`app.go` startup 只需调用 `appctx.Set(ctx)` 并在主初始化阶段设置相关 global singleton
- `internal/api/bindings/agent.go` 当前在 `agent.NewAgentFromProvider(...)` 后必须显式调用 `SetStore(storage.Default)`（以及如有外部 MCP 时再调用 `SetMCPManager(...)`）完成工具执行链装配；不要假设 `NewAgent()` / provider factory 会自动绑定 store 或 builtin tools
- 涉及上述 package-level globals 的单元测试也必须先显式设置对应全局（如 `storage.Default`、`proxy.Default`、`config.DefaultModelsStore`、`appctx.Ctx`）再创建 `Agent` / `Proxy` / bindings；测试中禁止继续使用已删除的带依赖参数构造函数或访问 `Agent.store`、`Proxy.storage`、`AgentAPI.ctx` 等已移除字段
- `build/` 下的 `appicon.png`、`windows/icon.ico`、`darwin/icon.icns` 为应用图标资源
- AI 相关 Wails 绑定事件统一使用 `agent:analysis`；流式过程应通过 `agent_event` 增量广播，结束时通过 `done=true` + `agent_result` 或 `error` 收敛
- 自更新相关 Wails 绑定由 `internal/api/bindings/updater.go` 提供；`UpdaterAPI.SetContext()` 负责注入运行时 context，并通过 `update:progress` / `update:done` 向前端广播下载进度与完成事件
- Wails `options.App` 必须配置有效的 `AssetServer`（生产构建至少提供嵌入的 `gui/dist` 资源），否则运行时会报 `AssetServer options invalid`
- 桌面窗口当前采用 `Frameless: true` 无边框模式；前端标题栏拖拽区域必须使用 Wails CSS 属性 `--wails-draggable: drag`，可交互元素必须显式使用 `--wails-draggable: no-drag`
- 当前桌面窗口控制（最小化 / 最大化 / 关闭）由前端通过 Wails runtime API 驱动；Windows/Linux 使用自定义按钮，macOS 保留原生 traffic lights
- Frameless 模式下当前依赖 Wails/OS 原生边缘缩放能力（未额外实现自定义 resize hot zones）；前端仅提供右下角视觉 resize grip 作为提示，不应劫持原生缩放行为
- 应用启动时应优先按主屏幕尺寸计算默认窗口大小与居中位置；若启动阶段无法可靠获取屏幕信息，则回退为较大的专业工作窗口尺寸并居中，而不是使用偏小的固定窗口
- `wails dev` 开发模式下 Vite 偏好端口 3000 但允许自动递增（`strictPort` 已禁用）；`wails.json` 中 `frontend:dev:serverUrl` 设为 `auto`，`viteServerTimeout` 为 30 秒；Playwright E2E 测试固定使用端口 3000（`gui/playwright.config.ts`），两者互不干扰

### 前端交互约定

- 抓包列表顶部工具条默认保持精简，优先保留搜索、总数统计与展开/折叠操作
- 抓包列表顶部工具条在窄宽度下也应优先保持单行布局；`Total`、`Expand All`、`Collapse All` 不应因容器压缩而断成多行
- 抓包列表关键字过滤应对历史拉取与 Wails Events 实时新增请求保持一致，实时请求必须遵循当前过滤条件，不得插入污染过滤结果
- 抓包列表搜索输入应在内容变化时即时驱动过滤更新，避免仅依赖 Enter/搜索按钮触发
- 抓包列表空状态文案不得硬编码具体代理端口；若端口可能变更，应使用通用提示或动态端口来源
- 抓包列表 Host/路径/请求节点在实时新增请求到达时应提供短时高亮反馈，并在短时间内自动淡出清理，不影响现有树结构与选中行为
- 非核心筛选项（如 Method 下拉筛选）应谨慎添加，避免破坏列表工具条的紧凑性与可读性
- 整体前端视觉优先贴近桌面抓包工具：更高信息密度、更低圆角、更强边框/分隔线、减少卡片化 Web 风格
- 顶部标题栏中的高频主操作按钮应使用明确语义色：启动类操作优先绿色，停止/关闭当前运行态操作优先红色；避免继续沿用 Ant 默认蓝/灰导致状态感弱
- 桌面无边框标题栏可承载 Charles 风格顶部应用菜单；菜单交互区域必须显式使用 `--wails-draggable: no-drag`，避免与窗口拖拽区域冲突
- 顶部应用菜单当前已支持二级子菜单（如 `Open Recent`、`Focused Hosts`、`Windows/macOS Proxy`、`Auto Save`）；新增菜单项时应优先复用现有 action string 路由到 `App.vue`，避免在菜单组件内直接堆叠业务逻辑
- 顶部应用菜单中的布尔状态项应优先清晰表达当前状态；若某项本质上是设置入口（如 `External Proxy Settings...`），可直接在该设置项上显示勾选状态而不必额外新增重复 toggle；已在顶部工具区提供的高频控制（如 Recording）不应在 Proxy 菜单中重复放置等价开关
- Session 列表左键点击仅用于切换当前查看的会话内容，不得隐式调用激活接口修改 `is_active`；激活会话应仅通过右键菜单中的显式 `Activate Session` 动作触发
- 当前前端 Session 列表需要区分“正在查看的 session”和“当前激活用于录制的 session”：前者可由前端本地 `selectedSessionId` 管理并用于高亮，后者继续由后端 `is_active`/`activeSession` 表示；不要把两者混为同一状态
- Session 列表当前查看态与激活态在视觉上也必须清晰区分：查看态优先通过列表高亮/选中标识表达，激活态通过独立 badge 或状态点表达，允许同一条目同时呈现两种状态
- `Proxy -> Web Interface Settings...` 必须打开专用 `web-interface` 设置弹窗，而不是回落到通用 `proxy` 设置视图
- 抓包列表应优先采用 Charles 风格结构视图：先按 Host 分组，再按路径目录逐层折叠；可拆分的路径段不要平铺成单行表格
- 抓包列表树在 Charles 风格下应优先减少无意义层级：单请求路径在无共享多级前缀时应直接渲染为叶子节点，仅在多个请求共享有意义路径前缀时创建 folder 节点
- 抓包列表节点应为不同协议/载荷类型提供前置图标提示，至少区分 HTTP、HTTPS、WebSocket、Protobuf、SOCKS/Socket、普通文件夹节点
- 抓包列表中的“进行中”请求应显式区别于失败请求：当前约定以 `status_code===0 && !error && duration===0` 判定 pending，并在树节点中显示蓝色加载图标/文字与轻量 spinner；完成后恢复既有状态码与耗时展示
- 抓包列表中的失败/异常请求（如 `status_code=0`、unknown/aborted 类节点）应具备明显的异常图标与弱化文本/状态样式，视觉上需与普通成功请求清晰区分
- 抓包列表请求树节点右键菜单应沿用 `SessionList.vue` 的 Teleport 桌面菜单模式：菜单固定挂到 `body`、按光标定位并做视口边界修正，点击外部/滚动/Escape 关闭；请求叶子节点提供 Copy/Save/Repeat/Agent Analyze/Delete，Host/Folder 节点提供 Copy Host 与整体展开/折叠操作
- 请求详情、抓包列表、AI 面板应尽量保持统一的灰阶工具栏/标签栏语言，避免单独出现强烈 Web 组件感
- 请求详情 Overview 视图应优先承载 Charles 风格基础元信息：URL、Method、Status、Response Code、Content-Type、Client/Remote Address、Protocol、Keep Alive、SSL、Connection、Timing、Size
- 请求详情 SSL 视图应优先承载 Charles 风格 TLS 分组信息：Protocol、Session Resumed、Cipher Suite、ALPN、Server Certificates、Extensions；无法可靠获取的 ClientHello 原始能力字段应显示为 `-`，不要伪造
- 请求详情 Summary 视图应优先承载 Charles 风格单请求摘要表格：Resource、Host、Code、Mime Type、Header、Body、Time，并包含 Total / Grand Total / Duration 汇总行
- 请求详情对 WebSocket 握手请求应在顶层额外显示 `WebSocket` 标签；当前最小实现按单连接展示捕获到的双向 frame/message 列表（Direction / Type / Size / Time / Message），抓包列表中的对应请求节点也应显示 WebSocket 图标
- 请求详情 Contents 视图中的 Request/Response Body 在 payload 可解析为 JSON/XML 时应额外提供 `Tree` 标签页；不可解析时不展示该标签页，并保留既有 Text/Hex/Raw 行为
- 请求详情 Contents 视图中的底部标签应按数据可用性动态显示：Request 侧仅在存在对应内容时显示 `Query String` / `Cookies` / `Text` / `Hex`；Response 侧仅在存在 body 时显示 `Text` / `Hex`；`Raw` 仍保留用于查看原始报文
- 请求详情 Response Body 的 `Text` 视图应优先显示文本类型响应（如 `text/*`、JSON、XML、SVG、脚本等）的**原始解码文本**，而不是 Base64 串；JSON 响应应额外提供 `JSON Text` 标签用于展示格式化后的可读 JSON；对 HTML/XHTML 响应应额外提供 `HTML` 标签用于沙箱预览；对图片类型响应（`image/*`）应额外提供 `Image` 标签用于直接预览
- 绑定层向前端传输 request/response body 时，应优先输出可读原文：若 `Content-Type` 为文本类且 `Content-Encoding` 为 `gzip`/`deflate`/`br`，应先解压，再结合 `Content-Type` 中的 `charset` 做 UTF-8 转码后返回文本；对无法安全解码或未知字符集/编码的内容必须回退为 Base64，避免前端直接展示压缩字节垃圾文本
- 抓包记录模型已支持 `request_start_time` / `request_end_time` / `response_start_time` / `response_end_time`、`connect_duration`、`tls_handshake_duration`、`request_duration`、`response_duration`、`latency_duration`、`client_addr`、`server_addr` 等字段，新增详情展示时应优先复用这些字段
- 抓包记录模型已扩展支持 `tls_server_name`、`tls_did_resume`、`tls_alpn`、`tls_curve_id`、`tls_ocsp_stapled`、`tls_sct_count`、`tls_server_certificates`、`tls_server_extensions` 等 SSL 细节字段，SSL 详情页应优先复用这些结构化数据
- 代理错误路径约定：凡代理链路中出现已记录日志的失败分支，若不会进入既有成功记录流程，必须同步落库请求记录并填充 `Error` 字段；最小化错误记录应优先复用 `Proxy.recordMinimalError(...)`，并遵循状态码约定（上游失败 502、协议/连接失败 0、拒绝访问 403、代理内部错误 500）
- 代理转发响应头时必须先移除 hop-by-hop 头（如 `Connection`、`Keep-Alive`、`Transfer-Encoding`、`Upgrade` 及 `Connection` 声明的附加头），HTTP 与 HTTPS MITM 路径都要遵守该规则
- HTTPS MITM 在读取请求体用于录制后，必须恢复 `req.Body` 再转发到上游；MITM keep-alive 仅在 `req.Close` 或显式 `Connection: close` 时才终止连接，不能以缺少 `keep-alive` 误判关闭
- HTTPS MITM 应在单个 TLS client connection 生命周期内复用上游 `http.Client`/`Transport`；当上游请求失败时必须立即向客户端返回 `502 Bad Gateway`，避免浏览器/客户端悬挂等待
- CONNECT 处理必须先确认 `http.Hijacker` 可用并成功 hijack 原始连接，再向客户端写入 `200 Connection Established`；禁止在 hijack 前通过 `ResponseWriter` 提前发送 200
- 代理超时策略约定：共享 HTTP client 总超时为 120s，MITM 上游 client 总超时为 60s，`ResponseHeaderTimeout` 保持 30s，`http.Server` `ReadTimeout` 为 60s，`WriteTimeout` 关闭以兼容长时间流式响应

## 常见任务

### 添加新的 Wails 绑定

1. 在 `internal/api/bindings/` 创建或编辑绑定文件
2. 实现绑定方法，返回 `SessionResponse` 类型
3. 在 `app.go` 中注册绑定
4. 运行 `wails generate module` 生成前端绑定
5. 更新 CHANGELOG.md

### 添加新的数据模型

1. 在 `internal/storage/models.go` 定义结构体
2. 在对应职责文件（`session.go`/`request.go`/`search.go`/`provenance_query.go`/`export.go`）中添加方法
3. 如果需要深拷贝辅助，在 `helpers.go` 中添加
4. 如果需要暴露给前端，在 bindings 中添加对应 API
5. 更新 CHANGELOG.md

**注意**: 当前使用 In-Memory 存储，- 数据存储在进程内存中（重启后清空）
- 支持通过 `ExportAll()` / `ImportAll()` 进行数据导入/导出
- 使用 `sync.RWMutex` 保证线程安全

### 集成新的 AI 提供商

1. 在 `internal/agent/` 创建新文件 (如 `claude.go`)
2. 实现 `Complete()` 和 `Stream()` 方法
3. 在 Agent 使用的 provider 工厂/分支中接入新 Provider
4. 在 `configs/models.json` 中添加新模型配置
5. 更新 CHANGELOG.md

### 添加新的 AI 模型

1. 编辑 `configs/models.json`，在 `models` 数组中添加新模型：
   ```json
   {
     "id": "new-model-id",
     "name": "Model Display Name",
     "provider": "zhipu|openai|mock",
     "description": "模型描述",
     "max_tokens": 4096,
     "supports_streaming": true
   }
   ```
2. 如需新 Provider，在 `provider_api_keys` 中添加 API Key 配置
3. 更新 CHANGELOG.md

---

## 变更记录要求

**每次代码改动后必须**:
1. 在 `CHANGELOG.md` 中记录变更
2. 如涉及架构/约定变更，同步更新本文件

### 代码组织示例

- 为提高可维护性，`internal/proxy/proxy.go` 可按功能拆分为多个源文件（例如 `transport.go` / `cert.go` / `socks5.go` / `websocket.go` 等），各文件保持 `package proxy` 并侧重单一职责：transport 负责连接/上游拨号、cert 负责 CA/签发证书、socks5/websocket 负责协议解析与帧捕获。此类变更应在 CHANGELOG.md 中记录并在 PR 描述中说明拆分范围。

### CHANGELOG.md 格式

```markdown
## [版本号] - YYYY-MM-DD

### 新增功能
- 功能描述

### 修复问题
- 问题描述

### 代码改进
- 改进描述

### 文档更新
- 更新描述
```

---

### 上下文弹窗绑定

- `GetSessionContext(sessionID)` Wails 绑定返回 storage-backed `SessionContextStats`（`session_id`、`has_history`、`message_count`、`estimated_tokens`、`active_model`、`active_provider`、`active_max_tokens`、`active_temperature`、`available_models`）
- 前端 `ContextModal.vue` 组件在弹窗打开时通过 `agentStore.fetchSessionContext()` 拉取数据；避免在普通响应式重算中重复触发同一请求
- 圆形按钮位于 AI Analysis 行，点击后打开"上下文"弹窗
- `ContextModal.vue` 应优先采用紧凑桌面工具布局：顶部为摘要/统计卡片，底部为左右分栏 inspector；左侧只负责选择消息，右侧常驻显示带行号的 JSON/原始结构查看器（标题行显示 `role • message id`），避免继续使用整行展开式原始消息框或过大的整屏弹窗
- `ContextModal.vue` 在桌面环境中应提供真实可拖拽的窗口尺寸调整能力；优先使用组件内部维护的 `width/height` 响应式状态 + 明确可见的右下角 resize handle，不要依赖未经验证的 UI 库隐式 `resizable` 属性或仅靠 CSS 猜测实现

### 智能滚动 UX

- `AgentPanel.vue` 使用 `isAutoFollowing` ref 追踪用户是否在消息底部
- 用户向上滚动超过 50px 时停止自动跟随，显示浮动"跳到最新"按钮
- 点击按钮恢复自动跟随并强制滚动到底部

### AI 面板思考过程渲染

- `AgentPanel.vue` 将连续的 trace 消息（`agent_thought`/`agent_action`/`agent_observation`/`agent_error`/`agent_decision`）按 ReAct 循环分组为 `ReActCycle` 结构
- AI 右侧面板标题当前统一使用 `AI AGENT`，避免回退为更泛化的 `AI Analysis`
- 每个 cycle 包含：thought（可选）、action（可选）、observation/error（可选）、decision（可选），并带递增 `stepNumber`
- 渲染为垂直时间线：左侧编号圆点 + 连接线，右侧内容区
- Tool call 卡片合并 action + observation 为单个可展开组件，带状态指示器（✓/✗/蓝色脉冲）
- 思考卡片折叠态显示摘要药丸（每步一行：💭 思考 / 🔧 工具名 / 🎯 决策 / ❌ 错误）
- Agent 工作中显示 CSS shimmer "思考中..." 动画
- 思考内容和观测结果支持 Markdown 渲染
- 每个 cycle 可显示执行耗时（基于 thought 与 observation 的 timestamp 差值）
- 多轮对话之间显示带时间戳的水平分隔线
- Tool 卡片展开状态由组件内 `reactive(Map)` 管理，不依赖全局 store 状态

---

## 快速参考

- **Go 版本**: 1.21+
- **模块路径**: `github.com/packetmind/packetmind`
- **Wails 入口**: `app.go`
- **配置文件**: `configs/packetmind.json`（桌面/代理统一运行配置）
- **模型配置**: `configs/models.json`
- **存储介质**: go-memdb 进程内存（重启后清空，可通过导入/导出恢复）
- **CA 证书**: `data/certs/`
- **版本源**: `internal/version/version.go`（通过 `-ldflags` 构建时注入 Version/BuildTime/Commit）
- **自更新**: `internal/updater/updater.go`（基于 `go-selfupdate v1.5.2`，GitHub Releases 为更新源）

---

**最后更新**: 2026-06-25
**维护者**: PacketMind Team

> 文档注记: 2026-04-09 - 已记录对 `internal/proxy` 的小修复（移除未使用导入、修复导入块不一致），并在本地运行 `go test ./internal/proxy` 验证通过。
>
> 文档注记: 2026-04-09 - 已从 `internal/proxy/proxy.go` 中移除对已迁移 helper (`newSharedTransport` / `sharedHTTPClient`) 的过期注释，保持代码注释与实现位置一致。
>
> 文档注记: 2026-04-09 - 已实现 provider 重试实时事件通知：`retryProviderCall` 通过 `RetryNotifyFunc` + `context.Value` 机制在重试时发射 `provider_retry` 事件，前端 AI 面板渲染为黄色旋转动画卡片；所有 `internal/agent` 测试通过。
>
> 文档注记: 2026-04-10 - 已将 `internal/ai` 重命名为 `internal/agent`，并将原 `internal/agent/agent.go` 按职责拆分为 `agent_dispatch.go`、`agent_types.go`、`provider_chat.go`、`provider_adapter.go`，在保持行为不变的前提下提升可读性与可维护性。
>
> 文档注记: 2026-04-10 - 已将 `ConversationRuntime` 的 ReAct 循环从有界迭代（`maxIterations=15` + soft-stop prompt）改为无限循环设计（opencode 模式），循环仅通过 context 取消、模型自停、错误三个出口终止；移除了 `maxIterations`/`WithMaxIterations`/`maxDepth`/`defaultAgentMaxDepth` 等迭代上限机制，保留 `maxToolCalls` 与重复调用检测作为安全网。
>
> 文档注记: 2026-04-13 - 已移除 `ConversationRuntime` 中的 `maxToolCalls`（默认 50）硬性工具调用上限与重复调用拦截机制（签名计数 >2 阻断、累计 >5 停止）；循环现在完全由模型自主决定何时停止，仅保留 context 取消、模型自停、错误三个出口；同步清理 `defaultAgentMaxToolCalls` 常量、`Agent.maxToolCalls` 字段、`WithMaxToolCalls` 选项、`normalizeToolCallSignature`/`normalizeJSONString` 辅助函数及 `tool_budget_exhausted`/`repetition_limit_reached` 停止原因映射。
>
> 文档注记: 2026-04-10 - 已新增 provider registry，并将 `NewAgentFromProvider`、`AIAPI.FetchModels`、`ModelsStore.GetProviders/GetModelsGrouped` 接入统一 provider 元数据与模型发现入口，供多 provider 切换能力复用。
>
> 文档注记: 2026-04-10 - 已完成前端多 provider 模型切换改造：`ModelConfigModal` 重构为左右分栏多 provider 管理界面；`AIPanel` 模型选择器改为按 provider 分组显示；`aiStore` 新增 `providers`/`modelsGrouped`/`loadProviders`/`switchModel`；`types` 新增 `ProviderInfo`/`ModelsByProvider`；新增 `ListProviders` Wails 绑定。
>
> 文档注记: 2026-04-10 - 已将 `AIPanel` 顶部模型切换器从窄宽度 `a-select` 重构为自定义 popover picker：弹层顶部提供搜索输入框与设置按钮，中部按 provider 分组列出模型并展示选中 checkmark；当 grouped 数据缺失时前端仍可基于 flat models 回退分组，避免再次出现仅显示 provider 层级而无法稳定选择具体模型的问题。
>
> 文档注记: 2026-04-11 - 已同步刷新 `docs/` 中文文档中的命名迁移引用，统一当前 `AgentAPI` / `AgentAnalysis` / `storage.Storage` / `gui/` / `/api/agent/*` 入口说明，并修正 AGENTS 中非历史性的旧路径残留。
>
> 文档注记: 2026-04-12 - 已新增 `internal/storage/decode.go` 共享 body 解压逻辑，并将 Agent `previewBody`、storage provenance artifact 提取、request/response body 搜索统一切换为先按 `Content-Encoding` 解压后再解析/匹配，修复压缩正文产生乱码与搜索失效的问题。
>
> 文档注记: 2026-04-12 - 已新增 `internal/updater/updater.go` 与 `internal/api/bindings/updater.go`，使用 `go-selfupdate v1.5.2` 接入 GitHub Releases 自更新、带进度回调的下载包装，以及 `update:progress` / `update:done` Wails 事件广播；`app.go` 已完成 `UpdaterAPI` 构造、注册与 startup context 注入。
>
> 文档注记: 2026-04-12 - 已实现完整自更新功能链：新增 `internal/version/version.go` 统一版本源（通过 ldflags 构建时注入）；新增 `gui/src/composables/useUpdater.ts` 前端更新状态管理；新增 `gui/src/components/UpdateModal.vue` 更新弹窗（checking/available/downloading/complete/error/up-to-date 六态）；`gui/src/App.vue` 接入 UpdateModal 并将 About 弹窗版本号改为动态获取；`.github/workflows/wails-build.yml` 改造为构建 + 打包 + checksum + GitHub Release 一体化流水线；CI 版本已对齐 Go 1.25 + Wails v2.11.0。
>
> 文档注记: 2026-04-13 - 已将桌面/代理配置源从 `config.yaml` + `desktop.json` 合并为单一 `configs/packetmind.json`：`internal/config` 移除 Viper 与旧 `Config/ProxyConfig/CertConfig` 结构，改由 `DefaultPacketMindSettings` / `LoadPacketMindSettings` / `SavePacketMindSettings` 管理统一 JSON 配置；`proxy.New()` 改为仅接收 `storage.Storage`，证书路径与监听设置全部从 `AppSettings.Cert/Proxy` 读取。
>
> 文档注记: 2026-04-13 - 已将 `internal/storage/storage.go`（1388 行）按职责拆分为 `session.go`（Storage 结构体 + Session CRUD）、`request.go`（Request CRUD + 过滤/排序）、`search.go`（FindInSession + 关键字搜索）、`provenance_query.go`（值溯源查询）、`export.go`（导入/导出）、`har.go`（HAR 类型与转换）、`helpers.go`（深拷贝/ID 生成/通用辅助）；原 `storage.go` 已删除。同步移除冗余的 `search_requests_by_host` Agent 工具与 `Storage.GetHistoryRequests` 方法。
>
> 文档注记: 2026-04-17 - 已将 `internal/config/config.go`（1308 行）拆分为 5 个文件：`models.go`（ModelConfig / ProviderConfig / ModelsConfig / ProviderInfo / ModelsByProvider / AgentSettings / UpdateAgentSettings 类型 + ModelsConfig 方法）、`app_settings.go`（AppSettings + 17 个嵌套设置类型 + normalize / defaults / load / save / clone）、`models_store.go`（ModelsStore + LoadModels + 所有 store 方法 + clone / validate / persist 辅助）、`app_settings_store.go`（AppSettingsStore + Snapshot / Update）、`normalize.go`（normalizeProviders / normalizeModels / mergeRuntimeModelsByProvider）。同步删除所有老版本兼容代码：`legacyModelsConfig` 结构体、`migrateLegacyProviders()` 迁移函数、`legacyProviderAPIType()` 兼容函数；`LoadModels()` 简化为直接反序列化 `ModelsConfig`，`normalizeProviders()` 将默认 APIType 逻辑内联。
>
> 文档注记: 2026-04-17 - `RequestAPI` 现注入 `*proxy.Proxy`；`ReplayRequest` / `ComposeRequest` 不再在 bindings 层创建独立 `http.Client`，而是通过 `Proxy.ExecuteRequest()` 复用当前 Proxy 实例的 shared upstream client 与 External Proxy / bypass 行为。当前实现已在 MITM HTTPS 请求路径捕获原始客户端 `ClientHello` 首个 TLS record，并在正常 MITM 上游转发与 `ReplayRequest` 的 HTTPS 重放路径中统一通过 `Proxy.ExecuteRequestWithClientHello()` + `uTLS` 做 best-effort `ClientHelloSpec` 复用；若无捕获数据、请求不可安全重发或 `uTLS` 握手失败，则回退到现有 shared upstream client。该能力仅覆盖 `ClientHello` 层，**不**承诺完整浏览器后续握手/HTTP2 指纹原样回放。
>
> 文档注记: 2026-04-24 - 已修复 `internal/agent/eino_events.go` 在 ADK `ExitTool` 路径下遗漏 assistant 侧 `exit` tool call `final_result` 的问题；当前最终答案提取除 `Content / MultiContent / ReasoningContent` 外，还必须兼容从 `exit` 工具参数回收终止答案，以避免 GLM / OpenAI-compatible 模型被误判为 `empty_final_answer`。
>
> 文档注记: 2026-05-22 - 已完成 `internal/storage` 从 `map + sync.RWMutex` 到 `go-memdb` 的完整迁移：`Storage` 仅保留 `db *memdb.MemDB`，不再维护 `d.mu` / `d.sessions` 镜像缓存，也不再保留 `activeMu` / `activeSession` 一类 active-session cache；Session 与 Request 落在独立 memdb table（`schema.go` 定义 3+6 索引）；`Session.Requests` / `Session.HostGroups` 仅作为派生读取字段；`ImportAll` 通过新建 memdb 实例原子替换；所有 `*Locked` 兼容 shim 已删除；`CreateSession` / `SaveRequest` 在克隆存储后回写自动生成字段（ID/SessionID/IsActive/CreatedAt/UpdatedAt）到调用方原始对象。同步修复 `ComposeRequest` / `ReplayRequest` 测试断言为正确的 upsert 语义（每 ID 恰好 1 条记录）。
>
> 文档注记: 2026-05-22 - 已对 `internal/storage/helpers.go` / `session.go` 与 `internal/agent/agent_dispatch.go` 做零行为变化洁癖清理：删除未使用的 `cloneSession()`、移除单次使用的 `isSessionRecordActive()`、收紧 `buildSessionTxn()` 到仅接收 `sessionRecord`，并确保 `get_request` / `diff_requests` 在 dispatcher 中继续优先使用已解析的 `session_id` 而不是回落到裸 `defaultSessionID`。
>
> 文档注记: 2026-05-26 - 已将 `internal/agent/runtime.go` 拆分提取为独立 `internal/agent/runtime/` 子包，`ReactRuntime` 重命名为 `runtime.Runner`，并通过 `ExecuteToolFunc` 注入 `a.safeExecuteTool` 消除 `agent ↔ runtime` 循环依赖；`AgentEvent` / `RuntimeResult` / `ToolExecutionResult` / `SafeToolResult` / `TruncateOutput` 现由 runtime 拥有，父包通过 type alias/薄包装继续对外保持兼容。
>
> 文档注记: 2026-05-26 - 已完成 Phase 2+3+4 工具提取：新增 `internal/agent/tools/` 与 `internal/agent/tools/builtin/`，将 schema/catalog、参数解析、schema 校验、executor、模糊匹配与全部 builtin tools 从父包提取出去；`Agent` 现持有 `executor *tools.Executor` 与显式注入的 `store *storage.Storage`，bindings/tests 通过 `SetStore(...)` 完成装配，旧 `agent_dispatch.go`/`tools*.go`/`sandbox.go` 等文件已删除，仅保留 `tool_registry_compat.go` 作为测试兼容层。
>
> 文档注记: 2026-05-26 - 已修正 request-targeted analysis 的最小后续问题：当 `Analyze()` 仅收到 `RequestID` 而未提供 `SessionID` 时，现先按 request ID 直接读取目标请求并回填其 session；`loadChatHistoryMessages()` 也已改为只向模型提供最近 12 条 storage-backed chat messages，避免历史上下文无界增长。
>
> 文档注记: 2026-06-29 - 已实现 Agent 认知工具套件（`summarize_session` / `classify_requests` / `trace_flow_sequence` / `test_hypothesis`），分别位于 `internal/agent/tools/builtin/summarize.go`、`classify.go`、`flow.go`、`hypothesis.go` + `hypothesis_parser.go`；同步更新 `catalog.go` schema、`exports.go` 导出、`mcp/builtin.go` MCP 注册与 `agent_role.txt` 六阶段分析工作流。

---

**最后更新**: 2026-06-29
**维护者**: PacketMind Team
