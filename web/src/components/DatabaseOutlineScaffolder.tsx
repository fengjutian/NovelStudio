import { useMemo } from 'react'
import { OutlineScaffolderPanel } from '@next-work-dashboard/outline-scaffolder/react'
import type { OutlineScaffolderAdapter } from '@next-work-dashboard/outline-scaffolder/react'
import '@next-work-dashboard/outline-scaffolder/styles.css'
import { api } from '../api'
import type { Document, GenerationResult, Project } from '../types'

type Result<T = unknown> = { success: boolean; data?: T; error?: string }
const ok = <T,>(data?: T): Result<T> => ({ success: true, data })
const fail = (error: unknown): Result => ({ success: false, error: error instanceof Error ? error.message : String(error) })
const delay = (milliseconds: number) => new Promise((resolve) => window.setTimeout(resolve, milliseconds))

export function DatabaseOutlineScaffolder({ project }: { project: Project }) {
  const adapter = useMemo<OutlineScaffolderAdapter>(() => {
    const root = { path: project.id, name: project.name }
    const documents = async () => (await api.documents(project.id)).items
    const find = async (path: string): Promise<Document | undefined> => (await documents()).find((item) => item.title === path)

    async function readTextFile(_root: string, path: string) {
      try {
        const document = await find(path)
        if (!document) return fail('NOT_FOUND')
        const versions = await api.versions(document.id)
        const version = versions.items.find((item) => item.id === document.currentVersionId) ?? versions.items[0]
        return version ? ok({ content: version.content, modifiedAt: document.currentVersionId }) : fail('NOT_FOUND')
      } catch (error) { return fail(error) }
    }

    async function writeTextFile(_root: string, path: string, content: string) {
      try {
        const document = await find(path)
        if (document) await api.saveVersion(document.id, { content, reason: 'OUTLINE_SCAFFOLDER', expectedVersionId: document.currentVersionId })
        else await api.createDocument(project.id, { title: path, content })
        return ok({ path, modifiedAt: new Date().toISOString() })
      } catch (error) { return fail(error) }
    }

    async function listFiles() {
      try {
        return ok((await documents()).map((item) => ({ type: 'file', path: item.title, name: item.title.split('/').pop(), modifiedAt: item.currentVersionId })))
      } catch (error) { return fail(error) }
    }

    async function mutateFiles(_root: string, mutations: Array<{ kind?: string; operation?: string; path: string; content?: string; contentBase64?: string }>) {
      try {
        for (const mutation of mutations) {
          if (mutation.kind === 'delete' || mutation.operation === 'delete') {
            const document = await find(mutation.path)
            if (document) await api.deleteDocument(document.id)
            continue
          }
          await writeTextFile(project.id, mutation.path, mutation.content ?? `data:application/octet-stream;base64,${mutation.contentBase64 ?? ''}`)
        }
        return ok()
      } catch (error) { return fail(error) }
    }

    async function llmChat(payload: { body?: { messages?: Array<{ role?: string; content?: string }> } }) {
      try {
        const messages = payload.body?.messages ?? []
        const prompt = messages.map((message) => `${message.role ?? 'user'}: ${message.content ?? ''}`).join('\n\n')
        const operation = /目录|大纲|outline/i.test(prompt) ? 'OUTLINE' : /润色|改写|修订|polish|revise/i.test(prompt) ? 'POLISH' : 'WRITE'
        const task = await api.createGenerationTask(project.id, {
          operation,
          instruction: prompt,
          title: `${project.name} AI 助写`,
          documentId: '',
          knowledgeQuery: '',
          content: '',
          save: false,
        })
        for (let attempt = 0; attempt < 240; attempt += 1) {
          const current = await api.task<GenerationResult>(task.id)
          if (current.status === 'SUCCESS' && current.result) {
            return { ok: true, status: 200, data: { choices: [{ message: { role: 'assistant', content: current.result.generation.content }, finish_reason: 'stop' }] } }
          }
          if (current.status === 'FAILED' || current.status === 'CANCELLED') return { ok: false, status: 422, error: current.error ?? 'AI 助写失败' }
          await delay(500)
        }
        return { ok: false, status: 504, error: 'AI 助写等待超时，请稍后重试' }
      } catch (error) {
        return { ok: false, status: 500, error: error instanceof Error ? error.message : String(error) }
      }
    }

    const workspace = {
      openFolder: async () => root,
      readTextFile,
      writeTextFile,
      readBinaryFile: readTextFile,
      writeBinaryFile: async (_root: string, path: string, value: string) => writeTextFile(project.id, path, `data:application/octet-stream;base64,${value}`),
      listFiles: async (_root: string) => listFiles(),
      listDirectory: async (_root: string, directory = '') => {
        const listed = await listFiles()
        if (!listed.success) return listed
        const prefix = directory ? `${directory.replace(/\/$/, '')}/` : ''
        const names = new Map<string, { name: string; type: 'file' | 'directory' }>()
        for (const entry of (listed.data as Array<{ path: string }> | undefined) ?? []) {
          if (!entry.path.startsWith(prefix)) continue
          const [name, ...tail] = entry.path.slice(prefix.length).split('/')
          if (name) names.set(name, { name, type: tail.length ? 'directory' : 'file' })
        }
        return ok([...names.values()])
      },
      createDirectory: async () => ok(),
      mutateFiles,
      reauthorize: async () => ok(root),
      gitStatus: async () => fail('数据库项目不支持 Git 操作'),
      gitInit: async () => fail('数据库项目不支持 Git 操作'),
      gitStage: async () => fail('数据库项目不支持 Git 操作'),
      gitCommit: async () => fail('数据库项目不支持 Git 操作'),
      gitOperation: async () => fail('数据库项目不支持 Git 操作'),
    }

    return {
      api: {
        workspace,
        outlineProjects: { load: async () => [], save: async () => ok() },
        outlineSecrets: { load: async () => ok(), save: async () => ok() },
        outlineResearch: { search: async () => ok([]) },
        outlineGithub: { pagesStatus: async () => fail('GitHub Pages 不可用') },
        workBrowser: { search: async () => fail('浏览器研究不可用') },
        shell: { openExternal: async () => fail('外部链接不可用') },
        llmChat,
        generateImage: async () => fail('图片生成不可用'),
        copyText: async (text: string) => navigator.clipboard.writeText(text),
      },
      aiConfig: { apiKey: 'database-model-gateway', baseUrl: `/api/v1/projects/${project.id}/scaffolder`, model: 'project-default' },
      files: {
        openFolder: workspace.openFolder, readText: readTextFile, writeText: writeTextFile,
        readBinary: workspace.readBinaryFile, writeBinary: workspace.writeBinaryFile,
        listFiles: workspace.listFiles, listDirectory: workspace.listDirectory,
        createDirectory: workspace.createDirectory, mutate: mutateFiles, reauthorize: workspace.reauthorize,
      },
      projects: { load: async () => [], save: async () => ok() },
    } as OutlineScaffolderAdapter
  }, [project.id, project.name])

  return <div className="database-outline-scaffolder"><OutlineScaffolderPanel adapter={adapter} /></div>
}
