<div align="center">

# 🧠 PacketMind

**AI-Powered Desktop Packet Capture & Request Tracing**

[![Version](https://img.shields.io/badge/version-v1.0.13-blue.svg)](https://github.com/packetmind/packetmind/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/packetmind/packetmind/blob/main/LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-2D8CFF)](https://wails.io/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)]()
[![Build](https://img.shields.io/github/actions/workflow/status/packetmind/packetmind/wails-build.yml?branch=main)](https://github.com/packetmind/packetmind/actions)

<br/>

*Other tools show you packets. PacketMind explains them.*

[English](./README.md) · [中文](./README.zh-CN.md)

<br/>

<!--
  📸 TODO: Replace placeholder with actual screenshot
  Recommended: 1280x720 PNG, dark theme, showing main UI with AI panel open
  <img src="docs/screenshot-main.png" width="90%" alt="PacketMind main interface" />
-->

**[ 🖼️ Screenshot: Main Interface — 4-panel layout with session list, request tree, detail view, and AI panel ]**

</div>

---

## 🎯 Why PacketMind?

You've used Charles Proxy, Fiddler, or mitmproxy. They capture traffic. But when you need to understand *why* a request fails, *where* a token came from, or *how* a session flows across 30 requests — you're on your own, scrolling through raw headers.

**PacketMind changes that.** It combines a full-featured desktop proxy with an AI Agent that can autonomously trace values across your captured session, explain authentication flows, and answer questions about your traffic in plain language.

| | PacketMind | Charles | Fiddler | mitmproxy | Proxyman |
|---|:---:|:---:|:---:|:---:|:---:|
| HTTP/HTTPS/SOCKS5 Proxy | ✅ | ✅ | ✅ | ✅ | ✅ |
| HTTPS MITM Decryption | ✅ | ✅ | ✅ | ✅ | ✅ |
| WebSocket Capture | ✅ | ✅ | ⚠️ | ⚠️ | ✅ |
| Desktop GUI | ✅ | ✅ | ✅ | ❌ | ✅ |
| **AI Request Analysis** | **✅** | ❌ | ❌ | ❌ | ❌ |
| **Provenance Tracing** | **✅** | ❌ | ❌ | ❌ | ❌ |
| **Multi-Agent Orchestration** | **✅** | ❌ | ❌ | ❌ | ❌ |
| **MCP Tool Integration** | **✅** | ❌ | ❌ | ❌ | ❌ |
| Open Source | ✅ | ❌ | ❌ | ✅ | ❌ |
| Free | ✅ | ❌ | ⚠️ | ✅ | ⚠️ |

---

## ✨ Features

### 🔄 Proxy Engine

- **HTTP / HTTPS / SOCKS5** — Full proxy stack with MITM decryption and auto CA certificate generation
- **WebSocket Capture** — Bidirectional frame recording across all proxy modes
- **Runtime Hot-Reload** — Change listener ports without restarting
- **Throttling** — Bandwidth control with latency injection and up/down speed limits
- **External Proxy Chaining** — Route traffic through upstream proxies with auth and bypass rules
- **Access Control** — Client IP whitelist
- **Breakpoints & Rewrite Rules** — Intercept and modify traffic in transit
- **Reverse Proxy & Port Forwarding** — Flexible traffic routing

### 🤖 AI Agent Analysis

- **ReAct Agent** — Multi-step reasoning loop with 8 built-in investigation tools
- **4 Specialist Agents** — Header, Body, Security, and General experts with custom tool whitelists
- **Multi-Specialist Handoff** — Agents collaborate: header specialist detects a token → hands off to security specialist for auth analysis
- **Provenance Tracing** — Ask *"Where did this auth token come from?"* and the Agent traces it across your entire captured session
- **Streaming Analysis** — Watch the Agent think, act, and observe in real time
- **Multi-Provider Support** — Works with OpenAI, Zhipu GLM-4, and any OpenAI-compatible endpoint
- **Dynamic Model Discovery** — Fetch available models from your provider's API directly in the UI
- **MCP Integration** — Extend the Agent's toolset via [Model Context Protocol](https://modelcontextprotocol.io/) servers
- **Session Memory** — Conversations maintain context across follow-up questions

### 🖥️ Desktop Experience

- **Charles-Style Request Tree** — Grouped by host, collapsible by path segments, with protocol icons (HTTP/HTTPS/WebSocket/SOCKS)
- **Multi-Tab Request Detail** — Overview, Contents, Summary, SSL, and WebSocket views
- **Smart Content Views** — JSON Tree, HTML Preview, Image Preview, Hex, Raw — based on content type
- **Right-Click AI Analysis** — Select any header, cookie, or field → "Ask AI about this"
- **Frameless Window** — Custom title bar with native macOS traffic light support
- **Application Menu** — Full desktop menu bar with submenus (Recent, Focused Hosts, Proxy Settings)
- **Real-Time Highlighting** — New requests flash briefly in the tree to draw attention
- **Dark Theme** — Professional dark UI designed for extended debugging sessions

### 📊 Rich Data Capture

- **60+ Fields Per Request** — Timing, TLS, WebSocket, errors, addresses — all captured
- **Full Timing Breakdown** — DNS, Connect, TLS Handshake, Request, Response, Latency durations
- **SSL/TLS Details** — Protocol version, cipher suite, ALPN, session resumption, full certificate chain with extensions
- **Content Decoding** — Automatic gzip/deflate/brotli decompression with charset transcoding (GBK, ISO-8859-1, etc.)
- **Error Recording** — All 13+ failure paths in the proxy record structured error data — no silent failures
- **Export** — HAR and cURL export, request replay
- **Session Import/Export** — Full session data persistence with multi-indexed in-memory storage

---

## 🤖 AI Agent — How It Works

<!--
  📸 TODO: Add a diagram showing the ReAct loop
  Recommended: ASCII flow diagram or SVG showing Route → Think → Act → Observe → Handoff → Conclude
-->

PacketMind's Agent uses a **ReAct (Reason + Act)** loop to investigate your traffic:

```
1. ROUTE    → Select the best specialist agent (header / body / security / general)
2. THINK    → LLM reasons about the question and available data
3. ACT      → Call investigation tools (search, trace, find related requests)
4. OBSERVE  → Process tool results
5. HANDOFF  → If another specialist is better suited, delegate with context
6. CONCLUDE → Return findings with full provenance chain
```

### Built-in Investigation Tools

| Tool | What It Does |
|------|-------------|
| `get_request` | Fetch full request details by ID |
| `search_requests_by_host` | Find all requests to a specific host |
| `search_requests_by_header` | Find requests matching a header name/value |
| `search_requests_by_body` | Find requests containing a body fragment |
| `search_requests_by_response` | Find requests whose response contains a value |
| `find_prior_response_sources` | Trace backward: where did a value first appear? |
| `find_later_request_usages` | Trace forward: where is a value reused? |
| `trace_value_flow` | End-to-end provenance tracing with confidence scoring |

### Multi-Specialist Collaboration

The Orchestrator routes questions to the right specialist based on keywords:

- **Header Specialist** → Cookies, Set-Cookie, Content-Type, Authorization headers
- **Body Specialist** → JSON, XML, form data, payload analysis
- **Security Specialist** → Auth tokens, session IDs, suspicious patterns
- **General Specialist** → Broad fallback, full tool access

When a specialist detects something outside its domain, it can **hand off** to another specialist — up to 3 hops deep, with loop prevention. The full chain is shown in the UI.

### MCP Extensibility

Connect [MCP-compatible](https://modelcontextprotocol.io/) tool servers, and the Agent automatically discovers and uses their tools. This means you can extend PacketMind's analysis capabilities without modifying the core — just plug in a tool server.

---

## ⚡ Quick Start

### Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/)

### Build & Run

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone the repository
git clone https://github.com/packetmind/packetmind.git
cd packetmind

# Start in development mode (with hot reload)
wails dev
```

### Configure AI Provider

1. Open PacketMind → Settings → AI Model Configuration
2. Select a provider (OpenAI, Zhipu GLM-4, or custom endpoint)
3. Enter your API key
4. Click "Initialize Models" to discover available models
5. Select your preferred model and start analyzing

### Configure Your Browser

Set your browser/system proxy to `localhost:8888` (default port). HTTPS traffic will be decrypted using PacketMind's auto-generated CA certificate.

> 💡 **First run?** PacketMind will prompt you to install the CA certificate for HTTPS MITM.

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.25 |
| **Desktop Framework** | [Wails v2](https://wails.io/) |
| **Frontend** | Vue 3 · TypeScript · Pinia · Ant Design Vue |
| **AI Engine** | ReAct Agent · Multi-Specialist Orchestration · MCP |
| **AI Providers** | OpenAI · Zhipu GLM-4 · OpenAI-compatible endpoints |
| **Storage** | In-Memory (multi-indexed, import/export) |
| **Proxy** | HTTP · HTTPS (MITM) · SOCKS5 · WebSocket |
| **CI/CD** | GitHub Actions (Windows · macOS · Linux) |

---

## 📁 Project Structure

```
packetmind/
├── app.go                  # Wails application entry point
├── internal/
│   ├── agent/              # Agent facade, llmcore, provider, specialists, memory, MCP, tools
│   ├── api/bindings/       # Wails frontend bindings
│   ├── config/             # Configuration & persistence
│   ├── proxy/              # Proxy core (HTTP/HTTPS/SOCKS5/MITM)
│   └── storage/            # In-memory storage with multi-index
├── gui/                    # Vue 3 frontend (src/components, stores, api)
├── configs/                # packetmind.json · models.json · specialists.json
├── build/                  # App icons & build resources
└── docs/                   # Design & module documentation
```

---

## 🔧 Development

```bash
# Development (hot reload)
wails dev

# Build production binary
wails build

# Run tests
make test

# Lint
make lint

# Format
make fmt
```

See [AGENTS.md](./AGENTS.md) for detailed coding conventions and architecture decisions.

---

## 📸 Screenshots

<!--
  📸 TODO: Replace all placeholders with actual screenshots
  Recommended specs:
  - PNG format, 1280x720 or 1920x1080
  - Dark theme preferred
  - Show realistic traffic, not placeholder data
  - Redact any real API keys or sensitive data
-->

| Main Interface |
|---|
| **[ 🖼️ Screenshot: Main 4-panel layout — Session List, Request Tree, Request Detail, AI Panel ]** |

| AI Agent Analysis | Specialist Handoff |
|---|---|
| **[ 🖼️ AI analyzing a request with visible Think/Act/Observe steps ]** | **[ 🖼️ Multi-specialist collaboration with handoff chain ]** |

| Request Detail | SSL Details |
|---|---|
| **[ 🖼️ Multi-tab request view with JSON Tree / Headers / Timing ]** | **[ 🖼️ TLS certificate chain, cipher suite, ALPN info ]** |

---

## 🗺️ Roadmap

### Done ✅
- [x] HTTP/HTTPS/SOCKS5 proxy with MITM
- [x] AI ReAct Agent with 8 investigation tools
- [x] Multi-specialist orchestration with handoff
- [x] MCP tool integration
- [x] WebSocket capture & inspection
- [x] Charles-style request tree with host grouping
- [x] SSL/TLS details view
- [x] Full timing breakdown
- [x] HAR/cURL export & request replay
- [x] Dynamic AI model discovery
- [x] Content decoding (gzip/deflate/brotli + charset)

### Next 🔜
- [ ] Interactive request/response breakpoints
- [ ] Map Remote / Map Local (URL rewriting)
- [ ] DNS spoofing & mirroring
- [ ] gRPC / Protobuf decoding
- [ ] Request diff & comparison
- [ ] Auto-save sessions to disk
- [ ] Scriptable mock server

### Future 🚀
- [ ] Plugin API via MCP ecosystem
- [ ] Team collaboration (shared sessions)
- [ ] Homebrew / AUR / Chocolatey packages
- [ ] REST API for automation

---

## 🤝 Contributing

Contributions are welcome! Here's how to get started:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Write** tests for your changes
4. **Ensure** all tests pass (`make test`) and code is formatted (`make fmt`)
5. **Submit** a Pull Request

Please read [AGENTS.md](./AGENTS.md) for detailed coding conventions, architecture decisions, and contribution guidelines.

---

## 📜 License

PacketMind is released under the [MIT License](./LICENSE).

```
MIT License
Copyright (c) 2026 Tsung
```

---

## 🙏 Acknowledgments

Built with these excellent open-source projects:

- [Wails](https://wails.io/) — Go framework for desktop applications
- [Vue.js](https://vuejs.org/) — Progressive JavaScript framework
- [Ant Design Vue](https://antdv.com/) — Enterprise-class UI components
- [Pinia](https://pinia.vuejs.org/) — Vue state management
- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket implementation for Go

---

<div align="center">

**[⭐ Star this repo](https://github.com/packetmind/packetmind) if you find it useful!**

Made with 🧠 by [PacketMind Team](https://github.com/packetmind)

</div>
