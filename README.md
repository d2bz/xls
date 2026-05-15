# XLS 短视频平台

一个基于 Go + 微服务架构的短视频平台，集成了 AI 推荐、内容审核、智能标签等能力。

## 项目概述

XLS 是一个类似抖音/快手的短视频社交平台后端系统，支持用户上传视频、智能推荐、关注社交、弹幕互动等核心功能。系统采用 Go 语言开发，基于 go-zero 框架构建微服务架构，并深度集成了 AI 能力（LLM 推理、向量检索）来提升内容分发效率。

## 技术栈

| 层级 | 技术选型 |
|------|---------|
| **语言** | Go 1.21+ |
| **框架** | go-zero (API + RPC + MQ) |
| **AI 运行时** | Eino  |
| **向量数据库** | Milvus |
| **消息队列** | Kafka |
| **数据库** | MySQL |
| **ORM** | GORM + go-redis |
| **API 协议** | Protobuf (gRPC) |
| **容器化** | Docker / Docker Compose |

## 架构图

```
                                    ┌─────────────────────────────┐
                                    │       AI Agent Service       │
                                    │  (意图路由 / 技能执行 / LLM)  │
                                    └──────────────┬──────────────┘
                                                   │ gRPC

┌──────────┐    gRPC     ┌──────────┐   Publish   ┌──────────────┐
│ User API │─────────────│ User RPC │─────────────│ User MQ      │
└────┬─────┘             └────┬─────┘             └──────┬───────┘
     │                        │                         │
     │ Publish                │ Publish                 │ Consume
     ▼                        ▼                         ▼
┌──────────┐   Publish   ┌──────────┐             ┌──────────────┐
│ Video API│─────────────│ Video RPC│─────────────│ Video MQ     │
└────┬─────┘             └────┬─────┘             └──────┬───────┘
     │                        │                         │
     │                        │                         │ Consume → Embedding / Milvus
     │                        │                         │ Consume → Tag / AI Analysis
     │                        │                         │ Consume → Like Sync
     │                        │                         │
     │ Subscribe             │ Subscribe               │
     ▼                        ▼                         ▼
┌──────────┐             ┌──────────┐             ┌──────────────┐
│ Follow   │             │ Core     │             │ Kafka        │
│ RPC      │             │ Service  │             │              │
└──────────┘             └────┬─────┘             └──────────────┘
                              │
                              │ Query Hot List
                              ▼
                       ┌──────────────┐
                       │  Hot Video   │
                       │  Recommendation │
                       └──────────────┘
```

## 核心模块

### 1. 用户服务 (`app/user/`)

- **MQ Consumer** — 消费 Kafka 消息，处理用户注册、积分变动等事件
- **数据模型** — `User` 实体（用户信息、积分、认证状态等）

### 2. 视频服务 (`app/video/`)

- **Video RPC** — 视频 CRUD、发布、点赞、标签筛选、热度推荐等核心接口
- **Video MQ Consumer** — 消费视频上传事件，触发后续处理流程：
  - **Embedding 生成** — 调用 OpenAI Embedding 接口，将视频标题/描述/标签转为向量，存入 Milvus 向量数据库
  - **AI 标签生成** — 调用 LLM 自动提取和生成视频标签
  - **点赞同步** — 维护点赞计数的最终一致性

### 3. 关注服务 (`app/follow/`)

- **Follow RPC** — 关注/取关、粉丝列表、关注列表查询

### 4. Core 服务 (`app/core/`)

- **热度视频推荐** — 基于实时数据（播放量、点赞、评论）计算视频热度，实现首页推荐流

### 5. AI Agent 服务 (`app/agent/`)

- **API 层** (`api/`) — HTTP 接口，负责对话路由
- **RPC 层** (`rpc/`) — gRPC 服务端，处理 Agent 核心逻辑：
  - **Intent Router** — 意图分类与路由，将用户请求分发到对应技能
  - **Skill System** — 技能引擎，支持内容审核、视频平台指南等可扩展技能
  - **Workflow Engine** — 工作流编排，支持简单任务、复杂图编排、视频分析、语义推荐等
  - **MCP Client** — 接入 MCP 协议扩展工具能力
  - **Memory** — 使用滑动窗口+摘要压缩实现记忆压缩，基于 MySQL 的会话历史存储

## 服务通信方式

| 路径 | 类型 | 用途 |
|------|------|------|
| User RPC ↔ User MQ | Kafka Publish | 用户事件发布与消费 |
| Video RPC ↔ Video MQ | Kafka Publish | 视频事件发布与消费 |
| Video API → Video RPC | gRPC | 视频服务调用 |
| User API → User RPC | gRPC | 用户服务调用 |
| Agent API → Agent RPC | gRPC | AI Agent 服务调用 |
| Video MQ → Milvus | Direct | 向量存储与检索 |

## API 一览

### 视频服务

| 方法 | 描述 |
|------|------|
| `PublishVideo` | 发布视频 |
| `GetVideoList` | 获取视频列表 |
| `GetVideosByTag` | 按标签筛选视频 |
| `GetVideosByDimensions` | 多维度筛选视频 |

### 关注服务

| 方法 | 描述 |
|------|------|
| `FollowList` | 获取关注列表 |
| `FansList` | 获取粉丝列表 |

### AI Agent

| 方法 | 描述 |
|------|------|
| `Chat` | 发送对话消息 |

## 配置说明

各服务配置文件位于 `etc/` 目录下（格式：`服务名.yaml`），核心配置项包括：

- **mysql** — 数据库连接（host、port、user、password、database）
- **redis** — 缓存连接
- **kafka** — Broker 地址与 Topic 配置
- **milvus** — Milvus 服务地址与 Collection 配置
- **openAI** — API Key 与 Endpoint（用于 Embedding 和 LLM 调用）
- **zrpc** — gRPC 服务监听地址

## 环境准备

```bash
# 依赖安装
go mod tidy

# Protobuf 代码生成（项目已预生成，修改 .proto 后需重新生成）
# 需要安装 protoc 及对应插件
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       *.proto
```

## 编译与运行

```bash
# 编译 User API
cd app/user && go build -o user-api etc/user-api.yaml

# 编译 Video API
cd app/video && go build -o video-api etc/video-api.yaml

# 编译 Video RPC
cd app/video/rpc && go build -o video-rpc etc/video.yaml

# 编译 User MQ
cd app/user/mq && go build -o user-mq etc/user-mq.yaml

# 编译 Video MQ
cd app/video/mq && go build -o video-mq etc/video-mq.yaml

# 编译 Agent API
cd app/agent/api && go build -o agent-api etc/agent.yaml

# 编译 Agent RPC
cd app/agent/rpc && go build -o agent-rpc etc/agent.yaml
```

## 项目结构

```
app/
├── agent/               # AI Agent 服务（意图路由、技能引擎、工作流）
│   ├── api/             # HTTP API 层
│   └── rpc/             # gRPC 服务端
├── core/                # 核心服务（热度推荐）
│   └── internal/logic/
├── follow/              # 关注服务 RPC
│   └── rpc/internal/logic/
├── user/                # 用户服务
│   ├── api/             # HTTP API 层
│   ├── mq/              # MQ Consumer
│   └── rpc/             # gRPC 服务端
└── video/               # 视频服务
    ├── mq/              # MQ Consumer（Embedding / Tag / Like）
    └── rpc/             # gRPC 服务端 + Video Model

pkg/
└── embedding/           # 通用 Embedding 工具（OpenAI）

go.mod                   # 依赖管理
go.sum
```

## 设计亮点

- **AI 深度集成** — Embedding 向量化存储、AI 标签生成、智能对话 Agent，全流程无感 AI
- **事件驱动架构** — Kafka MQ 解耦视频上传与后续处理，支持弹性扩展
- **向量检索** — Milvus 实现高效相似视频召回，支撑语义推荐场景
- **微服务拆分** — 独立部署、独立扩展，服务边界清晰
- **可扩展技能系统** — Agent 模块支持动态注册 Skill，可快速扩展新能力
- **会话记忆持久化** — Milvus 存储 Agent 对话历史，支持多轮上下文
