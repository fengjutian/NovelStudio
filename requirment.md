可以。下面我直接按**可以进入研发阶段的正式技术方案**来写，而不是只写一个概念方案。

# AI Novel Studio 产品技术设计文档

**文档版本：** v1.0
**产品类型：** Browser-based AI Novel Creation Platform
**后端：** Go
**数据库：** MySQL 8.4
**前端：** React + TypeScript
**核心模式：** Document-Driven + Agent Pipeline + Quality Gate

---

# 1. 产品概述

## 1.1 产品定位

AI Novel Studio 是一个基于浏览器的 AI 长篇小说创作平台。

它不是简单的：

> Prompt → AI → 小说

而是将小说创作拆解成一套可管理、可追踪、可回滚、可校验的生产流水线：

```text
创作意图
   ↓
小说策划
   ↓
世界观
   ↓
人物
   ↓
剧情
   ↓
目录
   ↓
章节细纲
   ↓
批量生成
   ↓
事实抽取
   ↓
AI润色
   ↓
一致性校验
   ↓
质量门禁
   ↓
人工审核
   ↓
发布
```

---

# 2. 产品核心理念

整个系统采用四个核心理念。

## 2.1 Document First

小说不是聊天记录，而是：

```text
Novel
 ↓
Volume
 ↓
Chapter
 ↓
Document
 ↓
Version
```

AI 围绕文档进行工作。

---

## 2.2 Story Bible

建立小说长期记忆：

```text
Story Bible
├── World
├── Characters
├── Organizations
├── Locations
├── Timeline
├── Plotlines
├── Foreshadowings
└── Facts
```

避免长篇小说出现：

* 人物 OOC
* 时间线冲突
* 世界观遗忘
* 人物关系错误
* 已死亡人物重新出现
* 伏笔丢失

---

## 2.3 Agent Pipeline

AI 不采用一个超级 Prompt，而是拆成多个 Agent：

```text
Planner
   ↓
Outliner
   ↓
Writer
   ↓
Extractor
   ↓
Polisher
   ↓
Validator
```

---

## 2.4 Quality Gate

每一章都可以像代码一样进行质量检查：

```text
Chapter
 ↓
Lint
 ↓
Test
 ↓
Quality Gate
 ↓
PASS / FAIL
```

最终形成：

> **Novel CI**

---

# 3. 产品目标

## 3.1 核心目标

用户能够：

1. 输入自然语言创作要求
2. 自动生成小说策划
3. 自动生成小说目录
4. 修改目录
5. 批量生成章节
6. 批量润色
7. 批量校验
8. 查看问题
9. AI 自动修复
10. 查看版本差异
11. 恢复历史版本
12. 导出完整小说

---

# 4. 用户角色

第一版只需要：

```text
User
```

后期：

```text
Owner
Editor
Writer
Reviewer
Viewer
```

---

# 5. 产品功能架构

```text
AI Novel Studio
│
├── Dashboard
│
├── Novel
│   ├── Overview
│   ├── Outline
│   ├── Characters
│   ├── World
│   ├── Timeline
│   ├── Plot
│   └── Foreshadowing
│
├── Writing
│   ├── Chapter Editor
│   ├── AI Copilot
│   └── Versions
│
├── AI Pipeline
│   ├── Generate
│   ├── Polish
│   └── Validate
│
├── Tasks
│
├── Quality
│
├── Export
│
└── Settings
```

---

# 6. 首页 Dashboard

展示：

```text
我的小说

┌───────────────────────────────────┐
│ 长安风云                          │
│ 历史小说 / 50万字                  │
│ 36 / 120章                        │
│ ███████████░░░░ 30%               │
│                                   │
│ [继续创作]                         │
└───────────────────────────────────┘
```

支持：

* 创建小说
* 删除小说
* 搜索小说
* 最近编辑
* 创作进度

---

# 7. 创建小说

用户输入：

```text
小说标题

创作 Prompt

小说类型

目标字数

目标章节

单章字数

写作风格
```

例如：

```text
标题：
长安风云

Prompt：

写一部唐朝历史小说……
```

点击：

> 创建小说

系统启动：

```text
Novel Planner
```

---

# 8. 小说策划

Planner 输出：

```text
NovelPlan

├── Title
├── Logline
├── Theme
├── Genre
├── Style
├── World
├── Characters
├── Organizations
├── Timeline
├── Plotlines
└── Volumes
```

用户可以：

```text
接受
重新生成
编辑
```

---

# 9. 目录系统

目录层级：

```text
Novel
 ├── Volume 1
 │    ├── Chapter 1
 │    ├── Chapter 2
 │    └── Chapter 3
 │
 ├── Volume 2
 │    ├── Chapter 20
 │    └── Chapter 21
 │
 └── Volume 3
```

章节字段：

```text
章节编号
章节标题
章节简介
剧情目标
出场人物
地点
时间
冲突
伏笔
目标字数
```

---

# 10. 章节编辑器

核心界面：

```text
┌────────────┬─────────────────────┬──────────────┐
│ 目录        │ Markdown Editor     │ AI Copilot   │
│            │                     │              │
│ 第1章       │ # 灰尘里的圣旨      │ 生成         │
│ 第2章       │                     │ 润色         │
│ 第3章       │ 沈砚站在城门下……    │ 扩写         │
│            │                     │ 校验         │
│ 第4章       │                     │              │
└────────────┴─────────────────────┴──────────────┘
```

编辑器：

> CodeMirror 6

---

# 11. AI Copilot

支持：

```text
润色
扩写
缩写
改写
续写
总结
分析
检查
```

支持选中文本操作：

```text
选中文本
 ↓
AI Menu
 ↓
润色 / 扩写 / 改写
```

---

# 12. 批量生成

用户选择：

```text
第37章 ～ 第50章
```

生成任务：

```text
Batch Generation Task
```

系统：

```text
37 ─┐
38 ─┤
39 ─┤
40 ─┤
41 ─┘
    ↓
Story State Update
    ↓
42 ─┐
43 ─┤
...
```

采用：

> Windowed Parallelism

避免所有章节完全并行导致上下文失控。

---

# 13. 章节生成 Pipeline

```text
Chapter
 ↓
Context Builder
 ↓
Prompt Compiler
 ↓
Writer Agent
 ↓
Draft
 ↓
Story Extractor
 ↓
Update Story Bible
 ↓
Validator
 ↓
Quality Gate
```

---

# 14. Context Builder

生成章节时上下文包括：

```text
Novel Settings
+
Current Volume
+
Current Chapter Outline
+
Characters
+
Character States
+
World
+
Timeline
+
Plotline
+
Foreshadowing
+
Recent Chapter Summaries
+
Relevant Facts
+
Style Guide
```

而不是把整个小说全文塞进 Prompt。

---

# 15. Story Memory

核心：

```text
Story Bible
```

包括：

### Character

```text
姓名
年龄
身份
性格
关系
状态
人物弧
```

### World

```text
国家
城市
制度
经济
军事
宗教
社会结构
```

### Timeline

```text
时间
事件
章节
人物
地点
```

### Fact

```text
Subject
Predicate
Object
Source
```

例如：

```text
王铁
死亡
第12章
```

---

# 16. AI 润色 Pipeline

```text
Draft
 ↓
Grammar
 ↓
Language
 ↓
Style
 ↓
Character
 ↓
Plot
 ↓
Rhythm
 ↓
Final
```

用户可以选择：

```text
☑ 错别字
☑ 病句
☑ 文风
☑ 人物
☑ 剧情
☑ 节奏
```

---

# 17. AI 校验

定义：

```text
Novel Linter
```

检查：

```text
CharacterConsistency
TimelineConsistency
WorldConsistency
PlotConsistency
ForeshadowingConsistency
StyleConsistency
Repetition
```

结果：

```json
{
  "score": 92,
  "status": "PASS",
  "issues": []
}
```

或者：

```json
{
  "score": 67,
  "status": "FAIL",
  "issues": [
    {
      "severity": "critical",
      "type": "timeline",
      "chapter": 38
    }
  ]
}
```

---

# 18. 质量门禁

建议：

```text
90+  Excellent
80-89 Good
70-79 Warning
<70  Failed
```

同时设置硬规则：

```text
人物死亡冲突 → FAIL

时间线冲突 → FAIL

世界观核心冲突 → FAIL
```

即：

> **不是所有问题都可以通过平均分掩盖。**

---

# 19. 版本管理

每次 AI 操作生成新版本：

```text
v1 Draft
 ↓
v2 AI Polish
 ↓
v3 Human Edit
 ↓
v4 AI Polish
```

支持：

```text
查看版本
Diff
恢复版本
创建版本
```

---

# 20. 任务中心

任务类型：

```text
NOVEL_PLAN
OUTLINE_GENERATE
CHAPTER_GENERATE
CHAPTER_POLISH
CHAPTER_VALIDATE
FACT_EXTRACT
EXPORT
```

状态：

```text
PENDING
RUNNING
SUCCESS
FAILED
CANCELLED
RETRYING
```

---

# 21. 后端技术架构

```text
Go
│
├── HTTP API
│
├── Application
│
├── Domain
│
├── Agent
│
├── Workflow
│
├── Task
│
├── Memory
│
├── Document
│
├── LLM
│
└── Repository
```

---

# 22. Go 技术栈

| 技术            | 用途      |
| ------------- | ------- |
| Go 1.25+      | 后端      |
| Gin           | HTTP    |
| sqlc          | 数据访问    |
| MySQL 8.4     | 数据库     |
| Redis         | 队列/缓存   |
| SSE           | 实时事件    |
| slog          | 日志      |
| Prometheus    | Metrics |
| OpenTelemetry | Tracing |
| Docker        | 部署      |

---

# 23. 前端技术栈

| 技术             | 用途           |
| -------------- | ------------ |
| React 19       | UI           |
| TypeScript     | 类型           |
| Vite           | 构建           |
| React Router   | 路由           |
| TanStack Query | Server State |
| Zustand        | Client State |
| Tailwind CSS   | CSS          |
| shadcn/ui      | UI组件         |
| CodeMirror 6   | 编辑器          |
| SSE            | 实时任务         |
| Vitest         | 单元测试         |
| Playwright     | E2E          |

---

# 24. 数据库架构

使用：

> MySQL 8.4 LTS

核心：

```text
users

novels
novel_settings

volumes
chapters

characters
character_states

locations
organizations

timelines
plotlines
foreshadowings

facts

documents
document_versions

ai_tasks
ai_runs

polish_results
validation_results
```

---

# 25. 核心数据库关系

```text
users
  │
  └── novels
        │
        ├── volumes
        │     └── chapters
        │
        ├── characters
        │
        ├── timelines
        │
        ├── plotlines
        │
        ├── foreshadowings
        │
        └── facts
```

---

# 26. chapters 表

核心字段：

```sql
id
novel_id
volume_id
chapter_no
title
outline
content
status
word_count
current_version
created_at
updated_at
```

建议：

```text
content LONGTEXT
```

第一版正文直接存 MySQL。

---

# 27. characters 表

```text
id
novel_id
name
profile
personality
background
current_status
created_at
updated_at
```

人物状态单独：

```text
character_states
```

记录不同章节的人物状态变化。

---

# 28. facts 表

这是 AI 长期记忆核心。

```text
id
novel_id
subject
predicate
object
source_type
source_id
confidence
created_at
```

例如：

```text
沈砚 | 身份 | 县衙书吏 | chapter | 1
王铁 | 死亡 | 第12章 | chapter | 12
```

---

# 29. Redis

Redis 不作为主数据库。

主要负责：

```text
Task Queue
Cache
Distributed Lock
Rate Limit
Event
```

结构：

```text
Go API
 ↓
Redis Queue
 ↓
Worker
 ↓
LLM
```

---

# 30. Worker

Worker 是整个 AI 系统的执行核心。

```go
type Worker struct {
    Queue Queue
    LLM LLM
    Repository Repository
}
```

处理：

```text
GenerateChapter
PolishChapter
ValidateChapter
ExtractFacts
```

支持：

```text
Retry
Timeout
Cancel
Concurrency Limit
```

---

# 31. LLM 抽象

不要在业务代码里写：

```go
openai.Chat(...)
```

应该：

```go
type LLM interface {
    Generate(
        ctx context.Context,
        req *GenerateRequest,
    ) (*GenerateResponse, error)
}
```

Provider：

```text
OpenAI
Anthropic
DeepSeek
Qwen
Gemini
Ollama
OpenAI Compatible
```

---

# 32. Model Routing

不同任务可以使用不同模型：

```text
Planner
 ↓
高推理模型

Writer
 ↓
高性价比模型

Polisher
 ↓
语言模型

Validator
 ↓
高推理模型
```

例如：

```text
Planner     → Model A
Writer      → Model B
Polisher    → Model C
Validator   → Model A
```

用户可以配置。

---

# 33. Prompt 管理

Prompt 不允许散落在 Go 代码里。

目录：

```text
prompts/

planner/
  system.md
  user.md

outline/
  system.md
  user.md

writer/
  system.md
  user.md

polish/
  system.md
  user.md

validator/
  system.md
  user.md
```

Prompt 本身也需要版本管理：

```text
writer/v1
writer/v2
writer/v3
```

AI Run 记录实际使用的 Prompt Version。

---

# 34. AI Run

每一次模型调用都记录：

```text
ai_runs

id
task_id
provider
model
prompt_version
input_tokens
output_tokens
latency
status
error
created_at
```

这样可以统计：

```text
一章花多少钱
哪个模型最好
哪个 Prompt 效果最好
哪个步骤最慢
```

---

# 35. API 设计

主要 API：

```http
POST /api/v1/novels

GET /api/v1/novels/:id

PUT /api/v1/novels/:id

DELETE /api/v1/novels/:id
```

目录：

```http
GET /api/v1/novels/:id/outline

POST /api/v1/novels/:id/outline/generate

PUT /api/v1/volumes/:id

PUT /api/v1/chapters/:id
```

章节：

```http
GET /api/v1/chapters/:id

POST /api/v1/chapters/:id/generate

POST /api/v1/chapters/:id/polish

POST /api/v1/chapters/:id/validate
```

批量：

```http
POST /api/v1/novels/:id/chapters/generate
```

请求：

```json
{
  "from": 37,
  "to": 50
}
```

---

# 36. SSE

任务：

```http
GET /api/v1/tasks/:id/events
```

事件：

```text
task.started
task.progress
chapter.started
chapter.generated
chapter.polished
chapter.validated
task.failed
task.completed
```

---

# 37. 前端页面

```text
/
├── Dashboard
│
├── /novels/new
│
├── /novels/:id
│
├── /novels/:id/outline
│
├── /novels/:id/write
│
├── /novels/:id/characters
│
├── /novels/:id/world
│
├── /novels/:id/timeline
│
├── /novels/:id/tasks
│
├── /novels/:id/quality
│
└── /settings
```

---

# 38. 编辑器状态

```text
currentNovel
currentChapter
currentSelection
editorContent
aiPanel
sidebar
taskStatus
```

Zustand 管理 UI 状态。

TanStack Query 管理服务端数据。

---

# 39. 安全

必须：

```text
HTTPS
JWT / Session
Password Hash
Rate Limit
CSRF
XSS Protection
SQL Injection Protection
API Permission
```

用户输入的 Prompt 必须经过：

```text
Validation
Sanitization
Length Limit
```

---

# 40. AI 安全

尤其需要防：

> Prompt Injection

因为小说正文本身也是 AI 的输入。

例如正文中可能出现：

```text
Ignore previous instructions...
```

因此：

```text
用户内容
小说内容
Story Bible
系统 Prompt
```

必须逻辑隔离。

---

# 41. 限流

需要针对：

```text
用户
IP
Novel
Task
LLM Provider
```

限流。

例如：

```text
单用户：
10 requests / second

AI：
5 concurrent generations
```

---

# 42. 可观测性

记录：

```text
HTTP latency
LLM latency
Task latency
Token
Error
Retry
Queue length
Worker utilization
```

核心指标：

```text
chapter_generation_success_rate

chapter_generation_latency

llm_error_rate

task_queue_depth

validation_failure_rate

average_chapter_quality
```

---

# 43. 部署

第一阶段：

```text
Docker Compose

Nginx
 │
 ├── React
 │
 └── Go
       │
       ├── MySQL
       └── Redis
```

例如：

```text
Browser
   ↓
Cloudflare
   ↓
Nginx
   ↓
Go API
   ├── MySQL
   ├── Redis
   └── LLM
```

---

# 44. 不建议第一版加入

暂时不要：

```text
Kafka
Kubernetes
Elasticsearch
Milvus
ClickHouse
微服务
Service Mesh
复杂 Workflow Engine
```

第一版：

```text
React
+
Go
+
MySQL
+
Redis
```

足够。

---

# 45. MVP 开发阶段

## Phase 1：基础平台

```text
用户
登录
小说
章节
Markdown
版本
```

---

## Phase 2：AI 创作

```text
Prompt
 ↓
Planner
 ↓
Outline
 ↓
Writer
```

---

## Phase 3：批量任务

```text
Redis
 ↓
Worker
 ↓
Batch Generation
 ↓
SSE
```

---

## Phase 4：小说记忆

```text
Characters
World
Timeline
Plot
Facts
Foreshadowing
```

---

## Phase 5：AI 质量

```text
Polisher
Validator
Quality Gate
Auto Fix
```

---

## Phase 6：商业化

```text
Subscription
AI Credits
Team
Collaboration
API
Export
```

---

# 46. MVP 最终功能

第一版只做：

```text
┌─────────────────────────────┐
│       AI Novel Studio       │
├─────────────────────────────┤
│                             │
│  创建小说                    │
│      ↓                      │
│  输入 Prompt                 │
│      ↓                      │
│  AI 小说策划                 │
│      ↓                      │
│  AI 生成目录                 │
│      ↓                      │
│  修改目录                    │
│      ↓                      │
│  批量生成章节                │
│      ↓                      │
│  Markdown 编辑器             │
│      ↓                      │
│  AI 润色                     │
│      ↓                      │
│  AI 校验                     │
│      ↓                      │
│  版本管理                    │
│                             │
└─────────────────────────────┘
```

---

# 47. V1 的核心竞争力

最终真正值得投入的不是 UI，而是这五个能力：

```text
① Story Bible
        ↓
② Context Builder
        ↓
③ Agent Pipeline
        ↓
④ Novel Memory
        ↓
⑤ Novel Quality Gate
```

其中最关键的是：

> **Novel Memory + Quality Gate**

因为普通 AI 写作工具可以很容易做到：

```text
Prompt → 生成一章
```

但很难做到：

```text
50万字
 ↓
120章
 ↓
数百个人物/事件/地点
 ↓
AI持续记忆
 ↓
前后逻辑一致
 ↓
自动发现错误
 ↓
自动修复
```

---

# 48. 产品最终形态

最终可以形成：

```text
                    AI Novel Studio

                         用户
                          │
                          ▼
                   Creative Intent
                          │
                          ▼
                   ┌─────────────┐
                   │   Planner   │
                   └──────┬──────┘
                          ▼
                   ┌─────────────┐
                   │   Outline   │
                   └──────┬──────┘
                          ▼
                    Story Bible
                          │
             ┌────────────┼────────────┐
             ▼            ▼            ▼
          Character     Timeline      Plot
             │            │            │
             └────────────┼────────────┘
                          ▼
                       Writer
                          │
                          ▼
                        Draft
                          │
                          ▼
                      Extractor
                          │
                          ▼
                    Story Memory
                          │
                          ▼
                       Polisher
                          │
                          ▼
                      Validator
                          │
                          ▼
                    Quality Gate
                       │     │
                     PASS   FAIL
                       │     │
                       │     ▼
                       │   Auto Fix
                       │     │
                       │     └───────┐
                       │             │
                       └─────────────┘
                              │
                              ▼
                          Published
```

---

# 49. 最终技术选型

**浏览器端：**

```text
React 19
TypeScript
Vite
React Router
TanStack Query
Zustand
Tailwind CSS
shadcn/ui
CodeMirror 6
SSE
Vitest
Playwright
```

**后端：**

```text
Go 1.25+
Gin
sqlc
REST
SSE
Go Worker
```

**数据：**

```text
MySQL 8.4
Redis
Markdown
```

**AI：**

```text
LLM Provider Abstraction
OpenAI
Claude
DeepSeek
Qwen
Gemini
Ollama
OpenAI Compatible
```

**运维：**

```text
Docker
Docker Compose
Nginx
Prometheus
Grafana
OpenTelemetry
```

---

# 50. 最核心的架构原则

整个项目建议牢牢记住这一条：

```text
                User Prompt
                     │
                     ▼
                  Novel
                     │
                     ▼
                Documents
                     │
                     ▼
                   Agents
                     │
                     ▼
                 Pipelines
                     │
                     ▼
                   Tasks
                     │
                     ▼
                    LLM
                     │
                     ▼
                  Results
                     │
                     ▼
              Quality Gate
                     │
                     ▼
                 Version
```

**不要让 LLM 成为系统本身。**

LLM 只是一个“推理执行器”。

真正的产品核心应该是：

> **Novel = Documents + Story Memory + Agent Pipeline + Task System + Quality Gate**

这样后面即使更换 GPT、Claude、DeepSeek、Qwen，甚至接入本地模型，整个产品架构也不需要推倒重来。
