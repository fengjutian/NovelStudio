import { FormEvent, useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import {ArrowLeft,ArrowRight,CircleGauge,Clock3,Database,Download,Home,MoreHorizontal,Plus} from 'lucide-react'
import {Toaster,toast} from 'sonner'
import {MarkdownEditor} from './components/MarkdownEditor'
import {Button} from './components/ui/button'
import {Dialog,DialogContent,DialogDescription,DialogHeader,DialogTitle} from './components/ui/dialog'
import {Input} from './components/ui/input'
import type { ProjectType } from './types'
import type { Document, GenerationResult, KnowledgeSource, MemoryEntry, OutlineItem, Project, QualityGenerationResult } from './types'

type WorkspaceTab = 'documents' | 'structure' | 'generation' | 'knowledge' | 'memory' | 'quality' | 'runs'

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
  const [selectedTab, setSelectedTab] = useState<WorkspaceTab>('documents')
  const [activeNav, setActiveNav] = useState<'projects' | 'knowledge' | 'tasks' | 'models'>('projects')
  const [projectQuery, setProjectQuery] = useState('')
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.projects })
  const create = useMutation({
    mutationFn: api.createProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      setOpen(false)
      setName('')
      setDescription('')
      toast.success('项目创建成功')
    },
  })

  function submit(event: FormEvent) {
    event.preventDefault()
    create.mutate({ name, type, description })
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <a className="brand" href="#" onClick={(event) => { event.preventDefault(); setActiveNav('projects') }}><span>字</span><strong>Content Studio</strong></a>
        <nav>
          <a className={activeNav === 'projects' ? 'active' : ''} href="#projects" onClick={(event) => { event.preventDefault(); setActiveNav('projects') }}><Home/>项目</a>
          <a className={activeNav === 'knowledge' ? 'active' : ''} href="#knowledge" onClick={(event) => { event.preventDefault(); setActiveNav('knowledge') }}><Database/>知识库</a>
          <a className={activeNav === 'tasks' ? 'active' : ''} href="#tasks" onClick={(event) => { event.preventDefault(); setActiveNav('tasks') }}><Clock3/>任务中心</a>
          <a className={activeNav === 'models' ? 'active' : ''} href="#models" onClick={(event) => { event.preventDefault(); setActiveNav('models') }}><CircleGauge/>模型与校验</a>
        </nav>
        <div className="sidebar-foot">
          <div className="avatar">CF</div>
          <div><strong>创作者</strong><small>本地工作区</small></div>
          <button aria-label="设置"><MoreHorizontal size={16}/></button>
        </div>
      </aside>

      <main>
        {activeNav === 'projects' ? <>
        <header>
          <div><p className="eyebrow">WORKSPACE</p><h1>创作项目</h1><p>用知识与多模型协作，构建可信的长篇内容。</p></div>
          <Button className="primary" onClick={() => setOpen(true)}><Plus size={16}/> 新建项目</Button>
        </header>

        <section className="metrics">
          <article><span>项目总数</span><strong>{projects.data?.total ?? '—'}</strong><small>三个内容模板已就绪</small></article>
          <article><span>知识来源</span><strong>0</strong><small>等待导入可靠资料</small></article>
          <article><span>待处理问题</span><strong>0</strong><small className="good">质量门禁运行正常</small></article>
        </section>

        <section className="section-head"><div><h2>最近编辑</h2><p>继续你的内容生产流程</p></div><input className="project-search" value={projectQuery} onChange={(event)=>setProjectQuery(event.target.value)} placeholder="搜索项目名称、类型或描述" /></section>

        {projects.isLoading && <div className="empty">正在加载项目…</div>}
        {projects.isError && <div className="empty error">{projects.error.message}</div>}
        <section className="grid">
          {projects.data?.items.filter(project=>`${project.name} ${project.description} ${typeInfo[project.type].label}`.toLowerCase().includes(projectQuery.trim().toLowerCase())).map((project) => {
            const info = typeInfo[project.type]
            return <article className="project-card" key={project.id} onClick={() => { setSelectedTab('documents'); setSelected(project) }}>
              <div className={`project-icon ${info.accent}`}>{info.icon}</div>
              <div className="project-top"><span>{info.label}</span><button aria-label="项目菜单"><MoreHorizontal size={16}/></button></div>
              <h3>{project.name}</h3>
              <p>{project.description || '尚未添加项目描述'}</p>
              <div className="progress"><i style={{ width: project.status === 'DRAFT' ? '18%' : '60%' }} /></div>
              <footer><span>{project.status === 'DRAFT' ? '草稿' : project.status}</span><time>{new Date(project.updatedAt).toLocaleDateString('zh-CN')}</time></footer>
            </article>
          })}
          {projectQuery && projects.data?.items.filter(project=>`${project.name} ${project.description} ${typeInfo[project.type].label}`.toLowerCase().includes(projectQuery.trim().toLowerCase())).length===0&&<p className="empty-line">没有匹配的项目</p>}
          <button className="new-card" onClick={() => setOpen(true)}><Plus/><strong>开始新的创作</strong><small>小说、电影解说或技术文档</small></button>
        </section>
        </> : <GlobalPage page={activeNav} projects={projects.data?.items ?? []} onOpen={(project, tab) => { setSelectedTab(tab); setSelected(project) }} />}
      </main>

      <Dialog open={open} onOpenChange={setOpen}><DialogContent className="dialog"><form onSubmit={submit}>
          <DialogHeader className="dialog-head"><div><p className="eyebrow">NEW PROJECT</p><DialogTitle>创建内容项目</DialogTitle><DialogDescription>选择内容类型并填写项目目标。</DialogDescription></div></DialogHeader>
          <label>项目名称<Input autoFocus required maxLength={120} value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：产品 API 使用指南" /></label>
          <fieldset><legend>内容类型</legend><div className="type-grid">
            {(Object.keys(typeInfo) as ProjectType[]).map((value) => <button className={type === value ? 'selected' : ''} type="button" key={value} onClick={() => setType(value)}><b>{typeInfo[value].icon}</b><span>{typeInfo[value].label}</span></button>)}
          </div></fieldset>
          <label>创作说明<textarea rows={4} maxLength={1000} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="描述目标、受众和内容要求…" /></label>
          {create.isError && <p className="form-error">{create.error.message}</p>}
          <div className="dialog-actions"><button type="button" className="secondary" onClick={() => setOpen(false)}>取消</button><button className="primary" disabled={create.isPending}>{create.isPending ? '创建中…' : '创建项目'}</button></div>
        </form></DialogContent></Dialog>
      {selected && <Workspace project={selected} initialTab={selectedTab} onClose={() => setSelected(null)} />}
      <Toaster richColors position="top-right"/>
    </div>
  )
}

function GlobalPage({ page, projects, onOpen }: { page: 'knowledge' | 'tasks' | 'models'; projects: Project[]; onOpen: (project: Project, tab: WorkspaceTab) => void }) {
  const queryClient = useQueryClient()
  const [taskFilter,setTaskFilter]=useState('ALL')
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: api.tasks, enabled: page === 'tasks', refetchInterval: page === 'tasks' ? 2000 : false })
  const retryTask=useMutation({mutationFn:api.retryTask,onSuccess:()=>queryClient.invalidateQueries({queryKey:['tasks']})})
  const models = useQuery({ queryKey: ['model-status'], queryFn: api.modelStatus, enabled: page === 'models' })

  if (page === 'knowledge') return <>
    <header><div><p className="eyebrow">KNOWLEDGE BASE</p><h1>知识库</h1><p>按项目管理权威资料、事实来源和检索上下文。</p></div></header>
    <section className="global-panel"><div className="section-head"><div><h2>选择项目</h2><p>知识来源按项目严格隔离</p></div></div><div className="global-list">{projects.map((project) => <button key={project.id} onClick={() => onOpen(project, 'knowledge')}><span className={`project-icon ${typeInfo[project.type].accent}`}>{typeInfo[project.type].icon}</span><div><strong>{project.name}</strong><small>{typeInfo[project.type].label} · 管理知识来源与检索</small></div><b>打开知识库 <ArrowRight size={13}/></b></button>)}</div></section>
  </>

  if (page === 'tasks') return <>
    <header><div><p className="eyebrow">TASK CENTER</p><h1>任务中心</h1><p>查看后台生成、校验和处理任务的实时状态。</p></div></header>
    <section className="global-panel"><div className="section-head"><div><h2>全部任务</h2><p>{tasks.data?.total ?? 0} 条任务记录</p></div><select className="task-filter" value={taskFilter} onChange={event=>setTaskFilter(event.target.value)}><option value="ALL">全部状态</option><option value="RUNNING">运行中</option><option value="FAILED">失败</option><option value="SUCCESS">成功</option><option value="CANCELLED">已取消</option></select></div><div className="task-table">{tasks.data?.items.filter(item=>taskFilter==='ALL'||item.status===taskFilter).map((item) => <article key={item.id}><span className={`task-state ${item.status.toLowerCase()}`}>{item.status}</span><div><strong>{item.type}</strong><small>{item.error||item.message}</small><time>{new Date(item.createdAt).toLocaleString('zh-CN')}</time></div><div className="mini-progress"><i style={{ width: `${item.progress}%` }} /></div>{['FAILED','CANCELLED'].includes(item.status)?<button className="task-retry" disabled={retryTask.isPending} onClick={()=>retryTask.mutate(item.id)}>重试</button>:<b>{item.progress}%</b>}</article>)}{tasks.data?.total === 0 && <p className="empty-line">暂无后台任务。进入项目的“多模型校验”即可创建任务。</p>}</div>{retryTask.isError&&<p className="quality-error">{retryTask.error.message}</p>}</section>
  </>

  return <>
    <header><div><p className="eyebrow">MODEL ROUTING</p><h1>模型与校验</h1><p>检查服务端模型路由，并进入项目执行严格质量校验。</p></div></header>
    <section className="metrics model-metrics"><article><span>配置状态</span><strong className="text-value">{models.data?.configured ? '已就绪' : '未配置'}</strong><small>OpenAI-Compatible Provider</small></article><article><span>Validator</span><strong>{models.data?.validatorCount ?? 0}</strong><small>并行独立校验模型</small></article><article><span>Judge</span><strong className="text-value">{models.data?.judgeConfigured ? '已配置' : '未配置'}</strong><small>只处理重大模型分歧</small></article></section>
    {!models.data?.configured && <div className="setup-callout"><strong>需要配置模型环境变量</strong><code>LLM_BASE_URL · LLM_API_KEY · VALIDATOR_MODELS · JUDGE_MODEL</code><span>设置后重启 Go API。</span></div>}
    <section className="global-panel"><div className="section-head"><div><h2>项目校验</h2><p>选择项目，使用它自己的知识库作为证据</p></div></div><div className="global-list">{projects.map((project) => <button key={project.id} onClick={() => onOpen(project, 'quality')}><span className={`project-icon ${typeInfo[project.type].accent}`}>{typeInfo[project.type].icon}</span><div><strong>{project.name}</strong><small>{typeInfo[project.type].label}</small></div><b>打开校验 →</b></button>)}</div></section>
  </>
}

function Workspace({ project, initialTab, onClose }: { project: Project; initialTab: WorkspaceTab; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<WorkspaceTab>(initialTab)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [selectedDocument, setSelectedDocument] = useState<Document | null>(null)
  const [sourceName, setSourceName] = useState('')
  const [sourceContent, setSourceContent] = useState('')
  const [authority, setAuthority] = useState<KnowledgeSource['authority']>('REFERENCE')
  const [query, setQuery] = useState('')
  const [submittedQuery, setSubmittedQuery] = useState('')
  const [validationText, setValidationText] = useState('')
  const [validationQuery, setValidationQuery] = useState('')
  const [taskId, setTaskId] = useState('')
  const [generationTaskId, setGenerationTaskId] = useState('')
  const [operation, setOperation] = useState('PLAN')
  const [instruction, setInstruction] = useState('')
  const [generationTitle, setGenerationTitle] = useState('')
  const [generationDocumentId, setGenerationDocumentId] = useState('')
  const [generationQuery, setGenerationQuery] = useState('')
  const [strictGeneration, setStrictGeneration] = useState(false)
  const [maxRepairs, setMaxRepairs] = useState(2)
  const [qualityTaskId, setQualityTaskId] = useState('')
  const documents = useQuery({ queryKey: ['documents', project.id], queryFn: () => api.documents(project.id) })
  const sources = useQuery({ queryKey: ['sources', project.id], queryFn: () => api.sources(project.id) })
  const search = useQuery({ queryKey: ['knowledge-search', project.id, submittedQuery], queryFn: () => api.searchKnowledge(project.id, submittedQuery), enabled: submittedQuery.length > 0 })
  const modelStatus = useQuery({ queryKey: ['model-status'], queryFn: api.modelStatus })
  const activeTask = useQuery({ queryKey: ['task', taskId], queryFn: () => api.task(taskId), enabled: taskId.length > 0, refetchInterval: (query) => ['SUCCESS', 'FAILED', 'CANCELLED'].includes(query.state.data?.status ?? '') ? false : 1000 })
  const generationTask = useQuery({ queryKey: ['generation-task', generationTaskId], queryFn: () => api.task<GenerationResult>(generationTaskId), enabled: generationTaskId.length > 0, refetchInterval: (query) => ['SUCCESS', 'FAILED', 'CANCELLED'].includes(query.state.data?.status ?? '') ? false : 1000 })
  const qualityTask = useQuery({ queryKey: ['quality-generation-task', qualityTaskId], queryFn: () => api.task<QualityGenerationResult>(qualityTaskId), enabled: qualityTaskId.length > 0, refetchInterval: (query) => ['SUCCESS', 'FAILED', 'CANCELLED'].includes(query.state.data?.status ?? '') ? false : 1000 })
  const createDocument = useMutation({ mutationFn: () => api.createDocument(project.id, { title, content }), onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTitle(''); setContent('') } })
  const createSource = useMutation({ mutationFn: () => api.createSource(project.id, { name: sourceName, authority, content: sourceContent }), onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['sources', project.id] }); setSourceName(''); setSourceContent('') } })
  const validate = useMutation({ mutationFn: () => api.createValidationTask(project.id, { text: validationText, task: '校验文本的事实依据、一致性、完整性、术语和风格', knowledgeQuery: validationQuery, dimensions: ['groundedness', 'consistency', 'completeness', 'terminology', 'style'] }), onSuccess: (task) => setTaskId(task.id) })
  const cancelTask = useMutation({ mutationFn: () => api.cancelTask(taskId), onSuccess: () => queryClient.invalidateQueries({ queryKey: ['task', taskId] }) })
  const generate = useMutation({ mutationFn: () => api.createGenerationTask(project.id, { operation, instruction, title: generationTitle, documentId: operation === 'POLISH' ? generationDocumentId : '', knowledgeQuery: generationQuery }), onSuccess: (created) => setGenerationTaskId(created.id) })
  const qualityGenerate = useMutation({ mutationFn: () => api.qualityGenerate(project.id, { instruction, title: generationTitle, knowledgeQuery: generationQuery, maxRepairs }), onSuccess: (created) => setQualityTaskId(created.id) })
  const validationResult = activeTask.data?.result

  return <div className="workspace-layer">
    <div className="workspace-bar"><button onClick={onClose}><ArrowLeft size={14}/> 返回项目</button><div><small>{typeInfo[project.type].label}</small><strong>{project.name}</strong></div><a className="export-link" href={api.exportURL(project.id)}><Download size={14}/> 导出 Markdown</a><span>草稿工作区</span></div>
    <div className="workspace-body">
      <aside className="workspace-nav"><button className={tab === 'documents' ? 'selected' : ''} onClick={() => setTab('documents')}>文档与版本</button><button className={tab === 'structure' ? 'selected' : ''} onClick={() => setTab('structure')}>内容结构</button><button className={tab === 'generation' ? 'selected' : ''} onClick={() => setTab('generation')}>AI 创作</button><button className={tab === 'knowledge' ? 'selected' : ''} onClick={() => setTab('knowledge')}>知识库</button><button className={tab === 'memory' ? 'selected' : ''} onClick={() => setTab('memory')}>Story Memory</button><button className={tab === 'quality' ? 'selected' : ''} onClick={() => setTab('quality')}>多模型校验</button><button className={tab === 'runs' ? 'selected' : ''} onClick={() => setTab('runs')}>AI 运行记录</button><div className="pipeline"><small>准确性流水线</small><p>资料检索</p><i /><p>模型生成</p><i /><p>双模型校验</p><i /><p>质量门禁</p></div></aside>
      <section className="workspace-content">
        {tab === 'documents' ? <>
          {selectedDocument ? <DocumentEditor document={selectedDocument} onBack={() => { setSelectedDocument(null); queryClient.invalidateQueries({ queryKey: ['documents', project.id] }) }} /> : <>
            <div className="workspace-title"><div><p className="eyebrow">DOCUMENTS</p><h2>文档与版本</h2><p>每次保存都会形成不可变版本，恢复历史也不会覆盖原始内容。</p></div></div>
            <form className="studio-form" onSubmit={(e) => { e.preventDefault(); createDocument.mutate() }}><input required value={title} onChange={(e) => setTitle(e.target.value)} placeholder="文档标题" /><textarea required rows={9} value={content} onChange={(e) => setContent(e.target.value)} placeholder="使用 Markdown 开始写作…" /><button className="primary" disabled={createDocument.isPending}>保存为初始版本</button></form>
            <div className="resource-list"><h3>项目文档 <span>{documents.data?.total ?? 0}</span></h3>{documents.data?.items.map((item) => <article className="clickable" key={item.id} onClick={() => setSelectedDocument(item)}><div><strong>{item.title}</strong><small>版本 {item.versionCount} · {new Date(item.updatedAt).toLocaleString('zh-CN')}</small></div><span>打开编辑器 →</span></article>)}{documents.data?.total === 0 && <p className="empty-line">暂无文档</p>}</div>
          </>}
        </> : tab === 'structure' ? <><OutlineImporter project={project}/><StructurePanel project={project} documents={documents.data?.items ?? []} /></> : tab === 'generation' ? <>
          <div className="workspace-title"><div><p className="eyebrow">AGENT PIPELINE</p><h2>AI 内容创作</h2><p>使用 Planner、Outliner、Writer 或 Polisher，并将结果自动保存为文档版本。</p></div></div>
          <div className={`model-status ${(modelStatus.data?.generationOperations?.length ?? 0) > 0 ? 'ready' : 'offline'}`}><strong>{(modelStatus.data?.generationOperations?.length ?? 0) > 0 ? '创作模型已就绪' : '创作模型未配置'}</strong><span>{modelStatus.data?.generationOperations?.join(' · ') || '请配置 WRITER_MODEL 或角色专用模型'}</span></div>
          <form className="studio-form generation-form" onSubmit={(event) => { event.preventDefault(); if (strictGeneration) { setQualityTaskId(''); qualityGenerate.mutate() } else { setGenerationTaskId(''); generate.mutate() } }}><div className="operation-grid">{[['PLAN', '策划'], ['OUTLINE', '目录'], ['WRITE', '正文'], ['POLISH', '润色']].map(([value, label]) => <button type="button" className={operation === value ? 'selected' : ''} onClick={() => { setOperation(value); if (value !== 'WRITE') setStrictGeneration(false) }} key={value}><b>{value}</b><span>{label}</span></button>)}</div>{operation === 'POLISH' ? <label>选择待润色文档<select required value={generationDocumentId} onChange={(event) => setGenerationDocumentId(event.target.value)}><option value="">请选择文档</option>{documents.data?.items.map((item) => <option value={item.id} key={item.id}>{item.title}</option>)}</select></label> : <label>输出文档标题<input value={generationTitle} onChange={(event) => setGenerationTitle(event.target.value)} placeholder="留空则使用默认标题" /></label>}<label>创作要求<textarea required rows={7} value={instruction} onChange={(event) => setInstruction(event.target.value)} placeholder="描述内容目标、受众、风格、篇幅和必须覆盖的要点…" /></label><label>知识检索词<input value={generationQuery} onChange={(event) => setGenerationQuery(event.target.value)} placeholder="从项目知识库检索相关证据（可选）" /></label>{operation === 'WRITE' && <div className="strict-options"><label><input type="checkbox" checked={strictGeneration} onChange={(event) => setStrictGeneration(event.target.checked)} /> 严格质量生成（生成 → 多 Validator 校验 → Repair）</label>{strictGeneration && <label>最多修复次数<select value={maxRepairs} onChange={(event) => setMaxRepairs(Number(event.target.value))}><option value={1}>1 次</option><option value={2}>2 次</option><option value={3}>3 次</option></select></label>}</div>}<button className="primary" disabled={generate.isPending || qualityGenerate.isPending || generationTask.data?.status === 'RUNNING' || qualityTask.data?.status === 'RUNNING' || !(modelStatus.data?.generationOperations?.includes(operation))}>{generate.isPending || qualityGenerate.isPending ? '正在创建任务…' : strictGeneration ? '运行严格质量生成' : `运行 ${operation} Agent`}</button></form>
          {generationTask.data && !['SUCCESS', 'FAILED', 'CANCELLED'].includes(generationTask.data.status) && <div className="task-progress"><div><strong>{generationTask.data.message}</strong><span>{generationTask.data.progress}%</span></div><i><b style={{ width: `${generationTask.data.progress}%` }} /></i></div>}
          {(generate.isError || generationTask.data?.status === 'FAILED') && <p className="quality-error">{generate.error?.message || generationTask.data?.error}</p>}
          {generationTask.data?.result && <section className="generation-result"><div><p className="eyebrow">GENERATED</p><h3>生成完成并已保存</h3><small>{generationTask.data.result.generation.model} · Prompt {generationTask.data.result.generation.promptVersion} · {generationTask.data.result.generation.outputTokens} tokens</small></div><pre>{generationTask.data.result.generation.content}</pre><button className="primary" onClick={() => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTab('documents') }}>查看项目文档</button></section>}
          {qualityTask.data && !['SUCCESS', 'FAILED', 'CANCELLED'].includes(qualityTask.data.status) && <div className="task-progress"><div><strong>{qualityTask.data.message}</strong><span>{qualityTask.data.progress}%</span></div><i><b style={{ width: `${qualityTask.data.progress}%` }} /></i></div>}
          {(qualityGenerate.isError || qualityTask.data?.status === 'FAILED') && <p className="quality-error">{qualityGenerate.error?.message || qualityTask.data?.error}</p>}
          {qualityTask.data?.result && <section className="generation-result"><div><p className="eyebrow">QUALITY GENERATED</p><h3>严格质量生成完成</h3><small>门禁 {qualityTask.data.result.workflow.validation.gate.status} · 得分 {qualityTask.data.result.workflow.validation.result.score} · 校验 {qualityTask.data.result.workflow.attempts} 轮 · 修复 {qualityTask.data.result.workflow.repairs.length} 次</small></div><pre>{qualityTask.data.result.workflow.content}</pre><button className="primary" onClick={() => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTab('documents') }}>查看项目文档</button></section>}
        </> : tab === 'knowledge' ? <>
          <div className="workspace-title"><div><p className="eyebrow">KNOWLEDGE BASE</p><h2>知识来源</h2><p>录入资料时标记权威等级，生成和校验将优先引用可靠来源。</p></div></div>
          <div className="knowledge-grid"><form className="studio-form" onSubmit={(e) => { e.preventDefault(); createSource.mutate() }}><h3>录入知识</h3><input required value={sourceName} onChange={(e) => setSourceName(e.target.value)} placeholder="来源名称" /><select value={authority} onChange={(e) => setAuthority(e.target.value as KnowledgeSource['authority'])}><option value="OFFICIAL">官方资料</option><option value="VERIFIED">人工确认</option><option value="INTERNAL">内部资料</option><option value="REFERENCE">普通参考</option></select><textarea required rows={7} value={sourceContent} onChange={(e) => setSourceContent(e.target.value)} placeholder="粘贴知识内容，系统将按结构分块…" /><button className="primary" disabled={createSource.isPending}>解析并加入知识库</button></form>
          <div className="search-panel"><h3>检索测试</h3><form onSubmit={(e) => { e.preventDefault(); setSubmittedQuery(query.trim()) }}><input required value={query} onChange={(e) => setQuery(e.target.value)} placeholder="输入事实、术语或问题" /><button>检索</button></form>{search.data?.items.map((hit) => <article key={hit.chunk.id}><p>{hit.chunk.content}</p><small>{hit.source.name} · {hit.source.authority} · 匹配度 {Math.round(hit.score * 100)}%</small></article>)}{submittedQuery && search.data?.total === 0 && <p className="empty-line">没有找到支持证据</p>}</div></div>
          <div className="resource-list"><h3>来源资料 <span>{sources.data?.total ?? 0}</span></h3>{sources.data?.items.map((item) => <article key={item.id}><div><strong>{item.name}</strong><small>{item.authority} · {item.status}</small></div><span>{item.version || '无版本标记'}</span></article>)}</div>
          <KnowledgeExtras project={project} documents={documents.data?.items ?? []} />
        </> : tab === 'memory' ? <MemoryPanel project={project}/> : tab === 'runs' ? <AIRunsPanel project={project} /> : <>
          <div className="workspace-title"><div><p className="eyebrow">QUALITY GATE</p><h2>多模型文本校验</h2><p>两个 Validator 独立检查，重大分歧交由 Judge 仲裁，并以知识库证据和硬规则作为最终依据。</p></div></div>
          <div className={`model-status ${modelStatus.data?.configured ? 'ready' : 'offline'}`}><strong>{modelStatus.data?.configured ? '模型流水线已就绪' : '模型流水线未配置'}</strong><span>{modelStatus.data?.configured ? `${modelStatus.data.validatorCount} 个 Validator · Judge ${modelStatus.data.judgeConfigured ? '已配置' : '未配置'}` : '请在服务端设置 LLM_BASE_URL 和 VALIDATOR_MODELS'}</span></div>
          <form className="studio-form validation-form" onSubmit={(e) => { e.preventDefault(); setTaskId(''); validate.mutate() }}><label>待校验文本<textarea required rows={11} value={validationText} onChange={(e) => setValidationText(e.target.value)} placeholder="粘贴需要进行事实、一致性和质量校验的文本…" /></label><label>知识检索词<input value={validationQuery} onChange={(e) => setValidationQuery(e.target.value)} placeholder="例如：CreateTask project_id（可选）" /></label><button className="primary" disabled={!modelStatus.data?.configured || validate.isPending || activeTask.data?.status === 'RUNNING'}>{validate.isPending ? '正在创建任务…' : '运行严格校验'}</button></form>
          {activeTask.data && !['SUCCESS', 'FAILED', 'CANCELLED'].includes(activeTask.data.status) && <div className="task-progress"><div><strong>{activeTask.data.message}</strong><span>{activeTask.data.progress}%</span></div><i><b style={{ width: `${activeTask.data.progress}%` }} /></i><button onClick={() => cancelTask.mutate()}>取消任务</button></div>}
          {activeTask.data?.status === 'FAILED' && <p className="quality-error">{activeTask.data.error}</p>}
          {activeTask.data?.status === 'CANCELLED' && <p className="quality-error">任务已取消</p>}
          {validate.isError && <p className="quality-error">{validate.error.message}</p>}
          {validationResult && <section className="quality-result"><div className={`score ${validationResult.gate.status.toLowerCase()}`}><strong>{validationResult.result.score}</strong><span>{validationResult.gate.status}</span></div><div className="quality-summary"><h3>质量门禁结果</h3><p>{validationResult.result.issues.length} 个问题 · {validationResult.disagreements} 个模型分歧 · {validationResult.runs.length} 次模型调用</p>{validationResult.gate.reasons.map((reason) => <small key={reason}>{reason}</small>)}</div><div className="issues">{validationResult.result.issues.map((issue) => <article key={issue.id || `${issue.type}-${issue.claim}`}><b className={issue.severity.toLowerCase()}>{issue.severity}</b><div><strong>{issue.claim || issue.type}</strong><p>{issue.explanation}</p><small>{issue.suggestedFix}</small></div></article>)}</div></section>}
          <QualityHistory project={project} />
        </>}
      </section>
    </div>
  </div>
}

function OutlineImporter({project}:{project:Project}){
  const queryClient=useQueryClient();const[content,setContent]=useState('');const[preview,setPreview]=useState<OutlineItem[]>([]);const run=useMutation({mutationFn:(previewMode:boolean)=>api.importOutline(project.id,{content,preview:previewMode}),onSuccess:(data,previewMode)=>{if(previewMode)setPreview(data.items as OutlineItem[]);else{setContent('');setPreview([]);queryClient.invalidateQueries({queryKey:['tree',project.id]})}}})
  return <section className="outline-importer"><div><p className="eyebrow">OUTLINE IMPORT</p><h3>Markdown 大纲自动落树</h3><p>粘贴带 # 标题层级的大纲，系统会按项目类型转换为卷/章/节、幕/解说段或模块/章节。</p></div><textarea rows={6} value={content} onChange={event=>setContent(event.target.value)} placeholder={'# 第一卷\n## 第一章\n### 第一节'} /><div className="outline-actions"><button className="secondary" disabled={!content||run.isPending} onClick={()=>run.mutate(true)}>预览结构</button><button className="primary" disabled={!content||run.isPending} onClick={()=>run.mutate(false)}>导入内容树</button></div>{run.isError&&<p className="quality-error">{run.error.message}</p>}{preview.length>0&&<div className="outline-preview">{preview.map((item,index)=><p style={{paddingLeft:`${(item.level-1)*22}px`}} key={`${index}-${item.title}`}><b>{item.nodeType}</b><span>{item.title}</span></p>)}</div>}</section>
}

function StructurePanel({ project, documents }: { project: Project; documents: Document[] }) {
  const queryClient=useQueryClient();const[title,setTitle]=useState('');const[parentId,setParentId]=useState('');const[nodeType,setNodeType]=useState('SECTION');const[selected,setSelected]=useState<string[]>([]);const[instruction,setInstruction]=useState('');const[taskId,setTaskId]=useState('')
  const tree=useQuery({queryKey:['tree',project.id],queryFn:()=>api.tree(project.id)});const refresh=()=>queryClient.invalidateQueries({queryKey:['tree',project.id]});const create=useMutation({mutationFn:()=>api.createNode(project.id,{parentId:parentId||undefined,nodeType,title,position:(tree.data?.items.length??0)+1}),onSuccess:()=>{setTitle('');refresh()}});const update=useMutation({mutationFn:({id,title,position,documentId}:{id:string;title:string;position:number;documentId?:string})=>api.updateNode(id,{title,position,documentId}),onSuccess:refresh});const remove=useMutation({mutationFn:api.deleteNode,onSuccess:refresh});const batch=useMutation({mutationFn:()=>api.batchGenerate(project.id,{nodeIds:selected,instruction,knowledgeQuery:'',windowSize:2}),onSuccess:(task)=>setTaskId(task.id)});const active=useQuery({queryKey:['batch-task',taskId],queryFn:()=>api.task(taskId),enabled:Boolean(taskId),refetchInterval:(q)=>['SUCCESS','FAILED','CANCELLED'].includes(q.state.data?.status??'')?false:1000});useEffect(()=>{if(active.data?.status==='SUCCESS'){refresh();queryClient.invalidateQueries({queryKey:['documents',project.id]})}},[active.data?.status])
  const items=tree.data?.items??[];const children=new Map<string,typeof items>();items.forEach(node=>{const key=node.parentId??'';children.set(key,[...(children.get(key)??[]),node])});const rows:Array<{node:typeof items[number];depth:number}>=[];const walk=(parent:string,depth:number)=>{(children.get(parent)??[]).sort((a,b)=>a.position-b.position).forEach(node=>{rows.push({node,depth});walk(node.id,depth+1)})};walk('',0);items.filter(node=>!rows.some(row=>row.node.id===node.id)).forEach(node=>rows.push({node,depth:0}))
  return <><div className="workspace-title"><div><p className="eyebrow">CONTENT TREE</p><h2>内容结构</h2><p>按层级管理章节节点，绑定文档，并选择多个节点批量生成正文。</p></div></div><form className="inline-form node-create" onSubmit={(e)=>{e.preventDefault();create.mutate()}}><select value={nodeType} onChange={(e)=>setNodeType(e.target.value)}><option value="VOLUME">卷</option><option value="CHAPTER">章</option><option value="SECTION">节</option></select><select value={parentId} onChange={(e)=>setParentId(e.target.value)}><option value="">顶层节点</option>{items.map(node=><option value={node.id} key={node.id}>{node.title}</option>)}</select><input required value={title} onChange={(e)=>setTitle(e.target.value)} placeholder="新节点标题"/><button className="primary">新增节点</button></form><div className="tree-list">{rows.map(({node,depth})=><article key={node.id} style={{paddingLeft:`${12+depth*28}px`}}><input type="checkbox" checked={selected.includes(node.id)} onChange={(e)=>setSelected(e.target.checked?[...selected,node.id]:selected.filter(id=>id!==node.id))}/><div><strong>{depth>0?'└ ':''}{node.title}</strong><small>{node.nodeType} · 层级 {depth+1} · 位置 {node.position}</small></div><select aria-label="绑定文档" value={node.documentId??''} onChange={(e)=>update.mutate({id:node.id,title:node.title,position:node.position,documentId:e.target.value||undefined})}><option value="">未绑定文档</option>{documents.map(doc=><option value={doc.id} key={doc.id}>{doc.title}</option>)}</select><button onClick={()=>remove.mutate(node.id)}>删除</button></article>)}</div><form className="studio-form batch-form" onSubmit={(e)=>{e.preventDefault();batch.mutate()}}><h3>批量生成选中节点</h3><textarea required rows={4} value={instruction} onChange={(e)=>setInstruction(e.target.value)} placeholder="统一的创作要求…"/><button className="primary" disabled={!selected.length||batch.isPending}>生成 {selected.length} 个节点</button></form>{active.data&&<div className="task-progress"><div><strong>{active.data.message}</strong><span>{active.data.progress}%</span></div><i><b style={{width:`${active.data.progress}%`}}/></i></div>}</>
}

function MemoryPanel({project}:{project:Project}){const queryClient=useQueryClient();const[type,setType]=useState<MemoryEntry['type']>('CHARACTER');const[filter,setFilter]=useState('');const[name,setName]=useState('');const[summary,setSummary]=useState('');const memories=useQuery({queryKey:['memories',project.id,filter],queryFn:()=>api.memories(project.id,filter)});const create=useMutation({mutationFn:()=>api.createMemory(project.id,{type,name,summary,status:'ACTIVE'}),onSuccess:()=>{setName('');setSummary('');queryClient.invalidateQueries({queryKey:['memories',project.id]})}});const remove=useMutation({mutationFn:api.deleteMemory,onSuccess:()=>queryClient.invalidateQueries({queryKey:['memories',project.id]})});const labels:Record<string,string>={CHARACTER:'人物 / 主体',PLACE:'地点 / 场景',TIMELINE:'时间线',PLOT:'剧情 / 主题线',FORESHADOW:'伏笔 / 待办'};return <><div className="workspace-title"><div><p className="eyebrow">STORY MEMORY</p><h2>长期记忆</h2><p>维护人物、地点、时间、剧情和伏笔，作为后续生成的连续性上下文。</p></div></div><div className="memory-layout"><form className="studio-form" onSubmit={event=>{event.preventDefault();create.mutate()}}><h3>新增记忆</h3><select value={type} onChange={event=>setType(event.target.value as MemoryEntry['type'])}>{Object.entries(labels).map(([value,label])=><option value={value} key={value}>{label}</option>)}</select><input required value={name} onChange={event=>setName(event.target.value)} placeholder="名称或事件标题"/><textarea rows={6} value={summary} onChange={event=>setSummary(event.target.value)} placeholder="描述状态、关系、发生时间、约束或需要兑现的内容…"/><button className="primary" disabled={create.isPending}>保存记忆</button></form><section><div className="memory-filter"><button className={!filter?'selected':''} onClick={()=>setFilter('')}>全部</button>{Object.entries(labels).map(([value,label])=><button className={filter===value?'selected':''} onClick={()=>setFilter(value)} key={value}>{label}</button>)}</div><div className="memory-list">{memories.data?.items.map(item=><article key={item.id}><b>{labels[item.type]}</b><div><strong>{item.name}</strong><p>{item.summary||'暂无详细描述'}</p><small>{item.status} · {new Date(item.updatedAt).toLocaleString('zh-CN')}</small></div><button onClick={()=>remove.mutate(item.id)}>删除</button></article>)}{memories.data?.total===0&&<p className="empty-line">暂无此类记忆。</p>}</div></section></div></>}

function AIRunsPanel({project}:{project:Project}){const runs=useQuery({queryKey:['ai-runs',project.id],queryFn:()=>api.aiRuns(project.id),refetchInterval:5000});return <><div className="workspace-title"><div><p className="eyebrow">OBSERVABILITY</p><h2>AI 运行记录</h2><p>查看每次模型调用的角色、模型、Token、耗时和状态。</p></div></div><section className="metrics run-metrics"><article><span>调用次数</span><strong>{runs.data?.total??0}</strong><small>最近 100 条</small></article><article><span>Token 总量</span><strong>{(runs.data?.stats.inputTokens??0)+(runs.data?.stats.outputTokens??0)}</strong><small>输入 {runs.data?.stats.inputTokens??0} · 输出 {runs.data?.stats.outputTokens??0}</small></article><article><span>累计耗时</span><strong>{Math.round((runs.data?.stats.latencyMs??0)/1000)}s</strong><small>模型请求耗时</small></article></section><div className="run-table">{runs.data?.items.map(run=><article key={run.id}><span className={`task-state ${run.status.toLowerCase()}`}>{run.status}</span><div><strong>{run.role} · {run.model}</strong><small>{run.provider} · Prompt {run.promptVersion||'—'} · {new Date(run.createdAt).toLocaleString('zh-CN')}</small></div><span>{run.inputTokens} / {run.outputTokens} tokens</span><b>{run.latencyMs} ms</b></article>)}{runs.data?.total===0&&<p className="empty-line">暂无模型调用记录。</p>}</div></>}

function QualityHistory({project}:{project:Project}){const history=useQuery({queryKey:['quality-history',project.id],queryFn:()=>api.qualityResults(project.id),refetchInterval:5000});return <section className="quality-history"><h3>历史质量结果 <span>{history.data?.total??0}</span></h3>{history.data?.items.map(item=><article key={item.id}><b className={`gate-badge ${item.gateStatus.toLowerCase()}`}>{item.gateStatus}</b><div><strong>{item.score} 分 · {item.verdict}</strong><small>{item.documentId?'已关联文档':'独立文本'} · {new Date(item.createdAt).toLocaleString('zh-CN')}</small></div><span>{item.result.result.issues.length} 个问题</span></article>)}{history.data?.total===0&&<p className="empty-line">暂无持久化校验结果。</p>}</section>}

function KnowledgeExtras({project,documents}:{project:Project;documents:Document[]}){
  const queryClient=useQueryClient();const[file,setFile]=useState<File|null>(null);const[documentId,setDocumentId]=useState('');const facts=useQuery({queryKey:['facts',project.id],queryFn:()=>api.facts(project.id)});const upload=useMutation({mutationFn:()=>api.uploadKnowledge(project.id,file!,'REFERENCE'),onSuccess:()=>queryClient.invalidateQueries({queryKey:['sources',project.id]})});const extract=useMutation({mutationFn:()=>api.extractFacts(project.id,documentId)});const update=useMutation({mutationFn:({id,status}:{id:string;status:string})=>api.updateFact(id,status),onSuccess:()=>queryClient.invalidateQueries({queryKey:['facts',project.id]})})
  return <section className="knowledge-extras"><form className="inline-form" onSubmit={(e)=>{e.preventDefault();upload.mutate()}}><input required type="file" accept=".txt,.md,.markdown,.json,.csv,.html,.htm" onChange={(e)=>setFile(e.target.files?.[0]??null)}/><button className="primary" disabled={!file||upload.isPending}>上传知识文件</button></form><form className="inline-form" onSubmit={(e)=>{e.preventDefault();extract.mutate()}}><select required value={documentId} onChange={(e)=>setDocumentId(e.target.value)}><option value="">选择文档抽取事实</option>{documents.map(doc=><option value={doc.id} key={doc.id}>{doc.title}</option>)}</select><button className="primary" disabled={!documentId||extract.isPending}>抽取事实</button></form><div className="fact-list"><h3>结构化事实 <span>{facts.data?.total??0}</span></h3>{facts.data?.items.map(fact=><article key={fact.id}><div><strong>{fact.subject} · {fact.predicate} · {fact.object}</strong><small>{fact.status} · 置信度 {Math.round(fact.confidence*100)}%</small></div>{fact.status==='PROPOSED'&&<span><button onClick={()=>update.mutate({id:fact.id,status:'CONFIRMED'})}>确认</button><button onClick={()=>update.mutate({id:fact.id,status:'REJECTED'})}>拒绝</button></span>}</article>)}</div></section>
}

function DocumentEditor({ document, onBack }: { document: Document; onBack: () => void }) {
  const queryClient = useQueryClient()
  const [content, setContent] = useState('')
  const [baseVersionId, setBaseVersionId] = useState('')
  const [savedContent, setSavedContent] = useState('')
  const [autoSave, setAutoSave] = useState(false)
  const [compareFrom, setCompareFrom] = useState('')
  const [selection,setSelection]=useState({start:0,end:0})
  const copilotRange=useRef({start:0,end:0})
  const [copilotTaskId,setCopilotTaskId]=useState('')
  const [appliedTaskId,setAppliedTaskId]=useState('')
  const versions = useQuery({ queryKey: ['versions', document.id], queryFn: () => api.versions(document.id) })
  const diff = useQuery({queryKey:['document-diff',document.id,compareFrom,versions.data?.items[0]?.id],queryFn:()=>api.diffVersions(document.id,compareFrom,versions.data!.items[0].id),enabled:Boolean(compareFrom&&versions.data?.items[0]?.id&&compareFrom!==versions.data?.items[0]?.id)})
  const copilotTask=useQuery({queryKey:['copilot-task',copilotTaskId],queryFn:()=>api.task<GenerationResult>(copilotTaskId),enabled:Boolean(copilotTaskId),refetchInterval:query=>['SUCCESS','FAILED','CANCELLED'].includes(query.state.data?.status??'')?false:700})
  const copilot=useMutation({mutationFn:(action:string)=>{copilotRange.current=selection.end>selection.start?selection:{start:0,end:content.length};const selected=content.slice(copilotRange.current.start,copilotRange.current.end);return api.createGenerationTask(document.projectId,{operation:'POLISH',instruction:`执行文本${action}。只输出处理后的文本，不要解释，不要添加代码围栏。`,title:'',documentId:'',knowledgeQuery:'',content:selected,save:false})},onSuccess:task=>setCopilotTaskId(task.id)})
  const save = useMutation({
    mutationFn: () => api.saveVersion(document.id, { content, reason: autoSave ? 'AUTO_SAVE' : 'HUMAN_EDIT', expectedVersionId: baseVersionId }),
    onSuccess: (version) => {
      setBaseVersionId(version.id)
      setSavedContent(content)
      queryClient.invalidateQueries({ queryKey: ['versions', document.id] })
    },
  })
  const restore = useMutation({
    mutationFn: (versionId: string) => api.restoreVersion(document.id, versionId),
    onSuccess: (version) => {
      setContent(version.content)
      setSavedContent(version.content)
      setBaseVersionId(version.id)
      queryClient.invalidateQueries({ queryKey: ['versions', document.id] })
    },
  })
  const latest = versions.data?.items[0]
  const dirty = Boolean(baseVersionId && content !== savedContent)

  useEffect(() => {
    if (latest && !baseVersionId) {
      setContent(latest.content)
      setSavedContent(latest.content)
      setBaseVersionId(latest.id)
    }
  }, [latest, baseVersionId])

  useEffect(() => {
    if (!autoSave || !dirty || save.isPending) return
    const timer = window.setTimeout(() => save.mutate(), 3000)
    return () => window.clearTimeout(timer)
  }, [autoSave, content, dirty, baseVersionId, save])

  useEffect(()=>{const generated=copilotTask.data?.result?.generation.content;if(!generated||copilotTask.data?.status!=='SUCCESS'||appliedTaskId===copilotTaskId)return;const range=copilotRange.current;setContent(value=>value.slice(0,range.start)+generated+value.slice(range.end));setAppliedTaskId(copilotTaskId)},[copilotTask.data?.status,copilotTask.data?.result,copilotTaskId,appliedTaskId])

  function reloadLatest() {
    if (!latest) return
    setContent(latest.content)
    setSavedContent(latest.content)
    setBaseVersionId(latest.id)
    save.reset()
  }

  return <section className="document-editor">
    <div className="editor-head"><button onClick={onBack}><ArrowLeft size={14}/> 文档列表</button><div><p className="eyebrow">MARKDOWN DOCUMENT</p><h2>{document.title}</h2></div><label><input type="checkbox" checked={autoSave} onChange={(event) => setAutoSave(event.target.checked)} /> 3 秒自动保存</label><button className="primary" disabled={!dirty || save.isPending} onClick={() => save.mutate()}>{save.isPending ? '保存中…' : dirty ? '保存新版本' : '已保存'}</button></div>
    {save.isError && <div className="conflict-banner"><span>{save.error.message}</span><button onClick={reloadLatest}>载入服务器最新版本</button></div>}
    <div className="copilot-bar"><strong>AI Copilot</strong><span>{selection.end>selection.start?`已选择 ${selection.end-selection.start} 字符`:'未选择时处理全文'}</span>{['润色','扩写','缩写','改写','续写','总结','分析','检查'].map(action=><button disabled={copilot.isPending||copilotTask.data?.status==='RUNNING'} onClick={()=>copilot.mutate(action)} key={action}>{action}</button>)}</div>
    {(copilot.isPending||copilotTask.data?.status==='RUNNING')&&<div className="copilot-status">AI 正在处理选中文本…</div>}{(copilot.isError||copilotTask.data?.status==='FAILED')&&<p className="quality-error">{copilot.error?.message||copilotTask.data?.error}</p>}
    <div className="editor-layout"><div className="markdown-pane"><MarkdownEditor value={content} onChange={setContent} onSelectionChange={(start,end)=>setSelection({start,end})}/><footer><span>{content.length} 字符</span><span>{dirty ? '有未保存修改' : `当前版本 v${latest?.versionNumber ?? '—'}`}</span></footer></div><aside className="version-panel"><h3>版本历史 <span>{versions.data?.total ?? 0}</span></h3>{versions.data?.items.map((version) => <article className={version.id === baseVersionId ? 'current' : ''} key={version.id}><button onClick={() => { setContent(version.content); setBaseVersionId(latest?.id ?? version.id) }}><strong>v{version.versionNumber}</strong><span>{version.reason}</span><small>{new Date(version.createdAt).toLocaleString('zh-CN')}</small></button>{version.id !== latest?.id && <div className="version-actions"><button className="restore" onClick={() => setCompareFrom(version.id)}>与最新版比较</button><button className="restore" disabled={restore.isPending} onClick={() => restore.mutate(version.id)}>恢复</button></div>}</article>)}</aside></div>
    {diff.data&&<section className="diff-panel"><header><div><h3>版本差异</h3><small>新增 {diff.data.added} 行 · 删除 {diff.data.deleted} 行</small></div><button onClick={()=>setCompareFrom('')}>关闭</button></header><pre>{diff.data.lines.map((line,index)=><span className={line.type.toLowerCase()} key={`${index}-${line.oldLine}-${line.newLine}`}><i>{line.oldLine??' '}</i><i>{line.newLine??' '}</i><b>{line.type==='ADDED'?'+':line.type==='DELETED'?'-':' '}</b>{line.content||' '}</span>)}</pre></section>}
  </section>
}
