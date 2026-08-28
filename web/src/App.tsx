import { FormEvent, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import type { ProjectType } from './types'

const typeInfo: Record<ProjectType, { label: string; icon: string; accent: string }> = {
  NOVEL: { label: '小说', icon: '文', accent: 'amber' },
  MOVIE_COMMENTARY: { label: '电影解说', icon: '映', accent: 'blue' },
  TECHNICAL_DOCUMENT: { label: '技术文档', icon: '术', accent: 'green' },
}

export function App() {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [type, setType] = useState<ProjectType>('NOVEL')
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.projects })
  const create = useMutation({
    mutationFn: api.createProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      setOpen(false)
      setName('')
      setDescription('')
    },
  })

  function submit(event: FormEvent) {
    event.preventDefault()
    create.mutate({ name, type, description })
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <a className="brand" href="#"><span>字</span><strong>Content Studio</strong></a>
        <nav>
          <a className="active" href="#"><span>⌂</span>项目</a>
          <a href="#knowledge"><span>◇</span>知识库</a>
          <a href="#tasks"><span>◷</span>任务中心</a>
          <a href="#models"><span>◎</span>模型与校验</a>
        </nav>
        <div className="sidebar-foot">
          <div className="avatar">CF</div>
          <div><strong>创作者</strong><small>本地工作区</small></div>
          <button aria-label="设置">•••</button>
        </div>
      </aside>

      <main>
        <header>
          <div><p className="eyebrow">WORKSPACE</p><h1>创作项目</h1><p>用知识与多模型协作，构建可信的长篇内容。</p></div>
          <button className="primary" onClick={() => setOpen(true)}>＋ 新建项目</button>
        </header>

        <section className="metrics">
          <article><span>项目总数</span><strong>{projects.data?.total ?? '—'}</strong><small>三个内容模板已就绪</small></article>
          <article><span>知识来源</span><strong>0</strong><small>等待导入可靠资料</small></article>
          <article><span>待处理问题</span><strong>0</strong><small className="good">质量门禁运行正常</small></article>
        </section>

        <section className="section-head"><div><h2>最近编辑</h2><p>继续你的内容生产流程</p></div><button className="quiet">全部项目 →</button></section>

        {projects.isLoading && <div className="empty">正在加载项目…</div>}
        {projects.isError && <div className="empty error">{projects.error.message}</div>}
        <section className="grid">
          {projects.data?.items.map((project) => {
            const info = typeInfo[project.type]
            return <article className="project-card" key={project.id}>
              <div className={`project-icon ${info.accent}`}>{info.icon}</div>
              <div className="project-top"><span>{info.label}</span><button aria-label="项目菜单">•••</button></div>
              <h3>{project.name}</h3>
              <p>{project.description || '尚未添加项目描述'}</p>
              <div className="progress"><i style={{ width: project.status === 'DRAFT' ? '18%' : '60%' }} /></div>
              <footer><span>{project.status === 'DRAFT' ? '草稿' : project.status}</span><time>{new Date(project.updatedAt).toLocaleDateString('zh-CN')}</time></footer>
            </article>
          })}
          <button className="new-card" onClick={() => setOpen(true)}><span>＋</span><strong>开始新的创作</strong><small>小说、电影解说或技术文档</small></button>
        </section>
      </main>

      {open && <div className="overlay" onMouseDown={(e) => e.target === e.currentTarget && setOpen(false)}>
        <form className="dialog" onSubmit={submit}>
          <div className="dialog-head"><div><p className="eyebrow">NEW PROJECT</p><h2>创建内容项目</h2></div><button type="button" onClick={() => setOpen(false)}>×</button></div>
          <label>项目名称<input autoFocus required maxLength={120} value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：产品 API 使用指南" /></label>
          <fieldset><legend>内容类型</legend><div className="type-grid">
            {(Object.keys(typeInfo) as ProjectType[]).map((value) => <button className={type === value ? 'selected' : ''} type="button" key={value} onClick={() => setType(value)}><b>{typeInfo[value].icon}</b><span>{typeInfo[value].label}</span></button>)}
          </div></fieldset>
          <label>创作说明<textarea rows={4} maxLength={1000} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="描述目标、受众和内容要求…" /></label>
          {create.isError && <p className="form-error">{create.error.message}</p>}
          <div className="dialog-actions"><button type="button" className="secondary" onClick={() => setOpen(false)}>取消</button><button className="primary" disabled={create.isPending}>{create.isPending ? '创建中…' : '创建项目'}</button></div>
        </form>
      </div>}
    </div>
  )
}
