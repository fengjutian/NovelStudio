export type ProjectType = string

export interface ContentType {
  code: ProjectType
  name: string
  icon: string
  accent: string
  description: string
  prompt: string
  createdAt: string
  updatedAt: string
}

export interface Project {
  id: string
  name: string
  type: ProjectType
  description: string
  status: string
  createdAt: string
  updatedAt: string
}

export interface ProjectList {
  items: Project[]
  total: number
}

export interface Document {
  id: string
  projectId: string
  title: string
  currentVersionId: string
  versionCount: number
  updatedAt: string
}

export interface DocumentVersion {
  id: string
  documentId: string
  parentVersionId?: string
  versionNumber: number
  content: string
  contentHash: string
  reason: string
  authorType: string
  createdAt: string
}

export interface DocumentDiff { fromVersionId:string; toVersionId:string; added:number; deleted:number; lines:Array<{type:'UNCHANGED'|'ADDED'|'DELETED';content:string;oldLine?:number;newLine?:number}> }

export interface KnowledgeSource {
  id: string
  name: string
  sourceType: string
  version?: string
  authority: 'OFFICIAL' | 'VERIFIED' | 'INTERNAL' | 'REFERENCE'
  status: string
  createdAt: string
}

export interface SearchHit {
  chunk: { id: string; content: string; position: number }
  source: KnowledgeSource
  score: number
  matchType: string
}
export interface ContentNode { id:string; projectId:string; parentId?:string; documentId?:string; nodeType:string; title:string; position:number; metadata?:Record<string,unknown> }
export interface OutlineItem { title:string; level:number; nodeType:string }
export interface Fact { id:string; projectId:string; subject:string; predicate:string; object:string; confidence:number; status:string; createdAt:string }
export interface MemoryEntry { id:string;projectId:string;type:'CHARACTER'|'PLACE'|'TIMELINE'|'PLOT'|'FORESHADOW';name:string;summary:string;status:string;attributes?:Record<string,unknown>;createdAt:string;updatedAt:string }
export interface MemorySuggestion {type:MemoryEntry['type'];name:string;summary:string;status:string;attributes?:Record<string,unknown>}
export interface MemoryExtractionResult {memories:MemorySuggestion[];generation:GenerationResult['generation'];documentId:string}
export interface StorylineAnalysisResult {memories:MemorySuggestion[];generation:GenerationResult['generation'];documentCount:number}
export interface LocalModelProvider {baseUrl:string;model:string;enabled:boolean;hasApiKey:boolean}
export interface LocalModelConfig {activeProvider:'deepseek'|'minimax';deepseek:LocalModelProvider;minimax:LocalModelProvider;path:string}
export interface KnowledgeFile {id:string;projectId:string;name:string;extension:string;mimeType:string;size:number;status:'INDEXED'|'STORED'|'FAILED';sourceId?:string;createdAt:string}

export interface AIRun { id:string; projectId:string; taskId?:string; role:string; provider:string; model:string; promptVersion:string; requestId?:string; inputTokens:number; outputTokens:number; latencyMs:number; status:string; error?:string; createdAt:string }
export interface AIRunList { items:AIRun[]; total:number; stats:{inputTokens:number;outputTokens:number;latencyMs:number} }
export interface QualityRecord { id:string; projectId:string; documentId?:string; versionId?:string; textHash:string; score:number; verdict:string; gateStatus:'PASS'|'WARNING'|'FAIL'; result:PipelineResult; createdAt:string }

export interface ValidationIssue {
  id: string
  type: string
  severity: 'CRITICAL' | 'MAJOR' | 'MINOR'
  claim: string
  explanation: string
  suggestedFix: string
  evidenceIds: string[]
  confidence: number
}

export interface PipelineResult {
  result: { score: number; verdict: string; dimensions: Record<string, number>; issues: ValidationIssue[] }
  disagreements: number
  gate: { status: 'PASS' | 'WARNING' | 'FAIL'; reasons: string[] }
  runs: Array<{ role: string; provider: string; model: string; latencyMs: number; status: string }>
}

export interface GenerationResult {
  generation: { content: string; operation: string; promptVersion: string; provider: string; model: string; inputTokens: number; outputTokens: number; latencyMs: number; evidenceIds: string[] }
  document?: Document
  documentId?: string
  version?: DocumentVersion
}

export interface QualityGenerationResult {
  workflow: { content:string; generation:GenerationResult['generation']; validation:PipelineResult; repairs:GenerationResult['generation'][]; attempts:number }
  document: Document
  version: DocumentVersion
}

export interface AITask<T = PipelineResult> {
  id: string
  projectId: string
  type: string
  status: 'PENDING' | 'RUNNING' | 'SUCCESS' | 'FAILED' | 'CANCELLED'
  progress: number
  message: string
  result?: T
  error?: string
  createdAt: string
}
