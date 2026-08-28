# AI Content Studio

面向小说、电影解说和技术文档的结构化 AI 内容创作平台。平台以文档版本为基础，通过知识库、多模型工作流、证据校验和质量门禁提高长篇内容的一致性与准确性。

## 当前可用能力

- 通用项目模型：小说、电影解说、技术文档
- 按项目模板初始化的内容树
- 项目列表、创建、详情、删除和内容树 API
- React 项目工作台和创建项目界面
- 文档不可变版本、版本历史和非破坏性恢复
- 知识来源录入、结构分块、项目隔离检索和权威来源排序
- OpenAI-Compatible 模型适配与 JSON Schema 结构化输出
- 多 Validator 并行独立校验、重大分歧 Judge 仲裁
- 知识证据注入、评分聚合和硬规则质量门禁
- 零外部依赖的内存存储，便于先验证产品闭环

当前使用内存仓储，数据在 API 重启后会重置；MySQL、Redis 和真实模型调用将在后续增量接入。

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
```

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

1. MySQL 迁移、sqlc Repository 和乐观锁
2. 文档 Diff、编辑器自动保存与引用定位
3. PDF/DOCX 文件摄取、全文检索和语义检索
4. 模型注册持久化、不同 Provider 路由和费用预算
5. 校验结果持久化、人工覆盖和自动修复闭环
6. Redis Worker、任务状态机与 SSE
