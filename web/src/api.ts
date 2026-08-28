import type { Project, ProjectList, ProjectType } from './types'

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
}
