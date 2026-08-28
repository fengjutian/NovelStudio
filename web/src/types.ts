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
