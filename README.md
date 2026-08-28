# AI Content Studio

面向小说、电影解说和技术文档的结构化 AI 内容创作平台。平台以文档版本为基础，通过知识库、多模型工作流、证据校验和质量门禁提高长篇内容的一致性与准确性。

## 当前可用能力

- 通用项目模型：小说、电影解说、技术文档
- 按项目模板初始化的内容树
- 项目列表、创建、详情、删除和内容树 API
- React 项目工作台和创建项目界面
- 知识来源、结构化事实、模型角色、模型路由和校验结果领域定义
- 零外部依赖的内存存储，便于先验证产品闭环

当前属于第一个可运行开发增量。数据在 API 重启后会重置；MySQL、Redis、真实模型调用和完整编辑器将在后续增量接入。

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
```

创建项目示例：

```json
{
  "name": "产品 API 使用指南",
  "type": "TECHNICAL_DOCUMENT",
  "description": "基于官方接口定义生成并校验文档"
}
```

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
2. 文档、不可变版本、Diff 与恢复
3. 知识文件摄取、分块、全文检索和引用
4. OpenAI-Compatible Provider、模型注册与路由
5. 双 Validator、Judge 仲裁和质量门禁
6. Redis Worker、任务状态机与 SSE
