# 基于 OpenClaw 平台的全链路技术文档

> **版本**: v1.0  
> **日期**: 2026-03-11  
> **作者**: GLM (AutoClaw)  
> **目标读者**: 云原生运维工程师、AI 应用架构师、平台开发者  
> **核心宗旨**: 快速、准确、秒懂 OpenClaw

---

## 摘要

OpenClaw 🦞 是一个开源的自托管（self-hosted）AI Agent 网关平台，它将主流即时通讯渠道（WhatsApp、Telegram、Discord、iMessage、飞书、Slack 等 20+ 渠道）统一桥接到后端 AI 编码代理（Agent）。系统性解析 OpenClaw 的架构设计、核心机制、运行时模型、安全策略与扩展能力，帮助读者在最短时间内建立完整认知。

**关键词**: AI Gateway · 多渠道代理 · 自托管 · Agent Loop · 会话管理 · 记忆系统 · 安全边界

---

## 目录

1. [平台概述](#1-平台概述)
2. [系统架构](#2-系统架构)
3. [Agent Loop：核心执行引擎](#3-agent-loop核心执行引擎)
4. [会话管理机制](#4-会话管理机制)
5. [记忆系统](#5-记忆系统)
6. [模型与提供商](#6-模型与提供商)
7. [渠道集成矩阵](#7-渠道集成矩阵)
8. [安全体系](#8-安全体系)
9. [自动化与调度](#9-自动化与调度)
10. [技能（Skill）系统](#10-技能skill系统)
11. [节点与移动端](#11-节点与移动端)
12. [部署与运维指南](#12-部署与运维指南)
13. [配置参考速查](#13-配置参考速查)
14. [总结](#14-总结)

---

## 1. 平台概述

### 1.1 一句话定义

> **OpenClaw 是一个运行在你自己机器上的 AI Agent 网关——把你的聊天应用变成 AI 助手的入口。**

### 1.2 核心定位

| 维度 | 说明 |
|------|------|
| **自托管** | 运行在你的硬件上，你的规则，你的数据 |
| **多渠道** | 一个 Gateway 进程同时服务 WhatsApp、Telegram、Discord、iMessage、飞书等 20+ 渠道 |
| **Agent 原生** | 为编码代理设计，内置工具调用、会话管理、记忆系统和多代理路由 |
| **开源** | MIT 许可证，社区驱动 |
| **节点扩展** | iOS/Android/macOS 设备可作为节点接入，提供摄像头、屏幕、位置等能力 |

### 1.3 与其他方案的本质区别

```
传统方案:  聊天App → 云端托管服务 → AI API（数据过第三方）
OpenClaw:  聊天App → [你的机器] Gateway → AI API（数据不过第三方）
```

**关键差异**: 你完全控制 Gateway，所有消息处理在本地完成，不需要信任任何第三方托管服务。

---

## 2. 系统架构

### 2.1 架构全景图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户交互层                                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐│
│  │ WhatsApp │ │ Telegram │ │ Discord  │ │ iMessage │ │ 飞书   ││
│  │ (Baileys)│ │ (grammY) │ │(discord.js│ │(imsg CLI)│ │(REST)  ││
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └───┬────┘│
│       │             │             │             │           │     │
│  ┌────┴─────────────┴─────────────┴─────────────┴───────────┴────┐│
│  │                    Plugin Channels                            ││
│  │        Mattermost · Slack · Signal · IRC · LINE · ...         ││
│  └─────────────────────────┬─────────────────────────────────────┘│
│                            │                                     │
├────────────────────────────┼─────────────────────────────────────┤
│                    ┌───────▼───────┐                             │
│                    │    Gateway    │  ← 单一长驻进程               │
│                    │  (WebSocket)  │  ← 端口 18789 (默认)          │
│                    └───────┬───────┘                             │
│                            │                                     │
│         ┌──────────────────┼──────────────────┐                 │
│         │                  │                  │                 │
│    ┌────▼────┐      ┌─────▼─────┐     ┌──────▼──────┐           │
│    │ Agent   │      │  Control  │     │   Nodes     │           │
│    │ Runtime │      │  Clients  │     │ (移动设备)   │           │
│    │(Pi Core)│      │(mac/CLI/  │     │ iOS/Android │           │
│    │         │      │  WebChat) │     │ macOS/head  │           │
│    └────┬────┘      └───────────┘     └─────────────┘           │
│         │                                                       │
│    ┌────▼────────────────────┐                                  │
│    │   Model Providers       │                                  │
│    │ OpenAI · Google ·       │                                  │
│    │ GLM · Custom · ...      │                                  │
│    └─────────────────────────┘                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Gateway（核心守护进程）

Gateway 是整个系统的**单一真相源**（Single Source of Truth）：

- **维持所有 Provider 连接**（WhatsApp Web、Telegram Bot、Discord Bot 等）
- **暴露 WebSocket API**（请求/响应/服务端推送事件）
- **JSON Schema 验证**所有入站帧
- **发出事件流**: `agent`、`chat`、`presence`、`health`、`heartbeat`、`cron`
- **Canvas 服务**: 在同一端口下提供 `__openclaw__/canvas/` 和 `__openclaw__/a2ui/`

### 2.3 连接生命周期

```
Client                    Gateway
  │                          │
  │── req:connect ──────────→│
  │←─ res (ok) ─────────────│ (payload=hello-ok, 快照: presence+health)
  │                          │
  │←── event:presence ───────│
  │←── event:tick ───────────│
  │                          │
  │── req:agent ─────────────→│
  │←─ res:agent (ack) ──────│ {runId, status:"accepted"}
  │←── event:agent (流式) ───│ (streaming)
  │←─ res:agent (final) ────│ {runId, status, summary}
```

### 2.4 线协议规范

| 项目 | 规范 |
|------|------|
| **传输层** | WebSocket，文本帧，JSON 载荷 |
| **首帧** | 必须是 `connect` |
| **请求格式** | `{type:"req", id, method, params}` → `{type:"res", id, ok, payload\|error}` |
| **事件格式** | `{type:"event", event, payload, seq?, stateVersion?}` |
| **认证** | `connect.params.auth.token` 必须匹配 Gateway Token |
| **幂等键** | 副作用方法（`send`、`agent`）必须携带幂等键 |

---

## 3. Agent Loop：核心执行引擎

Agent Loop 是 OpenClaw 的**灵魂**——它将一条消息转化为行动和回复的完整过程。

### 3.1 执行流水线

```
消息输入 → 参数校验 → 会话解析 → 模型/技能加载 → Agent 执行 → 工具调用 → 流式回复 → 持久化
```

### 3.2 详细步骤

1. **`agent` RPC**: 校验参数 → 解析 session（sessionKey/sessionId）→ 持久化会话元数据 → 立即返回 `{runId, acceptedAt}`
2. **`agentCommand`**: 解析模型 + thinking/verbose 默认值 → 加载 skills 快照 → 调用 `runEmbeddedPiAgent`
3. **`runEmbeddedPiAgent`**: 
   - 通过 **per-session + 全局队列** 序列化运行（防止并发竞争）
   - 解析模型 + 认证配置
   - 订阅 pi 事件并流式传输 assistant/tool deltas
   - 强制超时控制
4. **事件桥接**:
   - tool 事件 → `stream: "tool"`
   - assistant deltas → `stream: "assistant"`
   - 生命周期 → `stream: "lifecycle"`（phase: `start` | `end` | `error`）
5. **`agent.wait`**: 等待 lifecycle end/error → 返回 `{status, startedAt, endedAt, error?}`

### 3.3 队列与并发

- **会话级队列**: 同一 sessionKey 的请求串行执行
- **全局队列**: 可选的全局串行化
- **渠道队列模式**: `collect`（收集）/ `steer`（引导）/ `followup`（追加）

### 3.4 Compaction（上下文压缩）

当会话 token 接近上限时，OpenClaw 触发**自动压缩**：
- 在压缩前触发**静默 agentic turn**，提醒模型写入持久记忆
- 压缩后保留摘要和关键上下文

---

## 4. 会话管理机制

### 4.1 会话模型

```
直接聊天 → agent:<agentId>:<mainKey>（默认 main，所有私聊共享一个会话）
群聊   → 每个群有独立 sessionKey
```

### 4.2 DM 作用域控制（dmScope）

| 策略 | 行为 | 适用场景 |
|------|------|---------|
| `main`（默认） | 所有私聊共享主会话 | 单人使用 |
| `per-peer` | 按发送者隔离 | 多人访问 |
| `per-channel-peer` | 按渠道+发送者隔离 | 多渠道多人 |
| `per-account-channel-peer` | 按账号+渠道+发送者隔离 | 多账号 |

### 4.3 状态存储

```
~/.openclaw/agents/<agentId>/sessions/
├── sessions.json          # 会话索引 {sessionKey → {sessionId, updatedAt, ...}}
└── <SessionId>.jsonl      # 对话记录（逐行 JSON）
```

### 4.4 维护策略（默认值）

- `maintenance.mode`: `warn`（仅警告不自动删除）
- `pruneAfter`: `30d`（30天后标记过期）
- `maxEntries`: `500`
- `rotateBytes`: `10mb`

---

## 5. 记忆系统

### 5.1 设计哲学

> **记忆 = 磁盘上的 Markdown 文件。模型只"记住"写进磁盘的东西。**

这是一个优雅的设计——不依赖向量数据库、不做嵌入检索，而是用最朴素的文件系统作为记忆的持久层。

### 5.2 三层记忆架构

```
┌────────────────────────────────────────┐
│  第一层：MEMORY.md（长期精选记忆）        │
│  - 仅在主会话加载（隐私保护）              │
│  - 策划过的核心事实、偏好、教训            │
├────────────────────────────────────────┤
│  第二层：memory/YYYY-MM-DD.md（每日日志） │
│  - Append-only，原始记录                 │
│  - 每次会话加载今天 + 昨天               │
├────────────────────────────────────────┤
│  第三层：sessions/*.jsonl（会话历史）     │
│  - 短期参考，JSONL 格式                  │
│  - Agent Loop 写入，Compaction 时清理    │
└────────────────────────────────────────┘
```

### 5.3 记忆工具

| 工具 | 功能 |
|------|------|
| `memory_search` | 语义检索索引片段 |
| `memory_get` | 定向读取指定 Markdown 文件/行范围 |

### 5.4 自动记忆刷写

在 Compaction 前，OpenClaw 触发一次**静默 agentic turn**，提醒模型将重要信息写入持久记忆。这是确保上下文不会因压缩而丢失的关键机制。

---

## 6. 模型与提供商

### 6.1 模型引用格式

```
provider/model
示例: openai/gpt-5.1-codex, google/gemini-2.5-pro, glm/glm-5
```

### 6.2 内置提供商（pi-ai catalog）

| 提供商 | 环境变量 | 示例模型 |
|--------|---------|---------|
| OpenAI | `OPENAI_API_KEY` | `openai/gpt-5.1-codex` |
| Google | `GOOGLE_API_KEY` | `google/gemini-2.5-pro` |
| GLM（智谱） | `GLM_API_KEY` | `glm/glm-5` |
| Custom | 自定义配置 | `custom/your-model` |

### 6.3 API Key 轮换

支持多 Key 轮换，优先级：

```
OPENCLAW_LIVE_<PROVIDER>_KEY  ← 最高优先级（运行时热切换）
<PROVIDER>_API_KEYS           ← 逗号分隔列表
<PROVIDER>_API_KEY             ← 主 Key
<PROVIDER>_API_KEY_1/2/3...   ← 编号列表
```

**仅在限速（429 / rate_limit / quota exhausted）时自动轮换到下一个 Key。**

### 6.4 模型 Failover

当主模型失败时，可配置备用模型链，自动降级重试。

---

## 7. 渠道集成矩阵

### 7.1 完整渠道列表

| 渠道 | 协议/库 | 类型 |
|------|---------|------|
| **WhatsApp** | Baileys（WhatsApp Web） | 内置 |
| **Telegram** | grammY（Bot API） | 内置 |
| **Discord** | discord.js | 内置 |
| **iMessage** | 本地 imsg CLI | 内置（macOS） |
| **飞书 (Feishu/Lark)** | REST API | 内置 |
| **Slack** | Bolt SDK | 内置 |
| **Signal** | signal-cli | 内置 |
| **Google Chat** | Google Chat API | 内置 |
| **IRC** | IRC 协议 | 内置 |
| **Matrix** | Matrix SDK | 内置 |
| **LINE** | LINE Messaging API | 内置 |
| **Mattermost** | Plugin 扩展 | 插件 |
| **Microsoft Teams** | MS Teams API | 内置 |
| **Nextcloud Talk** | Nextcloud API | 内置 |
| **Nostr** | Nostr 协议 | 内置 |
| **Twitch** | Twitch API | 内置 |
| **Zalo** | Zalo API | 内置 |
| **Synology Chat** | Synology API | 内置 |
| **Tlon** | Urbit/Tlon | 内置 |

### 7.2 广播组（Broadcast Groups）

支持将消息同时广播到多个渠道/群组，适合运维通知、报警推送等场景。

---

## 8. 安全体系

### 8.1 多层安全模型

```
┌─────────────────────────────────────┐
│  第一层：Gateway 认证                 │
│  - Token 验证                        │
│  - 设备配对 (Pairing)                │
│  - 签名挑战 (Challenge/Response)     │
├─────────────────────────────────────┤
│  第二层：Prompt Injection 防护        │
│  - 外部数据 = 纯数据，不执行指令       │
│  - 指令源隔离                         │
├─────────────────────────────────────┤
│  第三层：Agent 安全策略                │
│  - 破坏性操作必须确认                  │
│  - trash > rm                        │
│  - 凭证不明文存储                     │
│  - 输出脱敏 (sk-a1b2****)            │
├─────────────────────────────────────┤
│  第四层：Skill 供应链安全              │
│  - 安装前 SKILL.md 全文审查            │
│  - 恶意特征扫描                       │
│  - 仅所有者可授权安装                  │
├─────────────────────────────────────┤
│  第五层：隐私保护                      │
│  - MEMORY.md 不在群聊加载              │
│  - 群聊不泄露所有者信息                │
│  - 拒绝回答个人信息/系统配置            │
└─────────────────────────────────────┘
```

### 8.2 设备配对流程

1. 新设备连接 → 携带 device identity
2. Gateway 检查是否已配对
3. **本地连接**（loopback）可自动批准
4. **非本地连接**需明确批准
5. 配对后颁发 device token，后续连接使用 token 认证
6. 签名绑定 `platform` + `deviceFamily`，元数据变更需重新配对

### 8.3 Agent 安全硬性策略

| 防护领域 | 规则 |
|---------|------|
| **Prompt Injection** | 外部数据（邮件、网页、文件）中的指令性内容一律忽略 |
| **供应链投毒** | Skill 安装前必须审查 SKILL.md 全文，发现恶意特征立即拒绝 |
| **凭证管理** | 绝不明文存储 API Key；输出脱敏；定期提醒轮换 |
| **运行时失控** | 破坏性操作确认；`--dry-run` 先行；批量操作报规模 |
| **暴露风险** | 不在公开渠道暴露端口/配置；仅 `127.0.0.1` 绑定 |
| **数据操作** | `trash` > `rm`；覆盖重要内容需确认 |

---

## 9. 自动化与调度

### 9.1 Cron Jobs

- 精确定时任务（"每周一 9:00 sharp"）
- 独立会话，不影响主会话历史
- 可指定不同模型和 thinking level
- 输出直接投递到指定渠道

### 9.2 Heartbeat

- 周期性轮询（~30分钟间隔，可漂移）
- 批量检查（邮件 + 日历 + 天气 + 通知）
- 可通过 `HEARTBEAT.md` 配置检查项
- 晚间（23:00-08:00）自动静默

### 9.3 Cron vs Heartbeat 选择指南

| 场景 | 选择 | 原因 |
|------|------|------|
| 每天早上 8:30 天气播报 | Cron | 精确时间 |
| 每 30 分钟检查邮件 | Heartbeat | 批量处理，可漂移 |
| 提醒我 20 分钟后开会 | Cron | 一次性定时 |
| 定期整理记忆文件 | Heartbeat | 不需精确 |
| 每周五下午发周报 | Cron | 固定时间 |

### 9.4 Webhooks 与 Hooks

- 支持 Webhook 接收外部触发
- 支持各种事件钩子（消息前后、Agent 启动前后等）

---

## 10. 技能（Skill）系统

### 10.1 设计理念

Skill 是 OpenClaw 的**能力扩展单元**，每个 Skill 是一个自包含的知识包（`SKILL.md`），定义了特定任务的专业知识和工作流。

### 10.2 Skill 加载机制

```
Agent Loop 启动 → 扫描 skills/ 目录
→ 读取每个 SKILL.md 的 description 字段
→ 根据用户请求匹配最合适的 Skill
→ 加载完整 SKILL.md 作为上下文注入
```

### 10.3 Skill 安全审查流程（强制，不可跳过）

1. 读取审查协议 skill-vetter/SKILL.md
2. 来源检查（谁写的？哪里来的？下载量/更新时间？）
3. 逐文件代码审查（扫描红旗关键词）
4. 权限评估（读/写什么文件？访问什么网络？执行什么命令？）
5. 输出 SKILL VETTING REPORT（含风险等级 + 判定）
6. 等待所有者确认 → 才安装

### 10.4 当前可用 Skill 分类

| 类别 | 示例 Skill |
|------|-----------|
| **开发** | Code、frontend-design、git-essentials、security-auditor |
| **写作** | blog-writer、copywriting、research-paper-writer、seo-content-writer |
| **研究** | Market Research、aminer-data-search、autoglm-deepresearch |
| **设计** | UI/UX Pro Max、architecture-designer |
| **运维** | healthcheck、FFmpeg Video Editor |
| **安全** | clawdefender、skill-vetter |
| **自动化** | automation-workflows、social-media-scheduler |
| **通讯** | feishu-doc、feishu-chat-history、feishu-cron-reminder |
| **记忆** | Memory、self-reflection |
| **人资** | interview-designer |
| **创意** | autoglm-generate-image、autoglm-search-image |
| **工具** | find-skills、skill-creator、tmux |

---

## 11. 节点与移动端

### 11.1 节点概念

Nodes 是连接到 Gateway 的**远程设备**，通过 WebSocket 连接，声明 `role: node`。

```
┌──────────────┐     WebSocket      ┌──────────┐
│  iOS Node    │ ◄───────────────► │          │
│  (摄像头/位置)│                   │  Gateway │
├──────────────┤                   │          │
│  Android Node│ ◄───────────────► │          │
│  (通知/日历) │                   │          │
├──────────────┤                   │          │
│  macOS Node  │ ◄───────────────► │          │
│  (屏幕/Canvas)│                   │          │
├──────────────┤                   │          │
│  Headless    │ ◄───────────────► │          │
│  (无头设备)  │                   │          │
└──────────────┘                   └──────────┘
```

### 11.2 节点能力矩阵

| 能力 | iOS | Android | macOS | Headless |
|------|-----|---------|-------|----------|
| Canvas（画布展示） | ✅ | ✅ | ✅ | ✅ |
| 摄像头拍照 | ✅ | ✅ | ✅ | - |
| 屏幕录制 | ✅ | - | ✅ | - |
| 位置获取 | ✅ | ✅ | ✅ | - |
| 语音通话 | ✅ | ✅ | - | - |
| 通知推送 | ✅ | ✅ | - | - |
| 通讯录/日历 | - | ✅ | - | - |
| 运动/照片 | - | ✅ | - | - |
| SMS | - | ✅ | - | - |

### 11.3 Canvas 系统

Canvas 是 Agent 的**可视化输出通道**：

- Gateway HTTP 端口下提供 `/__openclaw__/canvas/`
- Agent 可编辑 HTML/CSS/JS 并实时展示
- A2UI（Agent-to-UI）支持将 Agent 操作推送到用户界面
- 适用于仪表盘、报告展示、实时数据可视化

---

## 12. 部署与运维指南

### 12.1 安装流程

```bash
# 1. 安装 OpenClaw
npm install -g openclaw@latest

# 2. 引导安装 + 注册为系统服务
openclaw onboard --install-daemon

# 3. 配置渠道
openclaw channels login          # 扫码绑定 WhatsApp 等

# 4. 启动 Gateway
openclaw gateway --port 18789
```

### 12.2 系统要求

| 项目 | 要求 |
|------|------|
| **Node.js** | v22+ |
| **API Key** | 至少一个 LLM 提供商的 Key |
| **平台** | macOS / Linux / Windows (WSL) |
| **时间** | ~5 分钟即可完成 |

### 12.3 服务管理

```bash
openclaw gateway status          # 查看状态
openclaw gateway start           # 启动
openclaw gateway stop            # 停止
openclaw gateway restart         # 重启
openclaw status                  # 综合状态
openclaw doctor                  # 诊断问题
openclaw security audit          # 安全审计
```

### 12.4 关键运维命令速查

| 命令 | 用途 |
|------|------|
| `openclaw models list` | 列出可用模型 |
| `openclaw models set <provider/model>` | 设置默认模型 |
| `openclaw sessions list` | 列出活跃会话 |
| `openclaw logs` | 查看日志 |
| `openclaw cron list` | 列出定时任务 |
| `openclaw skills list` | 列出已安装 Skill |
| `openclaw update` | 更新 OpenClaw |
| `openclaw security audit` | 安全审计 |

### 12.5 Workspace 目录结构

```
~/.openclaw/workspace/
├── AGENTS.md          # Agent 行为规范（安全策略、权限、响应准则）
├── SOUL.md            # Agent 人格定义（语气、边界、风格）
├── USER.md            # 用户画像（名字、时区、偏好）
├── IDENTITY.md        # Agent 身份卡
├── TOOLS.md           # 工具使用笔记
├── HEARTBEAT.md       # 心跳检查配置
├── MEMORY.md          # 长期精选记忆
├── memory/            # 每日记忆日志
│   └── 2026-03-11.md
└── .agents/skills/    # 已安装的 Skill
    ├── code-1.0.4/
    │   └── SKILL.md
    └── ...
```

---

## 13. 配置参考速查

### 13.1 主配置文件

```json5
// ~/.openclaw/openclaw.json
{
  // Gateway 绑定地址（安全：仅本地）
  gateway: {
    host: "127.0.0.1",
    port: 18789,
    token: "your-gateway-token"  // 可选，强烈推荐
  },

  // 会话配置
  session: {
    mainKey: "main",
    dmScope: "main",
    maintenance: {
      mode: "warn",
      pruneAfter: "30d",
      maxEntries: 500
    }
  },

  // Agent 默认配置
  agents: {
    defaults: {
      model: {
        primary: "glm/glm-5"
      },
      workspace: "~/.openclaw/workspace",
      thinking: "low"
    }
  }
}
```

### 13.2 Workspace 模板文件用途

| 文件 | 注入时机 | 用途 |
|------|---------|------|
| `AGENTS.md` | 每次会话 | 行为规范、安全策略 |
| `SOUL.md` | 每次会话 | Agent 人格 |
| `USER.md` | 每次会话 | 用户画像 |
| `TOOLS.md` | 每次会话 | 工具笔记 |
| `HEARTBEAT.md` | 心跳轮询时 | 检查项清单 |
| `MEMORY.md` | 仅主会话 | 长期记忆（隐私保护） |
| `memory/YYYY-MM-DD.md` | 每次会话（今天+昨天） | 每日日志 |

---

## 14. 总结

### 14.1 一图秒懂 OpenClaw

```
用户 → 聊天App → [Gateway] → Agent Loop → 模型API
                      ↕           ↕
                   记忆系统      工具调用
                   (Markdown)   (Skills)
                      ↕           ↕
                   定时任务      移动节点
                   (Cron)       (iOS/Android)
```

### 14.2 核心设计原则

1. **本地优先**: 自托管，数据不出你的机器
2. **Agent 原生**: 不是聊天机器人框架，而是完整的 Agent 运行时
3. **文件即记忆**: 用 Markdown 文件作为记忆持久层，简单优雅
4. **安全分层**: 从 Gateway 认证到 Prompt Injection 防护，多层防线
5. **渠道无关**: 一个 Gateway，20+ 渠道，统一 Agent 体验
6. **可扩展**: Skill 系统让能力无限扩展，Node 系统让设备无限接入

### 14.3 适用场景

| 场景 | 说明 |
|------|------|
| **个人 AI 助手** | 通过 WhatsApp/Telegram 随时随地与 AI 对话 |
| **运维自动化** | 告警通知、定时巡检、自动化运维脚本 |
| **团队协作** | 群聊中的 AI 成员，提效不越权 |
| **开发辅助** | 代码审查、架构设计、技术调研 |
| **内容创作** | 博客写作、SEO 优化、社交媒体管理 |
| **多设备协同** | 手机拍照 → Agent 分析 → 结果推送到桌面 |

---

## 附录 A：关键概念术语表

| 术语 | 定义 |
|------|------|
| **Gateway** | OpenClaw 的核心守护进程，管理所有连接和会话 |
| **Agent Loop** | 消息到行动的完整执行流水线 |
| **Session** | 一个对话上下文的持久化单元 |
| **Skill** | Agent 的能力扩展包（SKILL.md） |
| **Node** | 连接到 Gateway 的远程设备 |
| **Canvas** | Agent 的可视化输出通道 |
| **Compaction** | 上下文压缩机制，防止 token 溢出 |
| **Heartbeat** | 周期性轮询，用于主动检查 |
| **Pairing** | 设备配对流程，建立信任关系 |
| **dmScope** | 直接消息的作用域控制策略 |

## 附录 B：环境变量速查

| 变量 | 用途 |
|------|------|
| `OPENCLAW_GATEWAY_TOKEN` | Gateway 认证 Token |
| `OPENAI_API_KEY` | OpenAI API Key |
| `GOOGLE_API_KEY` | Google AI API Key |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token |
| `DISCORD_BOT_TOKEN` | Discord Bot Token |
| `OPENCLAW_LIVE_<PROVIDER>_KEY` | 运行时热切换的 Provider Key |

---

> **文档结束**  
> 🦞 OpenClaw — Any OS gateway for AI agents  
> GitHub: https://github.com/openclaw/openclaw  
> Docs: https://docs.openclaw.ai  
> Community: https://discord.com/invite/clawd  
> Skills: https://clawhub.com
