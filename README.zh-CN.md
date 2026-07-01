<div align="center">

# 🧠 PacketMind

**AI 驱动的桌面抓包与请求溯源分析工具**

[![Version](https://img.shields.io/badge/version-v1.0.13-blue.svg)](https://github.com/packetmind/packetmind/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/packetmind/packetmind/blob/main/LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-2D8CFF)](https://wails.io/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)]()
[![Build](https://img.shields.io/github/actions/workflow/status/packetmind/packetmind/wails-build.yml?branch=main)](https://github.com/packetmind/packetmind/actions)

<br/>

*其他工具只展示数据包，PacketMind 帮你理解它们。*

[English](./README.md) · [中文](./README.zh-CN.md)

<br/>

</div>

---

## 🎯 为什么选择 PacketMind？

你用过 Charles Proxy、Fiddler 或者 mitmproxy。它们能抓到流量，但当你需要搞清楚一个请求*为什么*失败、一个 token *从哪里来*、或者一个会话*如何*在 30 个请求之间流转的时候——你只能自己翻原始 header。

**PacketMind 改变了这一点。** 它将一个功能完整的桌面代理与 AI Agent 相结合，可以自主追踪你的抓包会话中的值来源，解释认证流程，并用自然语言回答关于你流量的问题。

| | PacketMind | Charles | Fiddler | mitmproxy | Proxyman |
|---|:---:|:---:|:---:|:---:|:---:|
| HTTP/HTTPS/SOCKS5 代理 | ✅ | ✅ | ✅ | ✅ | ✅ |
| HTTPS MITM 解密 | ✅ | ✅ | ✅ | ✅ | ✅ |
| WebSocket 抓包 | ✅ | ✅ | ⚠️ | ⚠️ | ✅ |
| 桌面 GUI | ✅ | ✅ | ✅ | ❌ | ✅ |
| **AI 请求分析** | **✅** | ❌ | ❌ | ❌ | ❌ |
| **值溯源追踪** | **✅** | ❌ | ❌ | ❌ | ❌ |
| **假设驱动分析** | **✅** | ❌ | ❌ | ❌ | ❌ |
| **MCP 工具集成** | **✅** | ❌ | ❌ | ❌ | ❌ |
| 开源 | ✅ | ❌ | ❌ | ✅ | ❌ |
| 免费 | ✅ | ❌ | ⚠️ | ✅ | ⚠️ |

---

## ✨ 功能特性

### 🔄 代理引擎

- **HTTP / HTTPS / SOCKS5** — 完整代理栈，支持 MITM 解密与 CA 证书自动生成
- **WebSocket 抓包** — 跨所有代理模式的双向帧录制
- **端口热重载** — 无需重启即可更换监听端口
- **带宽限速** — 支持延迟注入和上下行速度限制
- **外部代理链** — 通过上游代理转发流量，支持认证和绕过规则
- **访问控制** — 客户端 IP 白名单
- **断点与重写规则** — 拦截并修改传输中的流量
- **反向代理与端口转发** — 灵活的流量路由

### 🤖 AI Agent 分析

- **ReAct Agent** — 多步推理循环，内置 12 个调查工具
- **认知工具** — 会话总览 (`summarize_session`)、请求分类 (`classify_requests`)、流程序列追踪 (`trace_flow_sequence`)
- **假设验证** — 对签名算法、token 编码、字段生成规则提出并验证假设
- **值溯源追踪** — 问一句 *"这个 auth token 从哪来的？"*，Agent 会自动在你的整个抓包会话中追踪溯源
- **流式分析** — 实时观看 Agent 的思考、行动和观察过程
- **运行中追加提问** — Agent 正在分析时仍可发送 follow-up 消息，在下一安全点自动注入当前分析上下文
- **多 Provider 支持** — 兼容 OpenAI、智谱 GLM-4 及任何 OpenAI 兼容接口
- **按渠道管理模型** — 每个 LLM 渠道独立管理自己的模型列表
- **MCP 集成** — 通过 [Model Context Protocol](https://modelcontextprotocol.io/) 服务器扩展 Agent 工具集
- **会话记忆** — 对话上下文在追问之间保持连续

### 🖥️ 桌面体验

- **Charles 风格请求树** — 按 Host 分组，按路径段折叠，带协议图标（HTTP/HTTPS/WebSocket/SOCKS）
- **Sessions 顶部栏** — 横向 Session 标签栏，像 Charles 一样切换
- **多标签请求详情** — Overview、Contents、Summary、SSL 和 WebSocket 视图
- **智能内容视图** — JSON Tree、HTML 预览、图片预览、Hex、Raw — 根据内容类型自动选择
- **无边框窗口** — 自定义标题栏，支持 macOS 原生红绿灯按钮
- **应用菜单** — 完整的桌面菜单栏，支持子菜单（最近打开、聚焦 Host、代理设置）
- **实时高亮** — 新请求在树中短暂闪烁以吸引注意
- **暗色主题** — 专为长时间调试设计的专业暗色界面

### 📊 丰富数据采集

- **每条请求 60+ 字段** — 时间、TLS、WebSocket、错误、地址 — 全部捕获
- **完整时序分解** — DNS、连接、TLS 握手、请求、响应、延迟时长
- **SSL/TLS 详情** — 协议版本、密码套件、ALPN、会话恢复、完整证书链及扩展
- **内容解码** — 自动 gzip/deflate/brotli 解压，支持字符集转码（GBK、ISO-8859-1 等）
- **错误记录** — 代理中 13+ 个失败路径全部记录结构化错误数据 — 无静默丢失
- **导出** — HAR 和 cURL 导出，请求重放
- **会话导入/导出** — 完整的会话数据持久化，基于多索引内存存储

---

## 🤖 AI Agent — 工作原理

PacketMind 的 Agent 使用 **ReAct（Reason + Act）** 循环来分析你的流量：

```
1. 总览    → 先了解 session 的整体结构和流量分布
2. 分类    → 识别关键节点（登录、token 下发、签名请求）
3. 流程    → 追踪请求间的时序依赖和字段流转关系
4. 深入    → 对关键请求进行详细检查和对比分析
5. 验证    → 对发现的模式提出假设并自动验证
6. 报告    → 返回结论及完整溯源链
```

### 内置调查工具

| 工具 | 功能 |
|------|------|
| `summarize_session` | Session 总览 — hosts、端点、认证检测 |
| `classify_requests` | 自动分类请求角色（认证入口、签名、token、数据查询、静态资源） |
| `trace_flow_sequence` | 时序流程追踪，自动检测 cookie/token/value 流转关系 |
| `get_request` | 按 ID 获取完整请求详情 |
| `search_all_fields` | 跨所有字段全量搜索 |
| `search_requests_by_header` | 按请求头名称/值搜索请求 |
| `search_requests_by_body` | 搜索请求体中包含指定内容的请求 |
| `search_requests_by_response` | 搜索响应体中包含指定值的请求 |
| `find_prior_response_sources` | 向前追溯：某个值首次出现在哪里？ |
| `find_later_request_usages` | 向后追踪：某个值在哪里被复用？ |
| `trace_value_flow` | 端到端值溯源追踪，带置信度评分 |
| `test_hypothesis` | 对签名/编码假设进行实际数据验证 |
| `diff_requests` | 逐字段比较两个请求的差异 |
| `analyze_encoding` | 检测并解码嵌套编码层 |
| `batch_execute` | 并发执行多个工具调用 |
| `bash` | 在沙箱工作区执行 Shell 命令 |

### 运行中追加提问

Agent 正在分析时，你可以发送 follow-up 消息。它们会进入待处理队列，在下一安全点自动注入 Agent 的上下文——Agent 读取你的新指令后调整分析方向，无需完全重启。这类似于 IDE Agent 和主流聊天工具处理用户中断的方式。

### MCP 可扩展性

连接 [MCP 兼容](https://modelcontextprotocol.io/)的工具服务器，Agent 会自动发现并使用其工具。这意味着你无需修改核心代码就能扩展 PacketMind 的分析能力——只需接入一个工具服务器。

---

## ⚡ 快速开始

### 前置条件

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/)

### 构建与运行

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 克隆仓库
git clone https://github.com/packetmind/packetmind.git
cd packetmind

# 启动开发模式（支持热重载）
wails dev
```

### 配置 AI Provider

1. 打开 PacketMind → 设置 → 模型配置
2. 新增一个渠道/Provider（OpenAI 兼容接口）
3. 输入 API Key 和 Base URL
4. 手动添加模型到该渠道
5. 在 AI 面板中选择你的首选模型，开始分析

### 配置浏览器

将浏览器/系统代理设置为 `localhost:8888`（默认端口）。HTTPS 流量将通过 PacketMind 自动生成的 CA 证书进行解密。

> 💡 **首次运行？** PacketMind 会提示你安装 CA 证书以支持 HTTPS MITM。

---

## 🛠️ 技术栈

| 层级 | 技术 |
|------|------|
| **后端** | Go |
| **桌面框架** | [Wails v2](https://wails.io/) |
| **前端** | Vue 3 · TypeScript · Pinia · Ant Design Vue |
| **AI 引擎** | ReAct Agent · 认知工具 · MCP |
| **AI Provider** | OpenAI · 智谱 GLM-4 · OpenAI 兼容接口 |
| **存储** | 内存（go-memdb，多索引，支持导入/导出） |
| **代理** | HTTP · HTTPS (MITM) · SOCKS5 · WebSocket |
| **CI/CD** | GitHub Actions (Windows · macOS · Linux) |

---

## 📁 项目结构

```
packetmind/
├── app.go                  # Wails 应用入口
├── internal/
│   ├── agent/              # Agent、runtime、llmtypes/llmcore、provider、MCP、tools
│   ├── api/bindings/       # Wails 前端绑定
│   ├── config/             # 配置与持久化
│   ├── proxy/              # 代理核心 (HTTP/HTTPS/SOCKS5/MITM)
│   └── storage/            # 内存存储 (go-memdb)
├── gui/                    # Vue 3 前端 (src/components, stores, api)
├── configs/                # packetmind.json · models.json
├── build/                  # 应用图标与构建资源
└── docs/                   # 设计与模块文档
```

---

## 🔧 开发

```bash
# 开发模式（热重载）
wails dev

# 构建生产版本
wails build

# 运行测试
make test

# 代码检查
make lint

# 格式化
make fmt
```

详细的编码规范和架构决策请参阅 [AGENTS.md](./AGENTS.md)。

---

## 🗺️ 路线图

### 已完成 ✅
- [x] HTTP/HTTPS/SOCKS5 代理与 MITM
- [x] AI ReAct Agent 及 12 个调查工具
- [x] 认知工具（总览、分类、流程追踪、假设验证）
- [x] 运行中追加提问（follow-up steering）
- [x] 假设驱动的签名/编码分析
- [x] MCP 工具集成
- [x] WebSocket 抓包与检查
- [x] Charles 风格请求树（按 Host 分组）
- [x] SSL/TLS 详情视图
- [x] 完整时序分解
- [x] HAR/cURL 导出与请求重放
- [x] 内容解码（gzip/deflate/brotli + 字符集）
- [x] 按渠道管理模型

### 近期计划 🔜
- [ ] 交互式请求/响应断点
- [ ] Map Remote / Map Local（URL 重写）
- [ ] DNS 欺骗与镜像
- [ ] gRPC / Protobuf 解码
- [ ] 会话自动保存到磁盘
- [ ] 可脚本化的 Mock 服务器

### 未来规划 🚀
- [ ] 通过 MCP 生态的插件 API
- [ ] 团队协作（共享会话）
- [ ] Homebrew / AUR / Chocolatey 包
- [ ] REST API 用于自动化

---

## 🤝 参与贡献

欢迎贡献！步骤如下：

1. **Fork** 本仓库
2. **创建**特性分支（`git checkout -b feature/amazing-feature`）
3. **编写**测试覆盖你的改动
4. **确保**所有测试通过（`make test`）且代码已格式化（`make fmt`）
5. **提交** Pull Request

详细的编码规范和架构决策请参阅 [AGENTS.md](./AGENTS.md)。

---

## 📜 许可证

PacketMind 基于 [MIT 许可证](./LICENSE) 发布。

```
MIT License
Copyright (c) 2026 Tsung
```

---

## 🙏 致谢

基于以下优秀的开源项目构建：

- [Wails](https://wails.io/) — Go 桌面应用框架
- [Vue.js](https://vuejs.org/) — 渐进式 JavaScript 框架
- [Ant Design Vue](https://antdv.com/) — 企业级 UI 组件库
- [Pinia](https://pinia.vuejs.org/) — Vue 状态管理
- [gorilla/websocket](https://github.com/gorilla/websocket) — Go WebSocket 实现

---

<div align="center">

**觉得有用？请 [⭐ 给个 Star](https://github.com/packetmind/packetmind)！**

Made with 🧠 by [PacketMind Team](https://github.com/packetmind)

</div>
