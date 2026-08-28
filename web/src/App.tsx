import { FormEvent, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import type { ProjectType } from './types'
import type { KnowledgeSource, Project } from './types'

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
  const [selected, setSelected] = useState<Project | null>(null)
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
            return <article className="project-card" key={project.id} onClick={() => setSelected(project)}>
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
      {selected && <Workspace project={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}

function Workspace({ project, onClose }: { project: Project; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<'documents' | 'knowledge' | 'quality'>('documents')
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [sourceName, setSourceName] = useState('')
  const [sourceContent, setSourceContent] = useState('')
  const [authority, setAuthority] = useState<KnowledgeSource['authority']>('REFERENCE')
  const [query, setQuery] = useState('')
  const [submittedQuery, setSubmittedQuery] = useState('')
  const [validationText, setValidationText] = useState('')
  const [validationQuery, setValidationQuery] = useState('')
  const documents = useQuery({ queryKey: ['documents', project.id], queryFn: () => api.documents(project.id) })
  const sources = useQuery({ queryKey: ['sources', project.id], queryFn: () => api.sources(project.id) })
  const search = useQuery({ queryKey: ['knowledge-search', project.id, submittedQuery], queryFn: () => api.searchKnowledge(project.id, submittedQuery), enabled: submittedQuery.length > 0 })
  const modelStatus = useQuery({ queryKey: ['model-status'], queryFn: api.modelStatus })
  const createDocument = useMutation({ mutationFn: () => api.createDocument(project.id, { title, content }), onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTitle(''); setContent('') } })
  const createSource = useMutation({ mutationFn: () => api.createSource(project.id, { name: sourceName, authority, content: sourceContent }), onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['sources', project.id] }); setSourceName(''); setSourceContent('') } })
  const validate = useMutation({ mutationFn: () => api.validate(project.id, { text: validationText, task: '校验文本的事实依据、一致性、完整性、术语和风格', knowledgeQuery: validationQuery, dimensions: ['groundedness', 'consistency', 'completeness', 'terminology', 'style'] }) })

  return <div className="workspace-layer">
    <div className="workspace-bar"><button onClick={onClose}>← 返回项目</button><div><small>{typeInfo[project.type].label}</small><strong>{project.name}</strong></div><span>草稿工作区</span></div>
    <div className="workspace-body">
      <aside className="workspace-nav"><button className={tab === 'documents' ? 'selected' : ''} onClick={() => setTab('documents')}>文档与版本</button><button className={tab === 'knowledge' ? 'selected' : ''} onClick={() => setTab('knowledge')}>知识库</button><button className={tab === 'quality' ? 'selected' : ''} onClick={() => setTab('quality')}>多模型校验</button><div className="pipeline"><small>准确性流水线</small><p>资料检索</p><i /><p>模型生成</p><i /><p>双模型校验</p><i /><p>质量门禁</p></div></aside>
      <section className="workspace-content">
        {tab === 'documents' ? <>
          <div className="workspace-title"><div><p className="eyebrow">DOCUMENTS</p><h2>文档与版本</h2><p>每次保存都会形成不可变版本，恢复历史也不会覆盖原始内容。</p></div></div>
          <form className="studio-form" onSubmit={(e) => { e.preventDefault(); createDocument.mutate() }}><input required value={title} onChange={(e) => setTitle(e.target.value)} placeholder="文档标题" /><textarea required rows={9} value={content} onChange={(e) => setContent(e.target.value)} placeholder="使用 Markdown 开始写作…" /><button className="primary" disabled={createDocument.isPending}>保存为初始版本</button></form>
          <div className="resource-list"><h3>项目文档 <span>{documents.data?.total ?? 0}</span></h3>{documents.data?.items.map((item) => <article key={item.id}><div><strong>{item.title}</strong><small>版本 {item.versionCount} · {new Date(item.updatedAt).toLocaleString('zh-CN')}</small></div><span>查看版本 →</span></article>)}{documents.data?.total === 0 && <p className="empty-line">暂无文档</p>}</div>
        </> : tab === 'knowledge' ? <>
          <div className="workspace-title"><div><p className="eyebrow">KNOWLEDGE BASE</p><h2>知识来源</h2><p>录入资料时标记权威等级，生成和校验将优先引用可靠来源。</p></div></div>
          <div className="knowledge-grid"><form className="studio-form" onSubmit={(e) => { e.preventDefault(); createSource.mutate() }}><h3>录入知识</h3><input required value={sourceName} onChange={(e) => setSourceName(e.target.value)} placeholder="来源名称" /><select value={authority} onChange={(e) => setAuthority(e.target.value as KnowledgeSource['authority'])}><option value="OFFICIAL">官方资料</option><option value="VERIFIED">人工确认</option><option value="INTERNAL">内部资料</option><option value="REFERENCE">普通参考</option></select><textarea required rows={7} value={sourceContent} onChange={(e) => setSourceContent(e.target.value)} placeholder="粘贴知识内容，系统将按结构分块…" /><button className="primary" disabled={createSource.isPending}>解析并加入知识库</button></form>
          <div className="search-panel"><h3>检索测试</h3><form onSubmit={(e) => { e.preventDefault(); setSubmittedQuery(query.trim()) }}><input required value={query} onChange={(e) => setQuery(e.target.value)} placeholder="输入事实、术语或问题" /><button>检索</button></form>{search.data?.items.map((hit) => <article key={hit.chunk.id}><p>{hit.chunk.content}</p><small>{hit.source.name} · {hit.source.authority} · 匹配度 {Math.round(hit.score * 100)}%</small></article>)}{submittedQuery && search.data?.total === 0 && <p className="empty-line">没有找到支持证据</p>}</div></div>
          <div className="resource-list"><h3>来源资料 <span>{sources.data?.total ?? 0}</span></h3>{sources.data?.items.map((item) => <article key={item.id}><div><strong>{item.name}</strong><small>{item.authority} · {item.status}</small></div><span>{item.version || '无版本标记'}</span></article>)}</div>
        </> : <>
          <div className="workspace-title"><div><p className="eyebrow">QUALITY GATE</p><h2>多模型文本校验</h2><p>两个 Validator 独立检查，重大分歧交由 Judge 仲裁，并以知识库证据和硬规则作为最终依据。</p></div></div>
          <div className={`model-status ${modelStatus.data?.configured ? 'ready' : 'offline'}`}><strong>{modelStatus.data?.configured ? '模型流水线已就绪' : '模型流水线未配置'}</strong><span>{modelStatus.data?.configured ? `${modelStatus.data.validatorCount} 个 Validator · Judge ${modelStatus.data.judgeConfigured ? '已配置' : '未配置'}` : '请在服务端设置 LLM_BASE_URL 和 VALIDATOR_MODELS'}</span></div>
          <form className="studio-form validation-form" onSubmit={(e) => { e.preventDefault(); validate.mutate() }}><label>待校验文本<textarea required rows={11} value={validationText} onChange={(e) => setValidationText(e.target.value)} placeholder="粘贴需要进行事实、一致性和质量校验的文本…" /></label><label>知识检索词<input value={validationQuery} onChange={(e) => setValidationQuery(e.target.value)} placeholder="例如：CreateTask project_id（可选）" /></label><button className="primary" disabled={!modelStatus.data?.configured || validate.isPending}>{validate.isPending ? '多模型校验中…' : '运行严格校验'}</button></form>
          {validate.isError && <p className="quality-error">{validate.error.message}</p>}
          {validate.data && <section className="quality-result"><div className={`score ${validate.data.gate.status.toLowerCase()}`}><strong>{validate.data.result.score}</strong><span>{validate.data.gate.status}</span></div><div className="quality-summary"><h3>质量门禁结果</h3><p>{validate.data.result.issues.length} 个问题 · {validate.data.disagreements} 个模型分歧 · {validate.data.runs.length} 次模型调用</p>{validate.data.gate.reasons.map((reason) => <small key={reason}>{reason}</small>)}</div><div className="issues">{validate.data.result.issues.map((issue) => <article key={issue.id || `${issue.type}-${issue.claim}`}><b className={issue.severity.toLowerCase()}>{issue.severity}</b><div><strong>{issue.claim || issue.type}</strong><p>{issue.explanation}</p><small>{issue.suggestedFix}</small></div></article>)}</div></section>}
        </>}
      </section>
    </div>
  </div>
}
