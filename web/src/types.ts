export type ProjectType = 'NOVEL' | 'MOVIE_COMMENTARY' | 'TECHNICAL_DOCUMENT'

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
