# AI Content Studio

面向小说、电影解说和技术文档的结构化 AI 内容创作平台。平台以文档版本为基础，通过知识库、多模型工作流、证据校验和质量门禁提高长篇内容的一致性与准确性。

## 当前可用能力

- 通用项目模型：小说、电影解说、技术文档
- 按项目模板初始化的内容树
- 项目列表、创建、详情、删除和内容树 API
- React 项目工作台和创建项目界面
- 文档不可变版本、版本历史和非破坏性恢复
- Markdown 文档编辑、手动/自动保存和乐观并发控制
- 知识来源录入、结构分块、项目隔离检索和权威来源排序
- OpenAI-Compatible 模型适配与 JSON Schema 结构化输出
- Planner、Outliner、Writer、Polisher 内容生成任务
- 基于知识证据生成并自动创建文档或版本
- 外部版本化 Prompt、AI Run 持久化、自动修复与再校验
- 结构化事实抽取和人工审核
- 内容树编辑、窗口式批量生成、知识文件上传
- 多 Validator 并行独立校验、重大分歧 Judge 仲裁
- 知识证据注入、评分聚合和硬规则质量门禁
- 后台任务状态机、取消、结果保存和 SSE 断线续传
- 零外部依赖的内存存储，便于先验证产品闭环

配置 `MYSQL_DSN` 后，项目、文档、知识、AI Run、后台任务和 SSE 事件均使用 MySQL 8.4 持久化；未配置时自动使用内存仓储，数据会在 API 重启后重置。

## 本地运行

需要 Go 1.26+、Node.js 22+。

启动 API：

```bash
go run ./cmd/api
```

API 默认监听 `http://localhost:8080`，健康检查为：

```text
GET http://localhost:8080/healthz
```

启动 Web：

```bash
cd web
npm install
npm run dev
```

浏览器打开 `http://localhost:5173`。Vite 会将 `/api` 请求代理到 Go API。

## 原生 MySQL 持久化

项目不依赖 Docker。请先在本机或服务器安装并启动 MySQL 8.4，然后创建数据库和应用账号：

```sql
CREATE DATABASE contentstudio CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE USER 'contentstudio'@'localhost' IDENTIFIED BY 'replace-with-a-strong-password';
GRANT ALL PRIVILEGES ON contentstudio.* TO 'contentstudio'@'localhost';
FLUSH PRIVILEGES;
```

设置环境变量后直接启动 Go API。PowerShell 示例：

```powershell
$env:MYSQL_DSN='contentstudio:replace-with-a-strong-password@tcp(127.0.0.1:3306)/contentstudio?parseTime=true&charset=utf8mb4'
go run ./cmd/api
```

应用启动时会自动执行嵌入式迁移。手工审查用迁移文件位于 `db/migrations`。DSN 必须包含 `parseTime=true`，例如：

```env
MYSQL_DSN=contentstudio:contentstudio@tcp(localhost:3306)/contentstudio?parseTime=true&charset=utf8mb4
```

## API

```text
GET    /healthz
GET    /api/v1/project-types
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/{id}
DELETE /api/v1/projects/{id}
GET    /api/v1/projects/{id}/tree
GET    /api/v1/projects/{id}/documents
POST   /api/v1/projects/{id}/documents
GET    /api/v1/documents/{id}
GET    /api/v1/documents/{id}/versions
POST   /api/v1/documents/{id}/versions
POST   /api/v1/documents/{id}/versions/{versionId}/restore
GET    /api/v1/projects/{id}/knowledge/sources
POST   /api/v1/projects/{id}/knowledge/sources
GET    /api/v1/projects/{id}/knowledge/search?q=关键词
GET    /api/v1/models/status
POST   /api/v1/projects/{id}/validate
GET    /api/v1/projects/{id}/tasks
POST   /api/v1/projects/{id}/validation-tasks
POST   /api/v1/projects/{id}/generation-tasks
POST   /api/v1/projects/{id}/quality-generation-tasks
POST   /api/v1/projects/{id}/batch-generation-tasks
POST   /api/v1/projects/{id}/fact-extraction-tasks
GET    /api/v1/projects/{id}/knowledge/facts
PUT    /api/v1/facts/{id}/status
POST   /api/v1/projects/{id}/knowledge/files
POST   /api/v1/projects/{id}/nodes
PUT    /api/v1/nodes/{id}
DELETE /api/v1/nodes/{id}
GET    /api/v1/tasks/{id}
POST   /api/v1/tasks/{id}/cancel
GET    /api/v1/tasks/{id}/events
```

创建文档版本时可以携带编辑器加载时的版本 ID：

```json
{
  "content": "更新后的 Markdown 内容",
  "reason": "HUMAN_EDIT",
  "expectedVersionId": "ver_xxx"
}
```

如果服务器当前版本已经变化，接口返回 `409 VERSION_CONFLICT`，客户端应保留本地修改并让用户决定是否重新加载。

创建项目示例：

```json
{
  "name": "产品 API 使用指南",
  "type": "TECHNICAL_DOCUMENT",
  "description": "基于官方接口定义生成并校验文档"
}
```

配置多模型校验：

```env
LLM_BASE_URL=https://your-openai-compatible-provider.example/v1
LLM_API_KEY=your-server-side-key
VALIDATOR_MODELS=validator-model-a,validator-model-b
JUDGE_MODEL=judge-model
PLANNER_MODEL=planner-model
OUTLINER_MODEL=outline-model
WRITER_MODEL=writer-model
POLISHER_MODEL=polisher-model
```

`VALIDATOR_MODELS` 支持逗号分隔的多个模型，它们会并行、独立校验。只有 Critical/Major 问题存在模型分歧时才调用 Judge，以控制成本。API Key 只由 Go 服务读取，不会通过状态接口或 Web 页面返回。

## 架构原则

```text
Project Template
      ↓
Content Tree + Knowledge Base
      ↓
Context Builder
      ↓
Writer Model
      ↓
Document Version + Citations
      ↓
Rule Validation + Validator Models
      ↓
Judge + Quality Gate
```

模型只是推理执行器。项目、文档、知识来源、引用、任务和质量结果均由平台自身持久化并保留审计信息。

## 下一阶段

1. 独立 Worker、任务租约、超时回收和自动重试
2. 文档 Diff、乐观锁、编辑器自动保存与引用定位
3. PDF/DOCX 文件摄取、全文检索和语义检索
4. 模型注册持久化、不同 Provider 路由和费用预算
5. 校验结果持久化、人工覆盖和自动修复闭环
6. Redis Worker、任务状态机与 SSE
