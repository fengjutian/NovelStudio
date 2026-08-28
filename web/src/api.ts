import type { AITask, Document, DocumentVersion, GenerationResult, KnowledgeSource, Project, ProjectList, ProjectType, SearchHit } from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!response.ok) {
    const data = await response.json().catch(() => null)
    throw new Error(data?.error?.message ?? `请求失败 (${response.status})`)
  }
  return response.status === 204 ? (undefined as T) : response.json()
}

export const api = {
  projects: () => request<ProjectList>('/api/v1/projects'),
  createProject: (input: { name: string; type: ProjectType; description: string }) =>
    request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input) }),
  documents: (projectId: string) => request<{ items: Document[]; total: number }>(`/api/v1/projects/${projectId}/documents`),
  createDocument: (projectId: string, input: { title: string; content: string }) =>
    request<{ document: Document }>(`/api/v1/projects/${projectId}/documents`, { method: 'POST', body: JSON.stringify(input) }),
  versions: (documentId: string) => request<{ items: DocumentVersion[]; total: number }>(`/api/v1/documents/${documentId}/versions`),
  saveVersion: (documentId: string, input: { content: string; reason: string; expectedVersionId: string }) =>
    request<DocumentVersion>(`/api/v1/documents/${documentId}/versions`, { method: 'POST', body: JSON.stringify(input) }),
  restoreVersion: (documentId: string, versionId: string) =>
    request<DocumentVersion>(`/api/v1/documents/${documentId}/versions/${versionId}/restore`, { method: 'POST' }),
  sources: (projectId: string) => request<{ items: KnowledgeSource[]; total: number }>(`/api/v1/projects/${projectId}/knowledge/sources`),
  createSource: (projectId: string, input: { name: string; authority: KnowledgeSource['authority']; content: string }) =>
    request<{ source: KnowledgeSource }>(`/api/v1/projects/${projectId}/knowledge/sources`, { method: 'POST', body: JSON.stringify(input) }),
  searchKnowledge: (projectId: string, query: string) =>
    request<{ items: SearchHit[]; total: number }>(`/api/v1/projects/${projectId}/knowledge/search?q=${encodeURIComponent(query)}`),
  modelStatus: () => request<{ configured: boolean; validatorCount: number; judgeConfigured: boolean; generationOperations: string[] }>('/api/v1/models/status'),
  createGenerationTask: (projectId: string, input: { operation: string; instruction: string; title: string; documentId: string; knowledgeQuery: string }) =>
    request<AITask<GenerationResult>>(`/api/v1/projects/${projectId}/generation-tasks`, { method: 'POST', body: JSON.stringify(input) }),
  createValidationTask: (projectId: string, input: { text: string; task: string; knowledgeQuery: string; dimensions: string[] }) =>
    request<AITask>(`/api/v1/projects/${projectId}/validation-tasks`, { method: 'POST', body: JSON.stringify(input) }),
  task: <T = import('./types').PipelineResult>(taskId: string) => request<AITask<T>>(`/api/v1/tasks/${taskId}`),
  tasks: () => request<{ items: AITask[]; total: number }>('/api/v1/tasks'),
  cancelTask: (taskId: string) => request<AITask>(`/api/v1/tasks/${taskId}/cancel`, { method: 'POST' }),
}
