import type { Document, KnowledgeSource, Project, ProjectList, ProjectType, SearchHit } from './types'

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
  sources: (projectId: string) => request<{ items: KnowledgeSource[]; total: number }>(`/api/v1/projects/${projectId}/knowledge/sources`),
  createSource: (projectId: string, input: { name: string; authority: KnowledgeSource['authority']; content: string }) =>
    request<{ source: KnowledgeSource }>(`/api/v1/projects/${projectId}/knowledge/sources`, { method: 'POST', body: JSON.stringify(input) }),
  searchKnowledge: (projectId: string, query: string) =>
    request<{ items: SearchHit[]; total: number }>(`/api/v1/projects/${projectId}/knowledge/search?q=${encodeURIComponent(query)}`),
}
