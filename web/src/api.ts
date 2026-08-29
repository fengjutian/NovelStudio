import type { AIRunList, AITask, ContentNode, Document, DocumentDiff, DocumentVersion, Fact, GenerationResult, KnowledgeFile, KnowledgeSource, LocalModelConfig, MemoryEntry, OutlineItem, PipelineResult, Project, ProjectList, ProjectType, QualityGenerationResult, QualityRecord, SearchHit } from './types'

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
  dashboardStats:()=>request<{projects:number;documents:number;knowledgeSources:number;pendingIssues:number;runningTasks:number;averageQualityScore:number}>('/api/v1/dashboard/stats'),
  createProject: (input: { name: string; type: ProjectType; description: string }) =>
    request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input) }),
  documents: (projectId: string) => request<{ items: Document[]; total: number }>(`/api/v1/projects/${projectId}/documents`),
  createDocument: (projectId: string, input: { title: string; content: string }) =>
    request<{ document: Document }>(`/api/v1/projects/${projectId}/documents`, { method: 'POST', body: JSON.stringify(input) }),
  versions: (documentId: string) => request<{ items: DocumentVersion[]; total: number }>(`/api/v1/documents/${documentId}/versions`),
  diffVersions: (documentId:string,from:string,to:string) => request<DocumentDiff>(`/api/v1/documents/${documentId}/diff?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`),
  exportURL: (projectId:string) => `/api/v1/projects/${projectId}/export.md`,
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
  localModelConfig:()=>request<LocalModelConfig>('/api/v1/models/local-config'),
  saveLocalModelConfig:(input:unknown)=>request<{config:LocalModelConfig;restartRequired:boolean}>('/api/v1/models/local-config',{method:'PUT',body:JSON.stringify(input)}),
  createGenerationTask: (projectId: string, input: { operation: string; instruction: string; title: string; documentId: string; knowledgeQuery: string; content?:string; save?:boolean }) =>
    request<AITask<GenerationResult>>(`/api/v1/projects/${projectId}/generation-tasks`, { method: 'POST', body: JSON.stringify(input) }),
  createValidationTask: (projectId: string, input: { text: string; task: string; knowledgeQuery: string; dimensions: string[] }) =>
    request<AITask>(`/api/v1/projects/${projectId}/validation-tasks`, { method: 'POST', body: JSON.stringify(input) }),
  task: <T = PipelineResult>(taskId: string) => request<AITask<T>>(`/api/v1/tasks/${taskId}`),
  tasks: () => request<{ items: AITask[]; total: number }>('/api/v1/tasks'),
  cancelTask: (taskId: string) => request<AITask>(`/api/v1/tasks/${taskId}/cancel`, { method: 'POST' }),
  retryTask: (taskId:string) => request<AITask>(`/api/v1/tasks/${taskId}/retry`,{method:'POST'}),
  tree: (projectId:string) => request<{items:ContentNode[]}>(`/api/v1/projects/${projectId}/tree`),
  createNode: (projectId:string,input:{parentId?:string;nodeType:string;title:string;position:number}) => request<ContentNode>(`/api/v1/projects/${projectId}/nodes`,{method:'POST',body:JSON.stringify(input)}),
  importOutline: (projectId:string,input:{content:string;parentId?:string;preview:boolean}) => request<{items:OutlineItem[]|ContentNode[];total:number}>(`/api/v1/projects/${projectId}/outline-import`,{method:'POST',body:JSON.stringify(input)}),
  updateNode: (id:string,input:{title:string;position:number;metadata?:Record<string,unknown>;documentId?:string}) => request<ContentNode>(`/api/v1/nodes/${id}`,{method:'PUT',body:JSON.stringify(input)}),
  deleteNode: (id:string) => request<void>(`/api/v1/nodes/${id}`,{method:'DELETE'}),
  batchGenerate: (projectId:string,input:{nodeIds:string[];instruction:string;knowledgeQuery:string;windowSize:number}) => request<AITask<GenerationResult>>(`/api/v1/projects/${projectId}/batch-generation-tasks`,{method:'POST',body:JSON.stringify(input)}),
  qualityGenerate: (projectId:string,input:{instruction:string;title:string;knowledgeQuery:string;maxRepairs:number}) => request<AITask<QualityGenerationResult>>(`/api/v1/projects/${projectId}/quality-generation-tasks`,{method:'POST',body:JSON.stringify(input)}),
  aiRuns: (projectId:string) => request<AIRunList>(`/api/v1/projects/${projectId}/ai-runs`),
  qualityResults: (projectId:string) => request<{items:QualityRecord[];total:number}>(`/api/v1/projects/${projectId}/quality-results`),
  facts: (projectId:string) => request<{items:Fact[];total:number}>(`/api/v1/projects/${projectId}/knowledge/facts`),
  updateFact: (id:string,status:string) => request<Fact>(`/api/v1/facts/${id}/status`,{method:'PUT',body:JSON.stringify({status})}),
  extractFacts: (projectId:string,documentId:string) => request<AITask>(`/api/v1/projects/${projectId}/fact-extraction-tasks`,{method:'POST',body:JSON.stringify({documentId})}),
  memories:(projectId:string,type='')=>request<{items:MemoryEntry[];total:number}>(`/api/v1/projects/${projectId}/memories?type=${encodeURIComponent(type)}`),
  createMemory:(projectId:string,input:{type:MemoryEntry['type'];name:string;summary:string;status:string;attributes?:Record<string,unknown>})=>request<MemoryEntry>(`/api/v1/projects/${projectId}/memories`,{method:'POST',body:JSON.stringify(input)}),
  deleteMemory:(id:string)=>request<void>(`/api/v1/memories/${id}`,{method:'DELETE'}),
  uploadKnowledge: async (projectId:string,file:File,authority:string) => { const form=new FormData();form.append('file',file);form.append('authority',authority);const response=await fetch(`/api/v1/projects/${projectId}/knowledge/files`,{method:'POST',body:form});if(!response.ok)throw new Error((await response.json().catch(()=>null))?.error?.message??'上传失败');return response.json() as Promise<{file:KnowledgeFile;chunkCount:number}> },
  knowledgeFiles:(projectId='')=>request<{items:KnowledgeFile[];total:number}>(`/api/v1/knowledge/files?projectId=${encodeURIComponent(projectId)}`),
  deleteKnowledgeFile:(id:string)=>request<void>(`/api/v1/knowledge/files/${id}`,{method:'DELETE'}),
  knowledgeFileDownloadURL:(id:string)=>`/api/v1/knowledge/files/${id}/download`,
}
