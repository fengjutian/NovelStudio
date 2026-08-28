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
