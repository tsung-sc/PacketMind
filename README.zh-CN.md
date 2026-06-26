<div align="center">

# 🧠 PacketMind

**AI 驱动的桌面抓包与请求溯源分析工具**

[![Version](https://img.shields.io/badge/version-v1.0.13-blue.svg)](https://github.com/packetmind/packetmind/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/packetmind/packetmind/blob/main/LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-2D8CFF)](https://wails.io/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)]()
[![Build](https://img.shields.io/github/actions/workflow/status/packetmind/packetmind/wails-build.yml?branch=main)](https://github.com/packetmind/packetmind/actions)

<br/>

*其他工具只展示数据包，PacketMind 帮你理解它们。*

[English](./README.md) · [中文](./README.zh-CN.md)

<br/>

<!--
  📸 TODO: 替换为实际截图
  建议：1280x720 PNG，暗色主题，展示主界面和 AI 面板
  <img src="docs/screenshot-main.png" width="90%" alt="PacketMind 主界面" />
-->

**[ 🖼️ 截图：主界面 — 四栏布局，包含会话列表、请求树、详情视图、AI 面板 ]**

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
| **多 Agent 协作** | **✅** | ❌ | ❌ | ❌ | ❌ |
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

- **ReAct Agent** — 多步推理循环，内置 8 个调查工具
- **4 个专家 Agent** — Header、Body、Security 和 General 专家，各有专属工具白名单
- **多专家交接** — Agent 之间协作：header 专家发现 token → 移交给 security 专家进行认证分析
- **值溯源追踪** — 问一句 *"这个 auth token 从哪来的？"*，Agent 会自动在你的整个抓包会话中追踪溯源
- **流式分析** — 实时观看 Agent 的思考、行动和观察过程
- **多 Provider 支持** — 兼容 OpenAI、智谱 GLM-4 及任何 OpenAI 兼容接口
- **动态模型发现** — 直接在 UI 中从 Provider API 获取可用模型列表
- **MCP 集成** — 通过 [Model Context Protocol](https://modelcontextprotocol.io/) 服务器扩展 Agent 工具集
- **会话记忆** — 对话上下文在追问之间保持连续

### 🖥️ 桌面体验

- **Charles 风格请求树** — 按 Host 分组，按路径段折叠，带协议图标（HTTP/HTTPS/WebSocket/SOCKS）
- **多标签请求详情** — Overview、Contents、Summary、SSL 和 WebSocket 视图
- **智能内容视图** — JSON Tree、HTML 预览、图片预览、Hex、Raw — 根据内容类型自动选择
- **右键 AI 分析** — 选中任意 header、cookie 或字段 → "向 AI 提问"
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

<!--
  📸 TODO: 添加 ReAct 循环流程图
  建议：SVG 展示 Route → Think → Act → Observe → Handoff → Conclude
-->

PacketMind 的 Agent 使用 **ReAct（Reason + Act）** 循环来分析你的流量：

```
1. ROUTE   → 选择最合适的专家 Agent（header / body / security / general）
2. THINK   → LLM 推理问题与可用数据
3. ACT     → 调用调查工具（搜索、追踪、查找关联请求）
4. OBSERVE → 处理工具返回结果
5. HANDOFF → 若其他专家更适合，携带上下文进行委托
6. CONCLUDE → 返回结论及完整溯源链
```

### 内置调查工具

| 工具 | 功能 |
|------|------|
| `get_request` | 按 ID 获取完整请求详情 |
| `search_requests_by_host` | 查找特定 Host 的所有请求 |
| `search_requests_by_header` | 按请求头名称/值搜索请求 |
| `search_requests_by_body` | 搜索请求体中包含指定内容的请求 |
| `search_requests_by_response` | 搜索响应体中包含指定值的请求 |
| `find_prior_response_sources` | 向前追溯：某个值首次出现在哪里？ |
| `find_later_request_usages` | 向后追踪：某个值在哪里被复用？ |
| `trace_value_flow` | 端到端值溯源追踪，带置信度评分 |

### 多专家协作

编排器根据关键词将问题路由到对应的专家：

- **Header 专家** → Cookie、Set-Cookie、Content-Type、Authorization 头
- **Body 专家** → JSON、XML、表单数据、载荷分析
- **Security 专家** → 认证 token、会话 ID、可疑模式
- **General 专家** → 广域回退，拥有完整工具权限

当某个专家发现超出自身领域的内容时，可以**交接**给另一个专家——最多 3 跳，带循环防护。完整协作链路在 UI 中可见。

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

1. 打开 PacketMind → 设置 → AI 模型配置
2. 选择 Provider（OpenAI、智谱 GLM-4 或自定义接口）
3. 输入 API Key
4. 点击"初始化模型"发现可用模型
5. 选择你的首选模型，开始分析

### 配置浏览器

将浏览器/系统代理设置为 `localhost:8080`（默认端口）。HTTPS 流量将通过 PacketMind 自动生成的 CA 证书进行解密。

> 💡 **首次运行？** PacketMind 会提示你安装 CA 证书以支持 HTTPS MITM。

---

## 🛠️ 技术栈

| 层级 | 技术 |
|------|------|
| **后端** | Go 1.25 |
| **桌面框架** | [Wails v2](https://wails.io/) |
| **前端** | Vue 3 · TypeScript · Pinia · Ant Design Vue |
| **AI 引擎** | ReAct Agent · 多专家协作 · MCP |
| **AI Provider** | OpenAI · 智谱 GLM-4 · OpenAI 兼容接口 |
| **存储** | 内存（多索引，支持导入/导出） |
| **代理** | HTTP · HTTPS (MITM) · SOCKS5 · WebSocket |
| **CI/CD** | GitHub Actions (Windows · macOS · Linux) |

---

## 📁 项目结构

```
packetmind/
├── app.go                  # Wails 应用入口
├── internal/
│   ├── agent/              # Agent、专家、记忆、MCP、工具
│   ├── api/bindings/       # Wails 前端绑定
│   ├── config/             # 配置与持久化
│   ├── proxy/              # 代理核心 (HTTP/HTTPS/SOCKS5/MITM)
│   └── storage/            # 多索引内存数据库
├── gui/                    # Vue 3 前端 (src/components, stores, api)
├── configs/                # config.yaml · models.json · desktop.json
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

## 📸 截图

<!--
  📸 TODO: 替换所有占位符为实际截图
  建议规格：
  - PNG 格式，1280x720 或 1920x1080
  - 优先使用暗色主题
  - 展示真实流量数据，而非占位数据
  - 需脱敏处理真实 API Key 或敏感数据
-->

| 主界面 |
|---|
| **[ 🖼️ 截图：四栏主界面布局 — 会话列表、请求树、请求详情、AI 面板 ]** |

| AI Agent 分析 | 专家交接协作 |
|---|---|
| **[ 🖼️ AI 分析请求过程，可见 Think/Act/Observe 步骤 ]** | **[ 🖼️ 多专家协作，显示完整交接链路 ]** |

| 请求详情 | SSL 详情 |
|---|---|
| **[ 🖼️ 多标签请求视图，含 JSON Tree / Headers / Timing ]** | **[ 🖼️ TLS 证书链、密码套件、ALPN 信息 ]** |

---

## 🗺️ 路线图

### 已完成 ✅
- [x] HTTP/HTTPS/SOCKS5 代理与 MITM
- [x] AI ReAct Agent 及 8 个调查工具
- [x] 多专家协作与交接机制
- [x] MCP 工具集成
- [x] WebSocket 抓包与检查
- [x] Charles 风格请求树（按 Host 分组）
- [x] SSL/TLS 详情视图
- [x] 完整时序分解
- [x] HAR/cURL 导出与请求重放
- [x] AI 模型动态发现
- [x] 内容解码（gzip/deflate/brotli + 字符集）

### 近期计划 🔜
- [ ] 交互式请求/响应断点
- [ ] Map Remote / Map Local（URL 重写）
- [ ] DNS 欺骗与镜像
- [ ] gRPC / Protobuf 解码
- [ ] 请求对比与差异分析
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
