import { FormEvent, useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import type {AuthUser} from './api'
import {ArrowLeft,ArrowRight,Bell,Check,ChevronDown,ChevronLeft,ChevronRight,CircleGauge,Clock3,Database,Download,FileImage,FileText,Home,MoreHorizontal,Pencil,Plus,Search,Settings2,Shapes,Sparkles,Trash2,Upload} from 'lucide-react'
import {Toaster,toast} from 'sonner'
import {MarkdownEditor} from './components/MarkdownEditor'
import {MarkdownPreview} from './components/MarkdownPreview'
import {Button} from './components/ui/button'
import {Dialog,DialogContent,DialogDescription,DialogHeader,DialogTitle} from './components/ui/dialog'
import {Input} from './components/ui/input'
import {Select} from './components/ui/select'
import type { ProjectType } from './types'
import type { ContentNode, ContentType, Document, GenerationResult, KnowledgeFile, KnowledgeSource, MemoryEntry, MemoryExtractionResult, MemorySuggestion, OutlineItem, Project, QualityGenerationResult } from './types'

type WorkspaceTab = 'documents' | 'structure' | 'generation' | 'knowledge' | 'memory' | 'quality' | 'runs' | 'storyline'

const fallbackTypes:Record<string,{name:string;icon:string;accent:string}>={NOVEL:{name:'小说',icon:'文',accent:'amber'},MOVIE_COMMENTARY:{name:'电影解说',icon:'映',accent:'blue'},TECHNICAL_DOCUMENT:{name:'技术文档',icon:'术',accent:'green'}}
const typeDisplay=(code:string,items:ContentType[]=[])=>items.find(item=>item.code===code)??fallbackTypes[code]??{name:code,icon:code.slice(0,1),accent:'amber'}
const normalizeGeneratedOutline=(content:string)=>{
  const cleaned=content.replace(/<think\b[^>]*>[\s\S]*?<\/think\s*>/gi,'').replace(/<\/?think\b[^>]*>/gi,'').replace(/^```(?:markdown)?\s*|\s*```$/gim,'').trim()
  const headings=cleaned.split(/\r?\n/).map(line=>line.trim()).filter(line=>/^#{1,2}\s+\S/.test(line)&&!/^(?:#{1,2})\s*(?:章节目标|核心要点|所需证据|预计篇幅|篇幅统计)\s*$/i.test(line))
  return headings.length?headings.join('\n'):cleaned
}

type ChapterGeneratorDraft={requirement:string;outline:string;knowledgeQuery:string;splitMode:'LEAF'|'ALL';nodeIds:string[];documentsReady:boolean;selectionVersion:number}
const chapterDraftKey=(projectId:string)=>`novelstudio:chapter-generator:${projectId}`
const loadChapterDraft=(project:Project):ChapterGeneratorDraft=>{
  const fallback:ChapterGeneratorDraft={requirement:'',outline:'',knowledgeQuery:'',splitMode:'LEAF',nodeIds:[],documentsReady:false,selectionVersion:3}
  try{
    const saved=localStorage.getItem(chapterDraftKey(project.id))
    if(!saved)return fallback
    const parsed=JSON.parse(saved)
    const current=parsed.selectionVersion===3
    return {...fallback,...parsed,outline:normalizeGeneratedOutline(String(parsed.outline??'')),nodeIds:current&&Array.isArray(parsed.nodeIds)?parsed.nodeIds:[],documentsReady:current&&parsed.documentsReady===true,selectionVersion:3}
  }catch{return fallback}
}

function AuthScreen({onAuthenticated}:{onAuthenticated:(user:AuthUser)=>void}){
  const [mode,setMode]=useState<'login'|'register'>('login')
  const [name,setName]=useState('')
  const [email,setEmail]=useState('')
  const [password,setPassword]=useState('')
  const authMutation=useMutation({
    mutationFn:()=>mode==='login'?api.login({email,password}):api.register({name,email,password}),
    onSuccess:({user})=>onAuthenticated(user),
  })
  const switchMode=(next:'login'|'register')=>{setMode(next);authMutation.reset()}
  return <main className="auth-page">
    <section className="auth-hero"><div className="auth-mark">字</div><p>CONTENT STUDIO</p><h1>让每一个长篇构想，<br/>都有清晰的创作路径。</h1><span>知识、结构与 AI 协作的一体化内容工作台。</span></section>
    <section className="auth-panel"><form className="auth-card" onSubmit={(event)=>{event.preventDefault();authMutation.mutate()}}>
      <header><p>欢迎使用</p><h2>{mode==='login'?'登录你的工作区':'创建创作账号'}</h2><span>{mode==='login'?'继续管理你的项目与灵感。':'只需一分钟，即可开始创作。'}</span></header>
      <div className="auth-tabs"><button type="button" className={mode==='login'?'active':''} onClick={()=>switchMode('login')}>登录</button><button type="button" className={mode==='register'?'active':''} onClick={()=>switchMode('register')}>注册</button></div>
      {mode==='register'&&<label>昵称<Input autoFocus required minLength={2} maxLength={80} value={name} onChange={e=>setName(e.target.value)} placeholder="你的创作昵称"/></label>}
      <label>邮箱<Input autoFocus={mode==='login'} required type="email" autoComplete="email" value={email} onChange={e=>setEmail(e.target.value)} placeholder="name@example.com"/></label>
      <label>密码<Input required type="password" minLength={8} maxLength={128} autoComplete={mode==='login'?'current-password':'new-password'} value={password} onChange={e=>setPassword(e.target.value)} placeholder="至少 8 位字符"/></label>
      {authMutation.isError&&<p className="auth-error">{authMutation.error.message}</p>}
      <Button className="primary auth-submit" disabled={authMutation.isPending}>{authMutation.isPending?'请稍候…':mode==='login'?'登录':'注册并进入'}</Button>
      <small>登录即表示你同意妥善保管自己的账号信息。</small>
    </form></section>
  </main>
}

export function App() {
  const queryClient = useQueryClient()
  const session=useQuery({queryKey:['session'],queryFn:api.me,retry:false,staleTime:60_000})
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [editingProject, setEditingProject] = useState<Project | null>(null)
  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [type, setType] = useState<ProjectType>('NOVEL')
  const [selected, setSelected] = useState<Project | null>(null)
  const [selectedTab, setSelectedTab] = useState<WorkspaceTab>('documents')
  const [activeNav, setActiveNav] = useState<'projects' | 'knowledge' | 'types' | 'tasks' | 'models'>('projects')
  const [projectQuery, setProjectQuery] = useState('')
  const signedIn=Boolean(session.data?.user)
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.projects,enabled:signedIn })
  const contentTypes=useQuery({queryKey:['content-types'],queryFn:api.contentTypes,enabled:signedIn})
  useEffect(()=>{if(contentTypes.data?.items.length&&!contentTypes.data.items.some(item=>item.code===type))setType(contentTypes.data.items[0].code)},[contentTypes.data,type])
  const dashboard=useQuery({queryKey:['dashboard-stats'],queryFn:api.dashboardStats,refetchInterval:10000,enabled:signedIn})
  const logout=useMutation({mutationFn:api.logout,onSuccess:()=>queryClient.clear()})
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
  const updateProject = useMutation({
    mutationFn: () => api.updateProject(editingProject!.id, { name: editName, description: editDescription }),
    onSuccess: (updated) => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      setSelected(current => current?.id === updated.id ? updated : current)
      setEditingProject(null)
      toast.success('项目已更新')
    },
    onError: (error) => toast.error('保存失败', { description: error.message }),
  })

  function submit(event: FormEvent) {
    event.preventDefault()
    create.mutate({ name, type, description })
  }

  function startEditing(project: Project) {
    setEditingProject(project)
    setEditName(project.name)
    setEditDescription(project.description)
    updateProject.reset()
  }

  function submitEdit(event: FormEvent) {
    event.preventDefault()
    updateProject.mutate()
  }

  const navLabels={projects:'项目',knowledge:'知识库',types:'内容类型',tasks:'任务中心',models:'模型与校验'} as const
  if(session.isLoading)return <div className="auth-loading">正在加载工作区…</div>
  if(!session.data?.user)return <AuthScreen onAuthenticated={(user)=>queryClient.setQueryData(['session'],{user})}/>
  const currentUser=session.data.user

  return (
    <div className="shell">
      <aside className="sidebar">
        <a className="brand" href="#" onClick={(event) => { event.preventDefault(); setActiveNav('projects') }}><span>字</span><strong>Content Studio</strong></a>
        <nav>
          <a className={activeNav === 'projects' ? 'active' : ''} href="#projects" onClick={(event) => { event.preventDefault(); setActiveNav('projects') }}><Home/>项目</a>
          <a className={activeNav === 'knowledge' ? 'active' : ''} href="#knowledge" onClick={(event) => { event.preventDefault(); setActiveNav('knowledge') }}><Database/>知识库</a>
          <a className={activeNav === 'types' ? 'active' : ''} href="#types" onClick={(event) => { event.preventDefault(); setActiveNav('types') }}><Shapes/>内容类型</a>
          <a className={activeNav === 'tasks' ? 'active' : ''} href="#tasks" onClick={(event) => { event.preventDefault(); setActiveNav('tasks') }}><Clock3/>任务中心</a>
          <a className={activeNav === 'models' ? 'active' : ''} href="#models" onClick={(event) => { event.preventDefault(); setActiveNav('models') }}><CircleGauge/>模型与校验</a>
        </nav>
        <div className="sidebar-foot">
          <div className="avatar">{currentUser.name.slice(0,2).toUpperCase()}</div>
          <div><strong>{currentUser.name}</strong><small>{currentUser.email}</small></div>
          <button aria-label="退出登录" title="退出登录" onClick={()=>logout.mutate()}><MoreHorizontal size={16}/></button>
        </div>
      </aside>

      <main>
        <div className="global-status-bar" role="status">
          <div className="status-location"><span>Content Studio</span><i>/</i><strong>{navLabels[activeNav]}</strong></div>
          <div className="status-actions">
            <span className={`service-status ${dashboard.isError?'offline':''}`}><i/>{dashboard.isError?'服务连接异常':'服务运行正常'}</span>
            <button className="status-notifications" type="button" aria-label="通知"><Bell size={16}/></button>
            <button className="status-user" type="button" aria-label="用户菜单">
              <span className="status-avatar">{currentUser.name.slice(0,2).toUpperCase()}</span>
              <span><strong>{currentUser.name}</strong><small>已登录</small></span>
              <ChevronDown size={14}/>
            </button>
          </div>
        </div>
        <div className="main-scroll-content">
        {activeNav === 'projects' ? <>
        <header>
          <div><p className="eyebrow">WORKSPACE</p><h1>创作项目</h1><p>用知识与多模型协作，构建可信的长篇内容。</p></div>
          <Button className="primary" onClick={() => setOpen(true)}><Plus size={16}/> 新建项目</Button>
        </header>

        <section className="metrics">
          <article><span>项目与文档</span><strong>{dashboard.data?.projects ?? '—'} <small>/ {dashboard.data?.documents??0} 篇</small></strong><small>项目数量与文档总量</small></article>
          <article><span>知识来源</span><strong>{dashboard.data?.knowledgeSources??'—'}</strong><small>{dashboard.data?.runningTasks??0} 个任务正在运行</small></article>
          <article><span>待处理问题</span><strong>{dashboard.data?.pendingIssues??'—'}</strong><small className={(dashboard.data?.pendingIssues??0)===0?'good':''}>平均质量分 {dashboard.data?.averageQualityScore??0}</small></article>
        </section>

        <section className="section-head"><div><h2>最近编辑</h2><p>继续你的内容生产流程</p></div><input className="project-search" value={projectQuery} onChange={(event)=>setProjectQuery(event.target.value)} placeholder="搜索项目名称、类型或描述" /></section>

        {projects.isLoading && <div className="empty">正在加载项目…</div>}
        {projects.isError && <div className="empty error">{projects.error.message}</div>}
        <section className="grid">
          {projects.data?.items.filter(project=>`${project.name} ${project.description} ${typeDisplay(project.type,contentTypes.data?.items).name}`.toLowerCase().includes(projectQuery.trim().toLowerCase())).map((project) => {
            const info = typeDisplay(project.type,contentTypes.data?.items)
            return <article className="project-card" key={project.id} onClick={() => { setSelectedTab('documents'); setSelected(project) }}>
              <div className={`project-icon ${info.accent}`}>{info.icon}</div>
              <div className="project-top"><span>{info.name}</span><button type="button" aria-label="编辑项目" title="编辑项目" onClick={(event)=>{event.stopPropagation();startEditing(project)}}><MoreHorizontal size={16}/></button></div>
              <h3>{project.name}</h3>
              <p>{project.description || '尚未添加项目描述'}</p>
              <div className="progress"><i style={{ width: project.status === 'DRAFT' ? '18%' : '60%' }} /></div>
              <footer><span>{project.status === 'DRAFT' ? '草稿' : project.status}</span><time>{new Date(project.updatedAt).toLocaleDateString('zh-CN')}</time></footer>
            </article>
          })}
          {projectQuery && projects.data?.items.filter(project=>`${project.name} ${project.description} ${typeDisplay(project.type,contentTypes.data?.items).name}`.toLowerCase().includes(projectQuery.trim().toLowerCase())).length===0&&<p className="empty-line">没有匹配的项目</p>}
          <button className="new-card" onClick={() => setOpen(true)}><Plus/><strong>开始新的创作</strong><small>小说、电影解说或技术文档</small></button>
        </section>
        </> : <GlobalPage page={activeNav} projects={projects.data?.items ?? []} />}
        </div>
      </main>

      <Dialog open={open} onOpenChange={setOpen}><DialogContent className="dialog"><form onSubmit={submit}>
          <DialogHeader className="dialog-head"><div><p className="eyebrow">NEW PROJECT</p><DialogTitle>创建内容项目</DialogTitle><DialogDescription>选择内容类型并填写项目目标。</DialogDescription></div></DialogHeader>
          <label>项目名称<Input autoFocus required maxLength={120} value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：产品 API 使用指南" /></label>
          <fieldset><legend>内容类型</legend><div className="type-grid">
            {contentTypes.data?.items.map((item) => <button className={type === item.code ? 'selected' : ''} type="button" key={item.code} onClick={() => setType(item.code)}><b>{item.icon}</b><span>{item.name}</span></button>)}
          </div></fieldset>
          <label>创作说明<textarea rows={4} maxLength={1000} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="描述目标、受众和内容要求…" /></label>
          {create.isError && <p className="form-error">{create.error.message}</p>}
          <div className="dialog-actions"><button type="button" className="secondary" onClick={() => setOpen(false)}>取消</button><button className="primary" disabled={create.isPending}>{create.isPending ? '创建中…' : '创建项目'}</button></div>
        </form></DialogContent></Dialog>
      <Dialog open={editingProject!==null} onOpenChange={(nextOpen)=>{if(!nextOpen&&!updateProject.isPending)setEditingProject(null)}}><DialogContent className="dialog project-edit-dialog"><form onSubmit={submitEdit}>
          <DialogHeader className="dialog-head"><div><p className="eyebrow">EDIT PROJECT</p><DialogTitle>编辑项目</DialogTitle><DialogDescription>修改项目名称和创作说明，内容类型保持不变。</DialogDescription></div></DialogHeader>
          <label>项目名称<Input autoFocus required maxLength={120} value={editName} onChange={(event)=>setEditName(event.target.value)} placeholder="请输入项目名称" /></label>
          <label>创作说明<textarea rows={4} maxLength={1000} value={editDescription} onChange={(event)=>setEditDescription(event.target.value)} placeholder="描述目标、受众和内容要求" /></label>
          <div className="dialog-actions"><button type="button" className="secondary" disabled={updateProject.isPending} onClick={()=>setEditingProject(null)}>取消</button><button className="primary" disabled={updateProject.isPending||!editName.trim()}>{updateProject.isPending?'保存中…':'保存修改'}</button></div>
        </form></DialogContent></Dialog>
      {selected && <Workspace project={selected} contentTypes={contentTypes.data?.items??[]} initialTab={selectedTab} onClose={() => setSelected(null)} />}
      <Toaster richColors position="top-right"/>
    </div>
  )
}

function GlobalPage({ page, projects }: { page: 'knowledge' | 'types' | 'tasks' | 'models'; projects: Project[] }) {
  const queryClient = useQueryClient()
  const [taskFilter,setTaskFilter]=useState('ALL')
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: api.tasks, enabled: page === 'tasks', refetchInterval: page === 'tasks' ? 2000 : false })
  const retryTask=useMutation({mutationFn:api.retryTask,onSuccess:()=>queryClient.invalidateQueries({queryKey:['tasks']})})
  const models = useQuery({ queryKey: ['model-status'], queryFn: api.modelStatus, enabled: page === 'models' })

  if (page === 'knowledge') return <>
    <KnowledgeFileManager projects={projects}/>
  </>

  if(page==='types')return <ContentTypeManager/>

  if (page === 'tasks') return <>
    <header><div><p className="eyebrow">TASK CENTER</p><h1>任务中心</h1><p>查看后台生成、校验和处理任务的实时状态。</p></div></header>
    <section className="global-panel"><div className="section-head"><div><h2>全部任务</h2><p>{tasks.data?.total ?? 0} 条任务记录</p></div><select className="task-filter" value={taskFilter} onChange={event=>setTaskFilter(event.target.value)}><option value="ALL">全部状态</option><option value="RUNNING">运行中</option><option value="FAILED">失败</option><option value="SUCCESS">成功</option><option value="CANCELLED">已取消</option></select></div><div className="task-table">{tasks.data?.items.filter(item=>taskFilter==='ALL'||item.status===taskFilter).map((item) => <article key={item.id}><span className={`task-state ${item.status.toLowerCase()}`}>{item.status}</span><div><strong>{item.type}</strong><small>{item.error||item.message}</small><time>{new Date(item.createdAt).toLocaleString('zh-CN')}</time></div><div className="mini-progress"><i style={{ width: `${item.progress}%` }} /></div>{['FAILED','CANCELLED'].includes(item.status)?<button className="task-retry" disabled={retryTask.isPending} onClick={()=>retryTask.mutate(item.id)}>重试</button>:<b>{item.progress}%</b>}</article>)}{tasks.data?.total === 0 && <p className="empty-line">暂无后台任务。进入项目的“多模型校验”即可创建任务。</p>}</div>{retryTask.isError&&<p className="quality-error">{retryTask.error.message}</p>}</section>
  </>

  return <>
    <header><div><p className="eyebrow">MODEL ROUTING</p><h1>模型与校验</h1><p>检查服务端模型路由，并进入项目执行严格质量校验。</p></div></header>
    <section className="metrics model-metrics"><article><span>配置状态</span><strong className="text-value">{models.data?.configured ? '已就绪' : '未配置'}</strong><small>OpenAI-Compatible Provider</small></article><article><span>Validator</span><strong>{models.data?.validatorCount ?? 0}</strong><small>并行独立校验模型</small></article><article><span>Judge</span><strong className="text-value">{models.data?.judgeConfigured ? '已配置' : '未配置'}</strong><small>只处理重大模型分歧</small></article></section>
    {!models.data?.configured && <div className="setup-callout"><strong>需要配置模型环境变量</strong><code>LLM_BASE_URL · LLM_API_KEY · VALIDATOR_MODELS · JUDGE_MODEL</code><span>设置后重启 Go API。</span></div>}
    <ModelConfigPanel/>
  </>
}

function ContentTypeManager(){
  const queryClient=useQueryClient()
  const types=useQuery({queryKey:['content-types'],queryFn:api.contentTypes})
  const[editing,setEditing]=useState<ContentType|null>(null)
  const[open,setOpen]=useState(false)
  const[code,setCode]=useState('')
  const[name,setName]=useState('')
  const[icon,setIcon]=useState('')
  const[accent,setAccent]=useState('amber')
  const[description,setDescription]=useState('')
  const[prompt,setPrompt]=useState('')
  const reset=(item?:ContentType)=>{setEditing(item??null);setCode(item?.code??'');setName(item?.name??'');setIcon(item?.icon??'');setAccent(item?.accent??'amber');setDescription(item?.description??'');setPrompt(item?.prompt??'');setOpen(true)}
  const save=useMutation({mutationFn:()=>editing?api.updateContentType(editing.code,{name,icon,accent,description,prompt}):api.createContentType({code,name,icon,accent,description,prompt}),onSuccess:()=>{toast.success(editing?'内容类型已更新':'内容类型已创建');setOpen(false);queryClient.invalidateQueries({queryKey:['content-types']});queryClient.invalidateQueries({queryKey:['projects']})}})
  const remove=useMutation({mutationFn:api.deleteContentType,onSuccess:()=>{toast.success('内容类型已删除');queryClient.invalidateQueries({queryKey:['content-types']})},onError:error=>toast.error(error.message)})
  const submit=(event:FormEvent)=>{event.preventDefault();save.mutate()}
  return <>
    <header><div><p className="eyebrow">CONTENT TYPES</p><h1>内容类型</h1><p>维护项目可选择的内容类型、视觉标识和贯穿文档生成流程的创作提示词。</p></div><Button className="primary" onClick={()=>reset()}><Plus size={16}/>新增类型</Button></header>
    <section className="type-manager-head"><div><strong>{types.data?.total??0}</strong><span>个可用类型</span></div><p>类型编码用于提示词和数据关联，创建后不可修改；生成提示词会自动应用于目录、正文及后续 AI 文档任务。</p></section>
    <section className="content-type-grid">
      {types.data?.items.map(item=><article key={item.code}>
        <div className={`project-icon ${item.accent}`}>{item.icon}</div>
        <div className="content-type-title"><h2>{item.name}</h2><code>{item.code}</code></div>
        <p>{item.description||'尚未填写类型说明'}</p>
        <div className={`type-prompt-status ${item.prompt?'configured':'empty'}`}><Sparkles size={13}/>{item.prompt?'已配置生成提示词':'未配置生成提示词'}</div>
        <footer><button onClick={()=>reset(item)}><Pencil size={14}/>编辑</button><button className="danger" disabled={remove.isPending} onClick={()=>window.confirm(`确定删除“${item.name}”？已被项目使用时将无法删除。`)&&remove.mutate(item.code)}><Trash2 size={14}/>删除</button></footer>
      </article>)}
      {!types.isLoading&&types.data?.total===0&&<div className="file-empty"><Shapes/><strong>暂无内容类型</strong><span>新增一个类型后即可创建项目</span></div>}
    </section>
    <Dialog open={open} onOpenChange={setOpen}><DialogContent className="dialog content-type-dialog"><form onSubmit={submit}>
      <DialogHeader className="content-type-dialog-head"><DialogTitle>{editing?'编辑内容类型':'新增内容类型'}</DialogTitle><DialogDescription>配置类型信息和后续目录、正文等 AI 文档任务共同使用的创作提示词。</DialogDescription></DialogHeader>
      <div className="type-form-row"><label>类型名称<Input required maxLength={80} value={name} onChange={event=>setName(event.target.value)} placeholder="例如：营销文案"/></label><label>图标文字<Input maxLength={4} value={icon} onChange={event=>setIcon(event.target.value)} placeholder="例如：营"/></label></div>
      <label>类型编码<Input required disabled={!!editing} maxLength={40} value={code} onChange={event=>setCode(event.target.value.toUpperCase().replace(/[^A-Z0-9_]/g,''))} placeholder="例如：MARKETING_COPY"/></label>
      <label>配色<Select value={accent} onChange={event=>setAccent(event.target.value)}><option value="amber">琥珀色</option><option value="blue">蓝色</option><option value="green">绿色</option><option value="rose">玫红色</option><option value="violet">紫色</option></Select></label>
      <label>类型说明<textarea rows={3} maxLength={500} value={description} onChange={event=>setDescription(event.target.value)} placeholder="说明适用的内容场景"/></label>
      <label>生成提示词<textarea required rows={8} maxLength={12000} value={prompt} onChange={event=>setPrompt(event.target.value)} placeholder="描述该内容类型的结构、风格、事实边界、质量要求和必须遵守的规则"/><small>作为系统级类型规范，自动用于创作简报、目录、正文、润色、修复、事实抽取和长期记忆。</small></label>
      {save.isError&&<p className="form-error">{save.error.message}</p>}
      <div className="dialog-actions"><button type="button" className="secondary" onClick={()=>setOpen(false)}>取消</button><button className="primary" disabled={save.isPending}>{save.isPending?'保存中…':'保存'}</button></div>
    </form></DialogContent></Dialog>
  </>
}

function KnowledgeFileManager({projects}:{projects:Project[]}){const queryClient=useQueryClient();const inputRef=useRef<HTMLInputElement|null>(null);const[projectId,setProjectId]=useState('');const[uploadProject,setUploadProject]=useState(projects[0]?.id??'');const[typeFilter,setTypeFilter]=useState('ALL');const[query,setQuery]=useState('');const[dragging,setDragging]=useState(false);useEffect(()=>{if(!uploadProject&&projects[0])setUploadProject(projects[0].id)},[projects,uploadProject]);const files=useQuery({queryKey:['knowledge-files',projectId],queryFn:()=>api.knowledgeFiles(projectId)});const uploadFiles=useMutation({mutationFn:async(items:File[])=>{if(!uploadProject)throw new Error('请先选择文件归属项目');for(const file of items)await api.uploadKnowledge(uploadProject,file,'REFERENCE');return items.length},onSuccess:count=>{toast.success(`已上传 ${count} 个文件`);queryClient.invalidateQueries({queryKey:['knowledge-files']})}});const remove=useMutation({mutationFn:api.deleteKnowledgeFile,onSuccess:()=>{toast.success('文件已删除');queryClient.invalidateQueries({queryKey:['knowledge-files']})}});const projectNames=Object.fromEntries(projects.map(item=>[item.id,item.name]));const category=(file:KnowledgeFile)=>['.png','.jpg','.jpeg','.gif','.webp','.svg'].includes(file.extension)?'IMAGE':['.doc','.docx'].includes(file.extension)?'WORD':['.ppt','.pptx'].includes(file.extension)?'PPT':file.extension==='.pdf'?'PDF':'TEXT';const visible=(files.data?.items??[]).filter(file=>(typeFilter==='ALL'||category(file)===typeFilter)&&file.name.toLowerCase().includes(query.toLowerCase()));const totalSize=(files.data?.items??[]).reduce((sum,item)=>sum+item.size,0);const submit=(list:FileList|null)=>{const items=Array.from(list??[]);if(items.length)uploadFiles.mutate(items)};const formatSize=(size:number)=>size>=1048576?`${(size/1048576).toFixed(1)} MB`:size>=1024?`${(size/1024).toFixed(1)} KB`:`${size} B`;return <><header><div><p className="eyebrow">KNOWLEDGE FILES</p><h1>知识文件库</h1><p>统一管理 PDF、Word、PowerPoint、图片和 Markdown，作为所有项目的可信资料来源。</p></div><Button onClick={()=>inputRef.current?.click()}><Upload size={16}/>上传文件</Button></header><section className="metrics file-metrics"><article><span>全部文件</span><strong>{files.data?.total??0}</strong><small>跨项目统一管理</small></article><article><span>已建立索引</span><strong>{files.data?.items.filter(item=>item.status==='INDEXED').length??0}</strong><small>可直接参与知识检索</small></article><article><span>本地存储</span><strong className="text-value">{formatSize(totalSize)}</strong><small>原文件保存在本机</small></article></section><section className={`file-dropzone ${dragging?'dragging':''}`} onDragOver={event=>{event.preventDefault();setDragging(true)}} onDragLeave={()=>setDragging(false)} onDrop={event=>{event.preventDefault();setDragging(false);submit(event.dataTransfer.files)}}><input ref={inputRef} hidden multiple type="file" accept=".pdf,.doc,.docx,.ppt,.pptx,.png,.jpg,.jpeg,.gif,.webp,.svg,.md,.markdown,.txt,.json,.csv,.html" onChange={event=>submit(event.target.files)}/><div className="drop-icon"><Upload/></div><div><strong>拖拽文件到这里，或点击右上角上传</strong><p>支持 PDF、Word、PPT、图片、Markdown，单个文件最大 50 MB</p></div><label>归属项目<Select value={uploadProject} onChange={event=>setUploadProject(event.target.value)}><option value="">请选择项目</option>{projects.map(project=><option value={project.id} key={project.id}>{project.name}</option>)}</Select></label></section>{uploadFiles.isError&&<p className="quality-error">{uploadFiles.error.message}</p>}<section className="file-library"><div className="file-toolbar"><div className="file-search"><Search size={15}/><input value={query} onChange={event=>setQuery(event.target.value)} placeholder="搜索文件名"/></div><Select value={projectId} onChange={event=>setProjectId(event.target.value)}><option value="">全部项目</option>{projects.map(project=><option value={project.id} key={project.id}>{project.name}</option>)}</Select><div className="file-types">{['ALL','PDF','WORD','PPT','IMAGE','TEXT'].map(value=><button className={typeFilter===value?'selected':''} onClick={()=>setTypeFilter(value)} key={value}>{value==='ALL'?'全部':value}</button>)}</div></div><div className="file-table"><header><span>文件</span><span>所属项目</span><span>大小</span><span>状态</span><span>上传时间</span><span>操作</span></header>{visible.map(file=><article key={file.id}><div className={`file-kind ${category(file).toLowerCase()}`}>{category(file)==='IMAGE'?<FileImage/>:<FileText/>}</div><div className="file-name"><strong>{file.name}</strong><small>{file.extension.toUpperCase()} · {file.mimeType||'未知类型'}</small></div><span>{projectNames[file.projectId]??file.projectId}</span><span>{formatSize(file.size)}</span><b className={`file-status ${file.status.toLowerCase()}`}>{file.status==='INDEXED'?'已索引':'已存储'}</b><time>{new Date(file.createdAt).toLocaleDateString('zh-CN')}</time><div className="file-actions"><a title="下载" href={api.knowledgeFileDownloadURL(file.id)}><Download size={15}/></a><button title="删除" onClick={()=>window.confirm(`确定删除 ${file.name}？`)&&remove.mutate(file.id)}><Trash2 size={15}/></button></div></article>)}{visible.length===0&&<div className="file-empty"><Database/><strong>暂无匹配文件</strong><span>上传资料后会显示在这里</span></div>}</div></section></>}

type ProviderDraft={baseUrl:string;model:string;apiKey:string;enabled:boolean;clearApiKey:boolean;hasApiKey:boolean}
const providerPayload=({baseUrl,model,apiKey,enabled,clearApiKey}:ProviderDraft)=>({baseUrl,model,apiKey,enabled,clearApiKey})
function ModelConfigPanel(){
  const queryClient=useQueryClient()
  const config=useQuery({queryKey:['local-model-config'],queryFn:api.localModelConfig})
  const[active,setActive]=useState<'deepseek'|'minimax'>('deepseek')
  const[deepseek,setDeepSeek]=useState<ProviderDraft>({baseUrl:'https://api.deepseek.com',model:'deepseek-v4-flash',apiKey:'',enabled:false,clearApiKey:false,hasApiKey:false})
  const[minimax,setMiniMax]=useState<ProviderDraft>({baseUrl:'https://api.minimaxi.com/v1',model:'MiniMax-M2.7',apiKey:'',enabled:false,clearApiKey:false,hasApiKey:false})
  useEffect(()=>{const value=config.data;if(!value)return;setActive(value.activeProvider);setDeepSeek({...value.deepseek,apiKey:'',clearApiKey:false});setMiniMax({...value.minimax,apiKey:'',clearApiKey:false})},[config.data])
  const save=useMutation({mutationFn:()=>api.saveLocalModelConfig({activeProvider:active,deepseek:providerPayload(deepseek),minimax:providerPayload(minimax)}),onSuccess:()=>{toast.success('配置已保存，请重启 Go API 生效');queryClient.invalidateQueries({queryKey:['local-model-config']})}})
  const currentPayload=()=>({activeProvider:active,deepseek:providerPayload(deepseek),minimax:providerPayload(minimax)})
  const test=useMutation({mutationFn:async()=>{const result=await api.testLocalModelConfig({provider:active,config:providerPayload(active==='minimax'?minimax:deepseek)});await api.saveLocalModelConfig(currentPayload());return result},onSuccess:result=>{toast.success(`连接成功并已保存：${result.model} · ${result.latencyMs} ms`);queryClient.invalidateQueries({queryKey:['local-model-config']})}})
  const providerCard=(id:'deepseek'|'minimax',label:string,draft:ProviderDraft,setDraft:(value:ProviderDraft)=>void,models:string[])=><article className={`provider-config ${active===id?'active':''}`}><header><div><span className="provider-mark">{id==='deepseek'?'DS':'MM'}</span><div><h3>{label}</h3><small>{draft.hasApiKey?'API Key 已安全保存在本机':'尚未设置 API Key'}</small></div></div><label className="provider-switch"><input type="radio" name="activeProvider" checked={active===id} onChange={()=>setActive(id)}/> 设为当前 Provider</label></header><label>Base URL<Input value={draft.baseUrl} onChange={event=>setDraft({...draft,baseUrl:event.target.value})}/></label><label>模型<Select value={draft.model} onChange={event=>setDraft({...draft,model:event.target.value})}>{models.map(model=><option value={model} key={model}>{model}</option>)}</Select></label><label>API Key<Input type="password" autoComplete="new-password" value={draft.apiKey} onChange={event=>setDraft({...draft,apiKey:event.target.value,clearApiKey:false})} placeholder={draft.hasApiKey?'留空则保留现有 Key':'输入 API Key'}/></label><div className="provider-options"><label><input type="checkbox" checked={draft.enabled} onChange={event=>setDraft({...draft,enabled:event.target.checked})}/> 启用</label>{draft.hasApiKey&&<label><input type="checkbox" checked={draft.clearApiKey} onChange={event=>setDraft({...draft,clearApiKey:event.target.checked,apiKey:''})}/> 清除本地 Key</label>}</div></article>
  return <section className="model-config-panel"><div className="section-head"><div><p className="eyebrow">LOCAL PROVIDERS</p><h2>本地模型配置</h2><p>Key 仅写入 <code>{config.data?.path??'.local/model-config.json'}</code>，不会保存到数据库或返回明文。</p></div><div className="model-config-actions"><Button variant="secondary" disabled={test.isPending||config.isLoading} onClick={()=>test.mutate()}>{test.isPending?'测试并保存中…':'测试并保存'}</Button><Button disabled={save.isPending||config.isLoading} onClick={()=>save.mutate()}>{save.isPending?'保存中…':'保存配置'}</Button></div></div><div className="provider-grid">{providerCard('deepseek','DeepSeek',deepseek,setDeepSeek,['deepseek-v4-flash','deepseek-v4-pro'])}{providerCard('minimax','MiniMax',minimax,setMiniMax,['MiniMax-M2.7','MiniMax-M2.7-highspeed','MiniMax-M2.5'])}</div>{save.isError&&<p className="quality-error">保存失败：{save.error.message}</p>}{test.isError&&<p className="quality-error">测试或保存失败：{test.error.message}</p>}{test.isSuccess&&<p className="model-test-success">连接成功且配置已保存：{test.data.model} · {test.data.latencyMs} ms</p>}<div className="local-security-note"><Database size={15}/><span>“测试并保存”通过后会立即持久化；刷新时仅隐藏 Key 明文，并显示“Key 已保存”。</span></div></section>
}

function Workspace({ project, contentTypes, initialTab, onClose }: { project: Project; contentTypes:ContentType[]; initialTab: WorkspaceTab; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<WorkspaceTab>(initialTab)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [expandOpen,setExpandOpen]=useState(false)
  const [expandInstruction,setExpandInstruction]=useState('保持现有内容的语气、结构和事实基础，补充细节、过渡与论述，使内容更加完整。')
  const [expandQuery,setExpandQuery]=useState('')
  const [expandTaskId,setExpandTaskId]=useState('')
  const [selectedDocument, setSelectedDocument] = useState<Document | null>(null)
  const [documentMode,setDocumentMode]=useState<'generator'|'library'>('generator')
  const [workspaceNavCollapsed,setWorkspaceNavCollapsed]=useState(false)
  const [documentBrowserCollapsed,setDocumentBrowserCollapsed]=useState(false)
  const [batchDocumentIds,setBatchDocumentIds]=useState<string[]>([])
  const [workspaceBatchTaskId,setWorkspaceBatchTaskId]=useState('')
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
  const documentTree=useQuery({queryKey:['tree',project.id],queryFn:()=>api.tree(project.id)})
  const orderedDocuments=(()=>{
    const items=documents.data?.items??[]
    const nodes=documentTree.data?.items??[]
    const children=new Map<string,typeof nodes>()
    nodes.forEach(node=>{const parent=node.parentId??'';children.set(parent,[...(children.get(parent)??[]),node])})
    const documentOrder:string[]=[]
    const visited=new Set<string>()
    const walk=(parent:string)=>{(children.get(parent)??[]).slice().sort((a,b)=>a.position-b.position||a.title.localeCompare(b.title,'zh-CN')).forEach(node=>{visited.add(node.id);if(node.documentId)documentOrder.push(node.documentId);walk(node.id)})}
    walk('')
    nodes.filter(node=>!visited.has(node.id)).sort((a,b)=>a.position-b.position).forEach(node=>{if(node.documentId)documentOrder.push(node.documentId)})
    const rank=new Map(documentOrder.map((id,index)=>[id,index]))
    return items.slice().sort((a,b)=>(rank.get(a.id)??Number.MAX_SAFE_INTEGER)-(rank.get(b.id)??Number.MAX_SAFE_INTEGER))
  })()
  const sources = useQuery({ queryKey: ['sources', project.id], queryFn: () => api.sources(project.id) })
  const search = useQuery({ queryKey: ['knowledge-search', project.id, submittedQuery], queryFn: () => api.searchKnowledge(project.id, submittedQuery), enabled: submittedQuery.length > 0 })
  const modelStatus = useQuery({ queryKey: ['model-status'], queryFn: api.modelStatus })
  const workspaceBatchTask=useQuery({queryKey:['workspace-batch-task',workspaceBatchTaskId],queryFn:()=>api.task(workspaceBatchTaskId),enabled:Boolean(workspaceBatchTaskId),refetchInterval:query=>['SUCCESS','FAILED','CANCELLED'].includes(query.state.data?.status??'')?false:700})
  const workspaceBatchRunning=workspaceBatchTask.data&&!['SUCCESS','FAILED','CANCELLED'].includes(workspaceBatchTask.data.status)
  const selectedBatchNodeIds=(documentTree.data?.items??[]).filter(node=>node.documentId&&batchDocumentIds.includes(node.documentId)).map(node=>node.id)
  const startWorkspaceBatch=useMutation({mutationFn:()=>api.batchGenerate(project.id,{nodeIds:selectedBatchNodeIds,instruction:`为《${project.name}》中的所选章节生成完整正文。保持章节之间设定一致，只输出当前章节的 Markdown 正文。`,knowledgeQuery:'',windowSize:2}),onSuccess:task=>setWorkspaceBatchTaskId(task.id)})
  const cancelWorkspaceBatch=useMutation({mutationFn:()=>api.cancelTask(workspaceBatchTaskId),onSuccess:()=>queryClient.invalidateQueries({queryKey:['workspace-batch-task',workspaceBatchTaskId]})})
  const activeTask = useQuery({ queryKey: ['task', taskId], queryFn: () => api.task(taskId), enabled: taskId.length > 0, refetchInterval: (query) => ['SUCCESS', 'FAILED', 'CANCELLED'].includes(query.state.data?.status ?? '') ? false : 1000 })
  const generationTask = useQuery({ queryKey: ['generation-task', generationTaskId], queryFn: () => api.task<GenerationResult>(generationTaskId), enabled: generationTaskId.length > 0, refetchInterval: (query) => ['SUCCESS', 'FAILED', 'CANCELLED'].includes(query.state.data?.status ?? '') ? false : 1000 })
  const qualityTask = useQuery({ queryKey: ['quality-generation-task', qualityTaskId], queryFn: () => api.task<QualityGenerationResult>(qualityTaskId), enabled: qualityTaskId.length > 0, refetchInterval: (query) => ['SUCCESS', 'FAILED', 'CANCELLED'].includes(query.state.data?.status ?? '') ? false : 1000 })
  useEffect(()=>setSelectedDocument(null),[project.id])
  useEffect(()=>{if(documentMode==='library'&&!selectedDocument&&orderedDocuments[0])setSelectedDocument(orderedDocuments[0])},[documentMode,selectedDocument,orderedDocuments])
  useEffect(()=>{if(selectedDocument){const updated=orderedDocuments.find(item=>item.id===selectedDocument.id);if(updated&&updated.currentVersionId!==selectedDocument.currentVersionId)setSelectedDocument(updated)}},[orderedDocuments,selectedDocument])
  useEffect(()=>{if(workspaceBatchTask.data&&['SUCCESS','FAILED','CANCELLED'].includes(workspaceBatchTask.data.status)){queryClient.invalidateQueries({queryKey:['documents',project.id]});for(const id of batchDocumentIds)queryClient.invalidateQueries({queryKey:['versions',id]})}},[workspaceBatchTask.data?.status])
  const expandTask=useQuery({queryKey:['expand-task',expandTaskId],queryFn:()=>api.task<GenerationResult>(expandTaskId),enabled:Boolean(expandTaskId),refetchInterval:query=>['SUCCESS','FAILED','CANCELLED'].includes(query.state.data?.status??'')?false:1000})
  const createDocument = useMutation({ mutationFn: () => api.createDocument(project.id, { title, content }), onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTitle(''); setContent('') } })
  const createSource = useMutation({ mutationFn: () => api.createSource(project.id, { name: sourceName, authority, content: sourceContent }), onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['sources', project.id] }); setSourceName(''); setSourceContent('') } })
  const validate = useMutation({ mutationFn: () => api.createValidationTask(project.id, { text: validationText, task: '校验文本的事实依据、一致性、完整性、术语和风格', knowledgeQuery: validationQuery, dimensions: ['groundedness', 'consistency', 'completeness', 'terminology', 'style'] }), onSuccess: (task) => setTaskId(task.id) })
  const cancelTask = useMutation({ mutationFn: () => api.cancelTask(taskId), onSuccess: () => queryClient.invalidateQueries({ queryKey: ['task', taskId] }) })
  const generate = useMutation({ mutationFn: () => api.createGenerationTask(project.id, { operation, instruction, title: generationTitle, documentId: operation === 'POLISH' ? generationDocumentId : '', knowledgeQuery: generationQuery }), onSuccess: (created) => setGenerationTaskId(created.id) })
  const qualityGenerate = useMutation({ mutationFn: () => api.qualityGenerate(project.id, { instruction, title: generationTitle, knowledgeQuery: generationQuery, maxRepairs }), onSuccess: (created) => setQualityTaskId(created.id) })
  const expand=useMutation({mutationFn:()=>api.createGenerationTask(project.id,{operation:'WRITE',instruction:`扩写文档《${title||'未命名文档'}》。${expandInstruction}\n只输出新增或重写后的正文，不要解释处理过程。`,title:title||'AI 扩写',documentId:'',knowledgeQuery:expandQuery,content,save:false}),onSuccess:task=>setExpandTaskId(task.id)})
  const outlineToTree=useMutation({mutationFn:()=>api.importOutline(project.id,{content:generationTask.data!.result!.generation.content,preview:false}),onSuccess:data=>{queryClient.invalidateQueries({queryKey:['tree',project.id]});toast.success(`已创建 ${data.total} 个内容节点`);setTab('structure')}})
  const validationResult = activeTask.data?.result

  return <div className="workspace-layer">
    <div className="workspace-bar"><button onClick={onClose}><ArrowLeft size={14}/> 返回项目</button><div><small>{typeDisplay(project.type,contentTypes).name}</small><strong>{project.name}</strong></div><a className="export-link" href={api.exportURL(project.id)}><Download size={14}/> 导出 Markdown</a><span>草稿工作区</span></div>
    <div className={workspaceNavCollapsed?'workspace-body nav-collapsed':'workspace-body'}>
      <aside className="workspace-nav"><button className={tab === 'documents' ? 'selected' : ''} onClick={() => setTab('documents')}>文档与版本</button><button className={tab === 'structure' ? 'selected' : ''} onClick={() => setTab('structure')}>内容结构</button><button className={tab === 'memory' ? 'selected' : ''} onClick={() => setTab('memory')}>Story Memory</button><button className={tab === 'quality' ? 'selected' : ''} onClick={() => setTab('quality')}>多模型校验</button><button className={tab === 'runs' ? 'selected' : ''} onClick={() => setTab('runs')}>AI 运行记录</button><button className={tab === 'storyline' ? 'selected' : ''} onClick={() => setTab('storyline')}>故事线</button><div className="pipeline"><small>准确性流水线</small><p>资料检索</p><i /><p>模型生成</p><i /><p>双模型校验</p><i /><p>质量门禁</p></div></aside>
      <button className="workspace-nav-toggle" title={workspaceNavCollapsed?'展开功能导航':'折叠功能导航'} aria-label={workspaceNavCollapsed?'展开功能导航':'折叠功能导航'} onClick={()=>setWorkspaceNavCollapsed(value=>!value)}>{workspaceNavCollapsed?<ChevronRight/>:<ChevronLeft/>}</button>
      <section className="workspace-content">
        {tab === 'documents' ? <>
          <div className="document-module-head"><div><Sparkles/><div><h2>章节文档生成器</h2><p>描述需求，生成并调整目录，再批量创建 Markdown 文档。</p></div></div><nav><button className={documentMode==='generator'?'active':''} onClick={()=>setDocumentMode('generator')}>生成器</button><button className={documentMode==='library'?'active':''} onClick={()=>setDocumentMode('library')}>文档工作区</button><span>{documents.data?.total??0} 个文档</span></nav></div>
          {documentMode==='generator'?<ChapterDocumentGenerator project={project} modelReady={modelStatus.data?.generationOperations??[]} onOpenWorkspace={()=>setDocumentMode('library')}/>:<div className={documentBrowserCollapsed?'document-workbench browser-collapsed':'document-workbench'}>
            <aside className="document-browser"><header><div><strong>{project.name}</strong><small>{documents.data?.total??0} 个 Markdown 文档</small></div><button className="document-select-all" onClick={()=>setBatchDocumentIds(batchDocumentIds.length===orderedDocuments.length?[]:orderedDocuments.map(item=>item.id))}>{batchDocumentIds.length===orderedDocuments.length?'取消全选':'全选'}</button></header><div className="document-batch-toolbar"><button className="batch-generate" disabled={!batchDocumentIds.length||Boolean(workspaceBatchRunning)||startWorkspaceBatch.isPending} onClick={()=>startWorkspaceBatch.mutate()}>{workspaceBatchRunning?`批量生成 ${workspaceBatchTask.data?.progress??0}%`:`批量生成 ${batchDocumentIds.length} 篇`}</button>{workspaceBatchRunning&&<button className="batch-cancel" disabled={cancelWorkspaceBatch.isPending} onClick={()=>cancelWorkspaceBatch.mutate()}>中断</button>}<small>{workspaceBatchRunning?workspaceBatchTask.data?.message:'勾选需要生成的文档'}</small>{(startWorkspaceBatch.isError||workspaceBatchTask.data?.status==='FAILED')&&<p>{startWorkspaceBatch.error?.message||workspaceBatchTask.data?.error}；再次发起将继续未完成文档。</p>}</div><div className="document-browser-list">{orderedDocuments.map(item=><div className="document-browser-item" key={item.id}><input type="checkbox" aria-label={`选择 ${item.title}`} checked={batchDocumentIds.includes(item.id)} onChange={event=>setBatchDocumentIds(current=>event.target.checked?[...current,item.id]:current.filter(id=>id!==item.id))}/><button className={selectedDocument?.id===item.id?'selected':''} onClick={()=>setSelectedDocument(item)}><FileText size={16}/><span><strong>{item.title}</strong><small>版本 {item.versionCount} · {new Date(item.updatedAt).toLocaleString('zh-CN')}</small></span></button></div>)}{documents.data?.total===0&&<p className="empty-line">暂无文档，请先通过生成器创建。</p>}</div></aside>
            <button className="document-browser-toggle" title={documentBrowserCollapsed?'展开文档列表':'折叠文档列表'} aria-label={documentBrowserCollapsed?'展开文档列表':'折叠文档列表'} onClick={()=>setDocumentBrowserCollapsed(value=>!value)}>{documentBrowserCollapsed?<ChevronRight/>:<ChevronLeft/>}</button>
            <section className="document-workbench-main">{selectedDocument?<DocumentEditor key={`${selectedDocument.id}-${selectedDocument.currentVersionId}`} document={selectedDocument} embedded onBack={()=>setSelectedDocument(null)}/>:<div className="document-workbench-empty"><FileText/><strong>选择一篇文档开始编辑</strong><span>正文将使用 Markdown 编辑器打开，并保留版本历史。</span></div>}</section>
          </div>}
        </> : tab === 'structure' ? <StructurePanel project={project} documents={documents.data?.items ?? []} /> : tab === 'generation' ? <>
          <div className="workspace-title"><div><p className="eyebrow">AGENT PIPELINE</p><h2>AI 内容创作</h2><p>使用 Planner、Outliner、Writer 或 Polisher，并将结果自动保存为文档版本。</p></div></div>
          <div className={`model-status ${(modelStatus.data?.generationOperations?.length ?? 0) > 0 ? 'ready' : 'offline'}`}><strong>{(modelStatus.data?.generationOperations?.length ?? 0) > 0 ? '创作模型已就绪' : '创作模型未配置'}</strong><span>{modelStatus.data?.generationOperations?.join(' · ') || '请配置 WRITER_MODEL 或角色专用模型'}</span></div>
          <form className="studio-form generation-form" onSubmit={(event) => { event.preventDefault(); if (strictGeneration) { setQualityTaskId(''); qualityGenerate.mutate() } else { setGenerationTaskId(''); generate.mutate() } }}><div className="operation-grid">{[['PLAN', '策划'], ['OUTLINE', '目录'], ['WRITE', '正文'], ['POLISH', '润色']].map(([value, label]) => <button type="button" className={operation === value ? 'selected' : ''} onClick={() => { setOperation(value); if (value !== 'WRITE') setStrictGeneration(false) }} key={value}><b>{value}</b><span>{label}</span></button>)}</div>{operation === 'POLISH' ? <label>选择待润色文档<select required value={generationDocumentId} onChange={(event) => setGenerationDocumentId(event.target.value)}><option value="">请选择文档</option>{documents.data?.items.map((item) => <option value={item.id} key={item.id}>{item.title}</option>)}</select></label> : <label>输出文档标题<input value={generationTitle} onChange={(event) => setGenerationTitle(event.target.value)} placeholder="留空则使用默认标题" /></label>}<label>创作要求<textarea required rows={7} value={instruction} onChange={(event) => setInstruction(event.target.value)} placeholder="描述内容目标、受众、风格、篇幅和必须覆盖的要点…" /></label><label>知识检索词<input value={generationQuery} onChange={(event) => setGenerationQuery(event.target.value)} placeholder="从项目知识库检索相关证据（可选）" /></label>{operation === 'WRITE' && <div className="strict-options"><label><input type="checkbox" checked={strictGeneration} onChange={(event) => setStrictGeneration(event.target.checked)} /> 严格质量生成（生成 → 多 Validator 校验 → Repair）</label>{strictGeneration && <label>最多修复次数<select value={maxRepairs} onChange={(event) => setMaxRepairs(Number(event.target.value))}><option value={1}>1 次</option><option value={2}>2 次</option><option value={3}>3 次</option></select></label>}</div>}<button className="primary" disabled={generate.isPending || qualityGenerate.isPending || generationTask.data?.status === 'RUNNING' || qualityTask.data?.status === 'RUNNING' || !(modelStatus.data?.generationOperations?.includes(operation))}>{generate.isPending || qualityGenerate.isPending ? '正在创建任务…' : strictGeneration ? '运行严格质量生成' : `运行 ${operation} Agent`}</button></form>
          {generationTask.data && !['SUCCESS', 'FAILED', 'CANCELLED'].includes(generationTask.data.status) && <div className="task-progress"><div><strong>{generationTask.data.message}</strong><span>{generationTask.data.progress}%</span></div><i><b style={{ width: `${generationTask.data.progress}%` }} /></i></div>}
          {(generate.isError || generationTask.data?.status === 'FAILED') && <p className="quality-error">{generate.error?.message || generationTask.data?.error}</p>}
          {generationTask.data?.result && <section className="generation-result"><div><p className="eyebrow">GENERATED</p><h3>生成完成并已保存</h3><small>{generationTask.data.result.generation.model} · Prompt {generationTask.data.result.generation.promptVersion} · {generationTask.data.result.generation.outputTokens} tokens</small></div><pre>{generationTask.data.result.generation.content}</pre><div className="result-actions">{operation==='OUTLINE'&&<button className="primary" disabled={outlineToTree.isPending} onClick={()=>outlineToTree.mutate()}>一键转换为内容树</button>}<button className="secondary" onClick={() => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTab('documents') }}>查看项目文档</button></div>{outlineToTree.isError&&<p className="quality-error">{outlineToTree.error.message}</p>}</section>}
          {qualityTask.data && !['SUCCESS', 'FAILED', 'CANCELLED'].includes(qualityTask.data.status) && <div className="task-progress"><div><strong>{qualityTask.data.message}</strong><span>{qualityTask.data.progress}%</span></div><i><b style={{ width: `${qualityTask.data.progress}%` }} /></i></div>}
          {(qualityGenerate.isError || qualityTask.data?.status === 'FAILED') && <p className="quality-error">{qualityGenerate.error?.message || qualityTask.data?.error}</p>}
          {qualityTask.data?.result && <section className="generation-result"><div><p className="eyebrow">QUALITY GENERATED</p><h3>严格质量生成完成</h3><small>门禁 {qualityTask.data.result.workflow.validation.gate.status} · 得分 {qualityTask.data.result.workflow.validation.result.score} · 校验 {qualityTask.data.result.workflow.attempts} 轮 · 修复 {qualityTask.data.result.workflow.repairs.length} 次</small></div><pre>{qualityTask.data.result.workflow.content}</pre><button className="primary" onClick={() => { queryClient.invalidateQueries({ queryKey: ['documents', project.id] }); setTab('documents') }}>查看项目文档</button></section>}
        </> : tab === 'knowledge' ? <>
          <div className="workspace-title"><div><p className="eyebrow">KNOWLEDGE BASE</p><h2>知识来源</h2><p>录入资料时标记权威等级，生成和校验将优先引用可靠来源。</p></div></div>
          <div className="knowledge-grid"><form className="studio-form" onSubmit={(e) => { e.preventDefault(); createSource.mutate() }}><h3>录入知识</h3><input required value={sourceName} onChange={(e) => setSourceName(e.target.value)} placeholder="来源名称" /><select value={authority} onChange={(e) => setAuthority(e.target.value as KnowledgeSource['authority'])}><option value="OFFICIAL">官方资料</option><option value="VERIFIED">人工确认</option><option value="INTERNAL">内部资料</option><option value="REFERENCE">普通参考</option></select><textarea required rows={7} value={sourceContent} onChange={(e) => setSourceContent(e.target.value)} placeholder="粘贴知识内容，系统将按结构分块…" /><button className="primary" disabled={createSource.isPending}>解析并加入知识库</button></form>
          <div className="search-panel"><h3>检索测试</h3><form onSubmit={(e) => { e.preventDefault(); setSubmittedQuery(query.trim()) }}><input required value={query} onChange={(e) => setQuery(e.target.value)} placeholder="输入事实、术语或问题" /><button>检索</button></form>{search.data?.items.map((hit) => <article key={hit.chunk.id}><p>{hit.chunk.content}</p><small>{hit.source.name} · {hit.source.authority} · 匹配度 {Math.round(hit.score * 100)}%</small></article>)}{submittedQuery && search.data?.total === 0 && <p className="empty-line">没有找到支持证据</p>}</div></div>
          <div className="resource-list"><h3>来源资料 <span>{sources.data?.total ?? 0}</span></h3>{sources.data?.items.map((item) => <article key={item.id}><div><strong>{item.name}</strong><small>{item.authority} · {item.status}</small></div><span>{item.version || '无版本标记'}</span></article>)}</div>
          <KnowledgeExtras project={project} documents={documents.data?.items ?? []} />
        </> : tab === 'memory' ? <MemoryPanel project={project} documents={orderedDocuments}/> : tab === 'runs' ? <AIRunsPanel project={project} /> : tab === 'storyline' ? <StorylinePanel project={project}/> : <>
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
    <Dialog open={expandOpen} onOpenChange={setExpandOpen}><DialogContent className="dialog ai-expand-dialog"><div className="ai-expand-content">
      <DialogHeader className="content-type-dialog-head"><DialogTitle>AI 扩写</DialogTitle><DialogDescription>AI 会参考当前正文和项目记忆生成扩写结果，确认后才会写回编辑区。</DialogDescription></DialogHeader>
      <label>扩写要求<textarea rows={4} value={expandInstruction} onChange={event=>setExpandInstruction(event.target.value)} placeholder="例如：扩写到 1500 字，加强场景细节和人物情绪"/></label>
      <label>知识库检索词<Input value={expandQuery} onChange={event=>setExpandQuery(event.target.value)} placeholder="可选：检索项目知识库作为事实依据"/></label>
      {!modelStatus.data?.generationOperations?.includes('WRITE')&&<p className="quality-error">写作模型尚未配置，请先在“模型与校验”中启用模型并重启 API。</p>}
      {!expandTask.data?.result&&<button className="primary expand-run" disabled={!expandInstruction.trim()||expand.isPending||expandTask.data?.status==='RUNNING'||!modelStatus.data?.generationOperations?.includes('WRITE')} onClick={()=>{setExpandTaskId('');expand.mutate()}}><Sparkles size={15}/>{expand.isPending?'正在创建任务…':expandTask.data?.status==='RUNNING'?`正在扩写 ${expandTask.data.progress}%`:'开始扩写'}</button>}
      {expandTask.data&&!['SUCCESS','FAILED','CANCELLED'].includes(expandTask.data.status)&&<div className="task-progress"><div><strong>{expandTask.data.message}</strong><span>{expandTask.data.progress}%</span></div><i><b style={{width:`${expandTask.data.progress}%`}}/></i></div>}
      {(expand.isError||expandTask.data?.status==='FAILED')&&<p className="quality-error">{expand.error?.message||expandTask.data?.error}</p>}
      {expandTask.data?.result&&<div className="expand-preview"><div><strong>扩写预览</strong><small>{expandTask.data.result.generation.model} · {expandTask.data.result.generation.outputTokens} tokens</small></div><pre>{expandTask.data.result.generation.content}</pre><div className="dialog-actions"><button className="secondary" onClick={()=>{setExpandTaskId('');expand.mutate()}}>重新生成</button><button className="secondary" onClick={()=>{setContent(value=>value.trim()?`${value.trimEnd()}\n\n${expandTask.data!.result!.generation.content}`:expandTask.data!.result!.generation.content);setExpandOpen(false);toast.success('扩写内容已追加')}}>追加正文</button><button className="primary" onClick={()=>{setContent(expandTask.data!.result!.generation.content);setExpandOpen(false);toast.success('正文已替换')}}>替换正文</button></div></div>}
    </div></DialogContent></Dialog>
  </div>
}

function ChapterDocumentGenerator(props:{project:Project;modelReady:string[];onOpenWorkspace:()=>void}){
  return <LegacyChapterDocumentGenerator {...props}/>
}

async function ensureChapterDocuments(projectId:string,nodes:ContentNode[]){
  const [tree,documents]=await Promise.all([api.tree(projectId),api.documents(projectId)])
  const latestById=new Map(tree.items.map(node=>[node.id,node]))
  const usedDocumentIds=new Set(tree.items.flatMap(node=>node.documentId?[node.documentId]:[]))
  for(const original of nodes){
    const node=latestById.get(original.id)??original
    if(node.documentId)continue
    const reusable=documents.items.find(document=>document.title===node.title&&!usedDocumentIds.has(document.id))
    const documentId=reusable?.id??(await api.createDocument(projectId,{title:node.title,content:''})).document.id
    usedDocumentIds.add(documentId)
    await api.updateNode(node.id,{title:node.title,position:node.position,metadata:node.metadata,documentId})
  }
}

function LegacyChapterDocumentGenerator({project,modelReady,onOpenWorkspace}:{project:Project;modelReady:string[];onOpenWorkspace:()=>void}){
  const queryClient=useQueryClient()
  const initialDraft=useRef(loadChapterDraft(project)).current
  const draftProjectId=useRef(project.id)
  const[requirement,setRequirement]=useState(initialDraft.requirement)
  const[outline,setOutline]=useState(initialDraft.outline)
  const bookTitle=project.name
  const[knowledgeQuery,setKnowledgeQuery]=useState(initialDraft.knowledgeQuery)
  const[splitMode,setSplitMode]=useState<'LEAF'|'ALL'>(initialDraft.splitMode)
  const[outlineTaskId,setOutlineTaskId]=useState('')
  const[nodeIds,setNodeIds]=useState<string[]>(initialDraft.nodeIds)
  const[documentsReady,setDocumentsReady]=useState(initialDraft.documentsReady)
  const[assistOpen,setAssistOpen]=useState(false)
  const[assistPrompt,setAssistPrompt]=useState('')
  const[assistTaskId,setAssistTaskId]=useState('')
  const appliedTask=useRef('')
  const notifiedAssistTask=useRef('')
  useEffect(()=>{
    if(draftProjectId.current!==project.id)return
    try{localStorage.setItem(chapterDraftKey(project.id),JSON.stringify({requirement,outline,knowledgeQuery,splitMode,nodeIds,documentsReady,selectionVersion:3} satisfies ChapterGeneratorDraft))}catch{/* Browser storage may be unavailable in private mode. */}
  },[project.id,requirement,outline,knowledgeQuery,splitMode,nodeIds,documentsReady])
  useEffect(()=>{
    if(draftProjectId.current===project.id)return
    const draft=loadChapterDraft(project)
    setRequirement(draft.requirement);setOutline(draft.outline);setKnowledgeQuery(draft.knowledgeQuery);setSplitMode(draft.splitMode);setNodeIds(draft.nodeIds);setDocumentsReady(draft.documentsReady)
    setOutlineTaskId('');setAssistTaskId('');appliedTask.current='';draftProjectId.current=project.id
  },[project])
  const outlineTask=useQuery({queryKey:['chapter-outline-task',outlineTaskId],queryFn:()=>api.task<GenerationResult>(outlineTaskId),enabled:Boolean(outlineTaskId),refetchInterval:q=>['SUCCESS','FAILED','CANCELLED'].includes(q.state.data?.status??'')?false:1000})
  const assistTask=useQuery({queryKey:['chapter-brief-assist-task',assistTaskId],queryFn:()=>api.task<GenerationResult>(assistTaskId),enabled:Boolean(assistTaskId),refetchInterval:q=>['SUCCESS','FAILED','CANCELLED'].includes(q.state.data?.status??'')?false:1000})
  const assist=useMutation({mutationFn:()=>api.createGenerationTask(project.id,{operation:modelReady.includes('PLAN')?'PLAN':'OUTLINE',instruction:`根据用户的初步想法，撰写一份清晰、具体、可直接用于规划目录的创作简报。简报应包含题材与主题、目标读者、故事或内容范围、主要风格、预计篇幅和关键创作要求。只输出简报正文，不要解释。\n初步想法：${assistPrompt}`,title:`${bookTitle}创作简报`,documentId:'',knowledgeQuery,content:requirement,save:false}),onSuccess:task=>setAssistTaskId(task.id)})
  useEffect(()=>{
    if(!assistTaskId||notifiedAssistTask.current===assistTaskId)return
    if(assistTask.data?.status==='SUCCESS'){
      notifiedAssistTask.current=assistTaskId
      toast.success('AI 简报生成完成',{description:'点击“AI 助写”可查看并使用生成结果。'})
    }else if(assistTask.data?.status==='FAILED'||assistTask.data?.status==='CANCELLED'){
      notifiedAssistTask.current=assistTaskId
      toast.error(assistTask.data.status==='FAILED'?'AI 简报生成失败':'AI 简报任务已取消',{description:assistTask.data.error||undefined})
    }
  },[assistTask.data?.status,assistTask.data?.error,assistTaskId])
  const generateOutline=useMutation({mutationFn:()=>api.createGenerationTask(project.id,{operation:'OUTLINE',instruction:outline?`依据写作需求扩写并优化现有 Markdown 目录。只保留一级作品标题和二级真实章节标题，不要输出章节目标、核心要点、所需证据、预计篇幅等说明项。\n写作需求：${requirement}`:`为《${bookTitle}》生成 Markdown 章节目录。一级标题为作品名，二级标题为真实章节；不要输出三级标题或规划说明。\n写作需求：${requirement}`,title:`${bookTitle}目录`,documentId:'',knowledgeQuery,content:outline,save:false}),onSuccess:task=>{appliedTask.current='';setOutlineTaskId(task.id);setNodeIds([]);setDocumentsReady(false)}})
  useEffect(()=>{const generated=outlineTask.data?.result?.generation.content;if(generated&&appliedTask.current!==outlineTaskId){setOutline(normalizeGeneratedOutline(generated));appliedTask.current=outlineTaskId}},[outlineTask.data?.result,outlineTaskId])
  const confirm=useMutation({mutationFn:async()=>{let selected:ContentNode[];if(nodeIds.length){const current=await api.tree(project.id);const byId=new Map(current.items.map(node=>[node.id,node]));const selectedIds=new Set(nodeIds);const previous=current.items.filter(node=>selectedIds.has(node.id));if(splitMode==='ALL')selected=previous;else{const chapters=new Map<string,ContentNode>();for(const item of previous){let node:ContentNode|undefined=item;while(node&&Number(node.metadata?.outlineLevel)>2&&node.parentId)node=byId.get(node.parentId);if(node&&Number(node.metadata?.outlineLevel)===2)chapters.set(node.id,node)}selected=[...chapters.values()]}}else{const data=await api.importOutline(project.id,{content:outline,preview:false});const nodes=data.items as ContentNode[];const parents=new Set(nodes.map(item=>item.parentId).filter(Boolean));const chapters=nodes.filter(item=>Number(item.metadata?.outlineLevel)===2);selected=splitMode==='ALL'?nodes:chapters.length?chapters:nodes.filter(item=>!parents.has(item.id))}await ensureChapterDocuments(project.id,selected);return selected},onSuccess:selected=>{const confirmedIds=selected.map(item=>item.id);setNodeIds(confirmedIds);setDocumentsReady(true);try{localStorage.setItem(chapterDraftKey(project.id),JSON.stringify({requirement,outline,knowledgeQuery,splitMode,nodeIds:confirmedIds,documentsReady:true,selectionVersion:3} satisfies ChapterGeneratorDraft))}catch{/* Browser storage may be unavailable in private mode. */}queryClient.invalidateQueries({queryKey:['tree',project.id]});queryClient.invalidateQueries({queryKey:['documents',project.id]});toast.success(`已准备 ${selected.length} 篇章节文档，已有正文保持不变`);onOpenWorkspace()}})
  const running=outlineTask.data&&!['SUCCESS','FAILED','CANCELLED'].includes(outlineTask.data.status)
  const step=outline&&!running?3:running?2:1
  const outlineLines=outline.split('\n').filter(line=>/^#{1,6}\s+/.test(line.trim()))
  const chapterCount=outlineLines.filter(line=>/^##\s+/.test(line.trim())).length
  const canOutline=modelReady.includes('OUTLINE')
  const steps=[['填写需求','描述主题与目标'],['生成目录','AI 规划章节'],['确认目录','创建空白章节文档']]
  return <div className="chapter-generator">
    <div className="generator-steps" aria-label="章节生成进度">{steps.map(([label,description],index)=><div className={step===index+1?'active':step>index+1?'done':''} key={label} aria-current={step===index+1?'step':undefined}><b>{step>index+1?<Check size={14}/>:index+1}</b><span><strong>{label}</strong><small>{description}</small></span>{index<3&&<ChevronRight className="step-chevron"/>}</div>)}</div>
    <div className="generator-layout"><div className="generator-flow">
      <section className="generator-main-card generator-flow-step" data-step="1">
        <div className="generator-section-title"><span><Sparkles/></span><div><p>步骤 1 · 创作简报</p><h3>你想写一部怎样的作品？</h3></div><small>{requirement.length} / 2000</small></div>
        <label className="generator-requirement"><span className="sr-only">写作需求</span><span className="generator-requirement-wrap"><textarea maxLength={2000} rows={7} value={requirement} onChange={event=>setRequirement(event.target.value)} placeholder={'描述主题、目标读者、内容范围与预计篇数。\n例如：面向悬疑小说读者，规划一部 20 章的近未来故事；节奏紧凑，每章保留一个悬念。'}/><button type="button" className="brief-assist-trigger" disabled={!modelReady.includes('PLAN')&&!modelReady.includes('OUTLINE')} onClick={()=>{if(!assistTaskId)setAssistPrompt(requirement);setAssistOpen(true)}}><Sparkles size={14}/> {assistTask.data?.status==='RUNNING'?`AI 助写 ${assistTask.data.progress}%`:'AI 助写'}</button></span></label>
      </section>
      <section className="generator-main-card generator-flow-step" data-step="2">
        <div className="generator-section-title"><span><Sparkles/></span><div><p>步骤 2</p><h3>生成目录</h3></div></div>
        <button className="generator-ai-button" disabled={!requirement.trim()||generateOutline.isPending||Boolean(running)||!canOutline} onClick={()=>generateOutline.mutate()}><Sparkles size={17}/>{running?`AI 正在规划目录 · ${outlineTask.data?.progress??0}%`:outline?'重新优化目录':'生成目录初稿'}<ArrowRight size={16}/></button>
        {running&&<div className="generator-progress"><i><b style={{width:`${outlineTask.data?.progress??0}%`}}/></i><span>正在分析创作需求并设计章节结构</span></div>}
        {!canOutline&&<p className="generator-hint error"><CircleGauge size={13}/> 目录模型尚未配置。你仍可手动填写目录，或前往“模型与校验”完成配置。</p>}
        {(generateOutline.isError||outlineTask.data?.status==='FAILED')&&<p className="quality-error">{generateOutline.error?.message||outlineTask.data?.error}</p>}
      </section>
      <section className="generator-main-card generator-flow-step" data-step="3">
        <div className="generator-section-title outline-editor-head"><span><Sparkles/></span><div><p>步骤 3</p><h3>修改并确认目录</h3><small>{outline?'校对层级和标题，确认后锁定生成范围':'等待生成目录，也可以直接粘贴已有目录'}</small></div>{outline&&<div className="outline-stats"><span>{outlineLines.length} 个条目</span><span>{chapterCount} 个章节</span></div>}<button disabled={!outline} onClick={()=>{setOutline('');setNodeIds([]);setDocumentsReady(false)}}>清空</button></div>
        <textarea className="outline-markdown-editor" rows={14} value={outline} onChange={event=>{setOutline(event.target.value);setNodeIds([]);setDocumentsReady(false)}} placeholder={'# 第一部分\n## 第一章\n### 第一节'}/>
        <button className="secondary confirm-outline" disabled={confirm.isPending||(!nodeIds.length&&!outline.trim())} onClick={()=>documentsReady?onOpenWorkspace():confirm.mutate()}>{documentsReady?<><Check size={15}/> 已准备 {nodeIds.length} 篇章节文档 · 进入文档工作区</>:(confirm.isPending?'正在补齐章节文档…':nodeIds.length?'补齐文档并进入工作区':'确认目录并进入文档工作区')}</button>
        {confirm.isError&&<p className="quality-error">{confirm.error.message}</p>}
      </section>
    </div><aside className="generator-side">
      <section className="generator-settings"><div className="side-section-title"><Settings2/><div><h3>输出设置</h3><p>决定正文如何拆分与引用资料</p></div></div><label>作品名称<Input value={bookTitle} readOnly aria-readonly="true" title="作品名称与当前项目名称一致"/></label><label>文档拆分方式<Select value={splitMode} onChange={event=>{setSplitMode(event.target.value as 'LEAF'|'ALL');setNodeIds([]);setDocumentsReady(false)}}><option value="LEAF">每个二级章节一篇文档</option><option value="ALL">每个目录条目一篇文档</option></Select></label><label>知识库检索词<Input value={knowledgeQuery} onChange={event=>setKnowledgeQuery(event.target.value)} placeholder="可选：人物、世界观或参考资料"/></label></section>
    </aside></div>
    <Dialog open={assistOpen} onOpenChange={setAssistOpen}><DialogContent className="dialog ai-expand-dialog"><div className="ai-expand-content">
      <DialogHeader className="content-type-dialog-head"><DialogTitle>AI 助写创作简报</DialogTitle><DialogDescription>输入一句初步想法，AI 会补全题材、受众、风格、篇幅和创作要求，确认后才会写入需求框。</DialogDescription></DialogHeader>
      <label>你的初步想法<textarea rows={6} value={assistPrompt} onChange={event=>setAssistPrompt(event.target.value)} placeholder="例如：写一部发生在海底城市的悬疑小说，主角是一名失忆的工程师。"/></label>
      <button className="primary expand-run" disabled={!assistPrompt.trim()||assist.isPending||assistTask.data?.status==='RUNNING'} onClick={()=>{setAssistTaskId('');assist.mutate()}}><Sparkles size={15}/>{assist.isPending||assistTask.data?.status==='RUNNING'?'正在助写…':'生成创作简报'}</button>
      {(assist.isError||assistTask.data?.status==='FAILED')&&<p className="quality-error">{assist.error?.message||assistTask.data?.error}</p>}
      {assistTask.data?.result&&<div className="expand-preview"><div><strong>助写结果</strong><small>{assistTask.data.result.generation.model} · {assistTask.data.result.generation.outputTokens} tokens</small></div><pre>{assistTask.data.result.generation.content}</pre><div className="dialog-actions"><button className="secondary" onClick={()=>assist.mutate()}>重新生成</button><button className="secondary" onClick={()=>{setRequirement(value=>(value.trim()?`${value.trimEnd()}\n\n${assistTask.data!.result!.generation.content}`:assistTask.data!.result!.generation.content).slice(0,2000));setAssistOpen(false);toast.success('AI 简报已追加')}}>追加</button><button className="primary" onClick={()=>{setRequirement(assistTask.data!.result!.generation.content.slice(0,2000));setAssistOpen(false);toast.success('AI 简报已写入')}}>替换需求</button></div></div>}
    </div></DialogContent></Dialog>
  </div>
}

function OutlineImporter({project}:{project:Project}){
  const queryClient=useQueryClient();const[open,setOpen]=useState(false);const[content,setContent]=useState('');const[preview,setPreview]=useState<OutlineItem[]>([]);const run=useMutation({mutationFn:(previewMode:boolean)=>api.importOutline(project.id,{content,preview:previewMode}),onSuccess:(data,previewMode)=>{if(previewMode)setPreview(data.items as OutlineItem[]);else{setContent('');setPreview([]);setOpen(false);queryClient.invalidateQueries({queryKey:['tree',project.id]});toast.success(`已导入 ${data.total} 个内容节点`)}}})
  return <><button className="secondary outline-import-trigger" onClick={()=>setOpen(true)}><Upload size={14}/> 导入 Markdown 大纲</button><Dialog open={open} onOpenChange={setOpen}><DialogContent className="dialog outline-import-dialog"><section className="outline-importer"><DialogHeader className="content-type-dialog-head"><p className="eyebrow">OUTLINE IMPORT</p><DialogTitle>导入 Markdown 大纲</DialogTitle><DialogDescription>粘贴带 # 标题层级的大纲，系统会按项目类型转换为对应的内容节点。</DialogDescription></DialogHeader><textarea rows={9} value={content} onChange={event=>{setContent(event.target.value);setPreview([])}} placeholder={'# 第一卷\n## 第一章\n### 第一节'} /><div className="outline-actions"><button className="secondary" disabled={!content.trim()||run.isPending} onClick={()=>run.mutate(true)}>预览结构</button><button className="primary" disabled={!content.trim()||run.isPending} onClick={()=>run.mutate(false)}>导入内容树</button></div>{run.isError&&<p className="quality-error">{run.error.message}</p>}{preview.length>0&&<div className="outline-preview">{preview.map((item,index)=><p style={{paddingLeft:`${(item.level-1)*22}px`}} key={`${index}-${item.title}`}><b>{item.nodeType}</b><span>{item.title}</span></p>)}</div>}</section></DialogContent></Dialog></>
}

function StructurePanel({ project, documents }: { project: Project; documents: Document[] }) {
  const queryClient=useQueryClient();const[title,setTitle]=useState('');const[parentId,setParentId]=useState('');const[nodeType,setNodeType]=useState('SECTION');const[selected,setSelected]=useState<string[]>([]);const[instruction,setInstruction]=useState('');const[taskId,setTaskId]=useState('')
  const tree=useQuery({queryKey:['tree',project.id],queryFn:()=>api.tree(project.id)});const refresh=()=>queryClient.invalidateQueries({queryKey:['tree',project.id]});const create=useMutation({mutationFn:()=>api.createNode(project.id,{parentId:parentId||undefined,nodeType,title,position:(tree.data?.items.length??0)+1}),onSuccess:()=>{setTitle('');refresh()}});const update=useMutation({mutationFn:({id,title,position,documentId}:{id:string;title:string;position:number;documentId?:string})=>api.updateNode(id,{title,position,documentId}),onSuccess:refresh});const remove=useMutation({mutationFn:api.deleteNode,onSuccess:refresh});const batch=useMutation({mutationFn:()=>api.batchGenerate(project.id,{nodeIds:selected,instruction,knowledgeQuery:'',windowSize:2}),onSuccess:(task)=>setTaskId(task.id)});const active=useQuery({queryKey:['batch-task',taskId],queryFn:()=>api.task(taskId),enabled:Boolean(taskId),refetchInterval:(q)=>['SUCCESS','FAILED','CANCELLED'].includes(q.state.data?.status??'')?false:1000});useEffect(()=>{if(active.data?.status==='SUCCESS'){refresh();queryClient.invalidateQueries({queryKey:['documents',project.id]})}},[active.data?.status])
  const items=tree.data?.items??[];const children=new Map<string,typeof items>();items.forEach(node=>{const key=node.parentId??'';children.set(key,[...(children.get(key)??[]),node])});const rows:Array<{node:typeof items[number];depth:number}>=[];const walk=(parent:string,depth:number)=>{(children.get(parent)??[]).sort((a,b)=>a.position-b.position).forEach(node=>{rows.push({node,depth});walk(node.id,depth+1)})};walk('',0);items.filter(node=>!rows.some(row=>row.node.id===node.id)).forEach(node=>rows.push({node,depth:0}))
  return <><div className="workspace-title structure-title"><div><p className="eyebrow">CONTENT TREE</p><h2>内容结构</h2><p>按层级管理章节节点，绑定文档，并选择多个节点批量生成正文。</p></div><OutlineImporter project={project}/></div><form className="inline-form node-create" onSubmit={(e)=>{e.preventDefault();create.mutate()}}><select value={nodeType} onChange={(e)=>setNodeType(e.target.value)}><option value="VOLUME">卷</option><option value="CHAPTER">章</option><option value="SECTION">节</option></select><select value={parentId} onChange={(e)=>setParentId(e.target.value)}><option value="">顶层节点</option>{items.map(node=><option value={node.id} key={node.id}>{node.title}</option>)}</select><input required value={title} onChange={(e)=>setTitle(e.target.value)} placeholder="新节点标题"/><button className="primary">新增节点</button></form><div className="tree-list">{rows.map(({node,depth})=><article key={node.id} style={{paddingLeft:`${12+depth*28}px`}}><input type="checkbox" checked={selected.includes(node.id)} onChange={(e)=>setSelected(e.target.checked?[...selected,node.id]:selected.filter(id=>id!==node.id))}/><div><strong>{depth>0?'└ ':''}{node.title}</strong><small>{node.nodeType} · 层级 {depth+1} · 位置 {node.position}</small></div><select aria-label="绑定文档" value={node.documentId??''} onChange={(e)=>update.mutate({id:node.id,title:node.title,position:node.position,documentId:e.target.value||undefined})}><option value="">未绑定文档</option>{documents.map(doc=><option value={doc.id} key={doc.id}>{doc.title}</option>)}</select><button onClick={()=>remove.mutate(node.id)}>删除</button></article>)}</div><form className="studio-form batch-form" onSubmit={(e)=>{e.preventDefault();batch.mutate()}}><h3>批量生成选中节点</h3><textarea required rows={4} value={instruction} onChange={(e)=>setInstruction(e.target.value)} placeholder="统一的创作要求…"/><button className="primary" disabled={!selected.length||batch.isPending}>生成 {selected.length} 个节点</button></form>{active.data&&<div className="task-progress"><div><strong>{active.data.message}</strong><span>{active.data.progress}%</span></div><i><b style={{width:`${active.data.progress}%`}}/></i></div>}</>
}

function MemoryPanel({project,documents}:{project:Project;documents:Document[]}){
  const queryClient=useQueryClient()
  const[type,setType]=useState<MemoryEntry['type']>('CHARACTER');const[filter,setFilter]=useState('');const[name,setName]=useState('');const[summary,setSummary]=useState('')
  const[documentId,setDocumentId]=useState('');const[extractTaskId,setExtractTaskId]=useState('');const[selectedSuggestions,setSelectedSuggestions]=useState<number[]>([])
  const labels:Record<string,string>={CHARACTER:'人物 / 主体',PLACE:'地点 / 场景',TIMELINE:'时间线',PLOT:'剧情 / 主题线',FORESHADOW:'伏笔 / 待办'}
  const memories=useQuery({queryKey:['memories',project.id,filter],queryFn:()=>api.memories(project.id,filter)})
  const extractTask=useQuery({queryKey:['memory-extraction-task',extractTaskId],queryFn:()=>api.task<MemoryExtractionResult>(extractTaskId),enabled:Boolean(extractTaskId),refetchInterval:query=>['SUCCESS','FAILED','CANCELLED'].includes(query.state.data?.status??'')?false:700})
  const extract=useMutation({mutationFn:()=>api.extractMemories(project.id,documentId),onSuccess:task=>{setExtractTaskId(task.id);setSelectedSuggestions([])}})
  const suggestions=extractTask.data?.result?.memories??[]
  useEffect(()=>{if(extractTask.data?.status==='SUCCESS')setSelectedSuggestions(suggestions.map((_,index)=>index))},[extractTask.data?.status,extractTaskId])
  const approve=useMutation({mutationFn:async()=>{for(const index of selectedSuggestions){const item=suggestions[index] as MemorySuggestion;await api.createMemory(project.id,{type:item.type,name:item.name,summary:item.summary,status:'ACTIVE',attributes:item.attributes})}},onSuccess:()=>{queryClient.invalidateQueries({queryKey:['memories',project.id]});setExtractTaskId('');setSelectedSuggestions([]);toast.success('已将确认项加入长期记忆')}})
  const create=useMutation({mutationFn:()=>api.createMemory(project.id,{type,name,summary,status:'ACTIVE'}),onSuccess:()=>{setName('');setSummary('');queryClient.invalidateQueries({queryKey:['memories',project.id]})}})
  const remove=useMutation({mutationFn:api.deleteMemory,onSuccess:()=>queryClient.invalidateQueries({queryKey:['memories',project.id]})})
  return <><div className="workspace-title"><div><p className="eyebrow">STORY MEMORY</p><h2>长期记忆</h2><p>从已保存正文提取连续性信息，经确认后供后续章节生成使用。</p></div></div><section className="memory-extractor"><div><h3>从文档提取记忆</h3><p>AI 识别人物、地点、时间线、剧情和伏笔；确认前不会写入。</p></div><select value={documentId} onChange={event=>{setDocumentId(event.target.value);setExtractTaskId('')}}><option value="">选择已有正文</option>{documents.map(item=><option value={item.id} key={item.id}>{item.title}</option>)}</select><button className="primary" disabled={!documentId||extract.isPending||extractTask.data?.status==='RUNNING'} onClick={()=>extract.mutate()}>{extractTask.data?.status==='RUNNING'?`正在提取 ${extractTask.data.progress}%`:'AI 提取记忆'}</button>{(extract.isError||extractTask.data?.status==='FAILED')&&<p className="quality-error">{extract.error?.message||extractTask.data?.error}</p>}</section>{suggestions.length>0&&<section className="memory-review"><header><div><h3>审核提取结果</h3><small>已选择 {selectedSuggestions.length} / {suggestions.length}</small></div><button className="primary" disabled={!selectedSuggestions.length||approve.isPending} onClick={()=>approve.mutate()}>确认加入长期记忆</button></header>{suggestions.map((item,index)=><label key={`${item.type}-${item.name}-${index}`}><input type="checkbox" checked={selectedSuggestions.includes(index)} onChange={event=>setSelectedSuggestions(current=>event.target.checked?[...current,index]:current.filter(value=>value!==index))}/><b>{labels[item.type]}</b><span><strong>{item.name}</strong><small>{item.summary}</small></span></label>)}</section>}<div className="memory-layout"><details className="memory-manual"><summary>手动新增记忆</summary><form className="studio-form" onSubmit={event=>{event.preventDefault();create.mutate()}}><select value={type} onChange={event=>setType(event.target.value as MemoryEntry['type'])}>{Object.entries(labels).map(([value,label])=><option value={value} key={value}>{label}</option>)}</select><input required value={name} onChange={event=>setName(event.target.value)} placeholder="名称或事件标题"/><textarea rows={5} value={summary} onChange={event=>setSummary(event.target.value)} placeholder="描述状态、关系、发生时间、约束或需要兑现的内容…"/><button className="primary" disabled={create.isPending}>保存记忆</button></form></details><section><div className="memory-filter"><button className={!filter?'selected':''} onClick={()=>setFilter('')}>全部</button>{Object.entries(labels).map(([value,label])=><button className={filter===value?'selected':''} onClick={()=>setFilter(value)} key={value}>{label}</button>)}</div><div className="memory-list">{memories.data?.items.map(item=><article key={item.id}><b>{labels[item.type]}</b><div><strong>{item.name}</strong><p>{item.summary||'暂无详细描述'}</p><small>{item.status} · {new Date(item.updatedAt).toLocaleString('zh-CN')}</small></div><button onClick={()=>remove.mutate(item.id)}>删除</button></article>)}{memories.data?.total===0&&<p className="empty-line">暂无长期记忆，请先从正文提取。</p>}</div></section></div></>
}

function AIRunsPanel({project}:{project:Project}){const runs=useQuery({queryKey:['ai-runs',project.id],queryFn:()=>api.aiRuns(project.id),refetchInterval:5000});return <><div className="workspace-title"><div><p className="eyebrow">OBSERVABILITY</p><h2>AI 运行记录</h2><p>查看每次模型调用的角色、模型、Token、耗时和状态。</p></div></div><section className="metrics run-metrics"><article><span>调用次数</span><strong>{runs.data?.total??0}</strong><small>最近 100 条</small></article><article><span>Token 总量</span><strong>{(runs.data?.stats.inputTokens??0)+(runs.data?.stats.outputTokens??0)}</strong><small>输入 {runs.data?.stats.inputTokens??0} · 输出 {runs.data?.stats.outputTokens??0}</small></article><article><span>累计耗时</span><strong>{Math.round((runs.data?.stats.latencyMs??0)/1000)}s</strong><small>模型请求耗时</small></article></section><div className="run-table">{runs.data?.items.map(run=><article key={run.id}><span className={`task-state ${run.status.toLowerCase()}`}>{run.status}</span><div><strong>{run.role} · {run.model}</strong><small>{run.provider} · Prompt {run.promptVersion||'—'} · {new Date(run.createdAt).toLocaleString('zh-CN')}</small></div><span>{run.inputTokens} / {run.outputTokens} tokens</span><b>{run.latencyMs} ms</b></article>)}{runs.data?.total===0&&<p className="empty-line">暂无模型调用记录。</p>}</div></>}

function StorylinePanel({project}:{project:Project}){
  const queryClient=useQueryClient()
  const[name,setName]=useState('');const[summary,setSummary]=useState('');const[stage,setStage]=useState('SETUP')
  const stageLabels:Record<string,string>={SETUP:'铺垫',DEVELOPMENT:'发展',CLIMAX:'高潮',RESOLUTION:'收束'}
  const storylines=useQuery({queryKey:['memories',project.id,'PLOT'],queryFn:()=>api.memories(project.id,'PLOT')})
  const create=useMutation({mutationFn:()=>api.createMemory(project.id,{type:'PLOT',name:name.trim(),summary:summary.trim(),status:'ACTIVE',attributes:{kind:'STORYLINE',stage}}),onSuccess:()=>{setName('');setSummary('');setStage('SETUP');queryClient.invalidateQueries({queryKey:['memories',project.id]});toast.success('故事线已添加')}})
  const remove=useMutation({mutationFn:api.deleteMemory,onSuccess:()=>queryClient.invalidateQueries({queryKey:['memories',project.id]})})
  return <><div className="workspace-title storyline-title"><div><p className="eyebrow">STORYLINE</p><h2>故事线</h2><p>梳理主线、支线与人物弧光，让每条叙事线都有清晰目标和推进阶段。</p></div><span>{storylines.data?.total??0} 条故事线</span></div><div className="storyline-layout"><form className="studio-form storyline-form" onSubmit={event=>{event.preventDefault();create.mutate()}}><div><p className="eyebrow">NEW THREAD</p><h3>添加故事线</h3><small>记录一条贯穿多个章节的叙事目标。</small></div><label>故事线名称<input required maxLength={120} value={name} onChange={event=>setName(event.target.value)} placeholder="例如：复制人追寻身份真相"/></label><label>当前阶段<select value={stage} onChange={event=>setStage(event.target.value)}>{Object.entries(stageLabels).map(([value,label])=><option value={value} key={value}>{label}</option>)}</select></label><label>目标与关键冲突<textarea required rows={6} maxLength={1000} value={summary} onChange={event=>setSummary(event.target.value)} placeholder="描述起点、核心冲突、关键转折与预期落点…"/></label>{create.isError&&<p className="quality-error">{create.error.message}</p>}<button className="primary" disabled={create.isPending||!name.trim()||!summary.trim()}><Plus size={15}/>{create.isPending?'正在添加…':'添加故事线'}</button></form><section className="storyline-board">{storylines.isLoading&&<p className="empty-line">正在加载故事线…</p>}{storylines.data?.items.map((item,index)=>{const itemStage=String(item.attributes?.stage??'DEVELOPMENT');return <article className="storyline-card" key={item.id}><div className="storyline-rail"><i/><span>{String(index+1).padStart(2,'0')}</span></div><div><header><span className={`storyline-stage ${itemStage.toLowerCase()}`}>{stageLabels[itemStage]??'发展'}</span><small>{new Date(item.updatedAt).toLocaleDateString('zh-CN')}</small></header><h3>{item.name}</h3><p>{item.summary||'暂无故事线说明'}</p></div><button type="button" aria-label={`删除故事线 ${item.name}`} title="删除故事线" disabled={remove.isPending} onClick={()=>remove.mutate(item.id)}><Trash2 size={15}/></button></article>})}{storylines.data?.total===0&&<div className="storyline-empty"><Sparkles/><strong>还没有故事线</strong><span>从左侧添加主线、支线或人物弧光。</span></div>}</section></div></>
}

function QualityHistory({project}:{project:Project}){const history=useQuery({queryKey:['quality-history',project.id],queryFn:()=>api.qualityResults(project.id),refetchInterval:5000});return <section className="quality-history"><h3>历史质量结果 <span>{history.data?.total??0}</span></h3>{history.data?.items.map(item=><article key={item.id}><b className={`gate-badge ${item.gateStatus.toLowerCase()}`}>{item.gateStatus}</b><div><strong>{item.score} 分 · {item.verdict}</strong><small>{item.documentId?'已关联文档':'独立文本'} · {new Date(item.createdAt).toLocaleString('zh-CN')}</small></div><span>{item.result.result.issues.length} 个问题</span></article>)}{history.data?.total===0&&<p className="empty-line">暂无持久化校验结果。</p>}</section>}

function KnowledgeExtras({project,documents}:{project:Project;documents:Document[]}){
  const queryClient=useQueryClient();const[file,setFile]=useState<File|null>(null);const[documentId,setDocumentId]=useState('');const facts=useQuery({queryKey:['facts',project.id],queryFn:()=>api.facts(project.id)});const upload=useMutation({mutationFn:()=>api.uploadKnowledge(project.id,file!,'REFERENCE'),onSuccess:()=>queryClient.invalidateQueries({queryKey:['sources',project.id]})});const extract=useMutation({mutationFn:()=>api.extractFacts(project.id,documentId)});const update=useMutation({mutationFn:({id,status}:{id:string;status:string})=>api.updateFact(id,status),onSuccess:()=>queryClient.invalidateQueries({queryKey:['facts',project.id]})})
  return <section className="knowledge-extras"><form className="inline-form" onSubmit={(e)=>{e.preventDefault();upload.mutate()}}><input required type="file" accept=".txt,.md,.markdown,.json,.csv,.html,.htm" onChange={(e)=>setFile(e.target.files?.[0]??null)}/><button className="primary" disabled={!file||upload.isPending}>上传知识文件</button></form><form className="inline-form" onSubmit={(e)=>{e.preventDefault();extract.mutate()}}><select required value={documentId} onChange={(e)=>setDocumentId(e.target.value)}><option value="">选择文档抽取事实</option>{documents.map(doc=><option value={doc.id} key={doc.id}>{doc.title}</option>)}</select><button className="primary" disabled={!documentId||extract.isPending}>抽取事实</button></form><div className="fact-list"><h3>结构化事实 <span>{facts.data?.total??0}</span></h3>{facts.data?.items.map(fact=><article key={fact.id}><div><strong>{fact.subject} · {fact.predicate} · {fact.object}</strong><small>{fact.status} · 置信度 {Math.round(fact.confidence*100)}%</small></div>{fact.status==='PROPOSED'&&<span><button onClick={()=>update.mutate({id:fact.id,status:'CONFIRMED'})}>确认</button><button onClick={()=>update.mutate({id:fact.id,status:'REJECTED'})}>拒绝</button></span>}</article>)}</div></section>
}

function DocumentEditor({ document, onBack, embedded=false }: { document: Document; onBack: () => void; embedded?:boolean }) {
  const queryClient = useQueryClient()
  const [content, setContent] = useState('')
  const [baseVersionId, setBaseVersionId] = useState('')
  const [savedContent, setSavedContent] = useState('')
  const [autoSave, setAutoSave] = useState(false)
  const [compareFrom, setCompareFrom] = useState('')
  const [editorMode,setEditorMode]=useState<'edit'|'preview'>('edit')
  const [versionsCollapsed,setVersionsCollapsed]=useState(true)
  const [selection,setSelection]=useState({start:0,end:0})
  const copilotRange=useRef({start:0,end:0})
  const [copilotTaskId,setCopilotTaskId]=useState('')
  const [appliedTaskId,setAppliedTaskId]=useState('')
  const [draftTaskId,setDraftTaskId]=useState('')
  const appliedDraftTask=useRef('')
  const versions = useQuery({ queryKey: ['versions', document.id], queryFn: () => api.versions(document.id) })
  const diff = useQuery({queryKey:['document-diff',document.id,compareFrom,versions.data?.items[0]?.id],queryFn:()=>api.diffVersions(document.id,compareFrom,versions.data!.items[0].id),enabled:Boolean(compareFrom&&versions.data?.items[0]?.id&&compareFrom!==versions.data?.items[0]?.id)})
  const copilotTask=useQuery({queryKey:['copilot-task',copilotTaskId],queryFn:()=>api.task<GenerationResult>(copilotTaskId),enabled:Boolean(copilotTaskId),refetchInterval:query=>['SUCCESS','FAILED','CANCELLED'].includes(query.state.data?.status??'')?false:700})
  const copilot=useMutation({mutationFn:(action:string)=>{copilotRange.current=selection.end>selection.start?selection:{start:0,end:content.length};const selected=content.slice(copilotRange.current.start,copilotRange.current.end);return api.createGenerationTask(document.projectId,{operation:'POLISH',instruction:`执行文本${action}。只输出处理后的文本，不要解释，不要添加代码围栏。`,title:'',documentId:'',knowledgeQuery:'',content:selected,save:false})},onSuccess:task=>setCopilotTaskId(task.id)})
  const draftTask=useQuery({queryKey:['document-draft-task',draftTaskId],queryFn:()=>api.task<GenerationResult>(draftTaskId),enabled:Boolean(draftTaskId),refetchInterval:query=>['SUCCESS','FAILED','CANCELLED'].includes(query.state.data?.status??'')?false:700})
  const generateDraft=useMutation({mutationFn:()=>api.createGenerationTask(document.projectId,{operation:'WRITE',instruction:`根据章节标题《${document.title}》生成完整正文。正文应结构完整、内容连贯，只输出 Markdown 正文，不要解释。`,title:document.title,documentId:document.id,knowledgeQuery:'',content:'',save:true}),onSuccess:task=>setDraftTaskId(task.id)})
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
  useEffect(()=>{const result=draftTask.data?.result;if(!result||draftTask.data?.status!=='SUCCESS'||appliedDraftTask.current===draftTaskId)return;setContent(result.generation.content);setSavedContent(result.generation.content);if(result.version)setBaseVersionId(result.version.id);appliedDraftTask.current=draftTaskId;queryClient.invalidateQueries({queryKey:['versions',document.id]});queryClient.invalidateQueries({queryKey:['documents',document.projectId]});toast.success('正文已生成并保存为新版本')},[draftTask.data?.status,draftTask.data?.result,draftTaskId,document.id,document.projectId,queryClient])

  function reloadLatest() {
    if (!latest) return
    setContent(latest.content)
    setSavedContent(latest.content)
    setBaseVersionId(latest.id)
    save.reset()
  }

  return <section className={embedded?'document-editor embedded':'document-editor'}>
    <div className="editor-head">{!embedded&&<button onClick={onBack}><ArrowLeft size={14}/> 文档列表</button>}<div><p className="eyebrow">MARKDOWN DOCUMENT</p><h2>{document.title}</h2></div><label><input type="checkbox" checked={autoSave} onChange={(event) => setAutoSave(event.target.checked)} /> 3 秒自动保存</label><div className="markdown-mode-switch editor-mode-tabs" role="group" aria-label="Markdown 显示模式"><button className={editorMode==='edit'?'active':''} onClick={()=>setEditorMode('edit')}>编辑</button><button className={editorMode==='preview'?'active':''} onClick={()=>setEditorMode('preview')}>预览</button></div><button className="primary" disabled={!dirty || save.isPending} onClick={() => save.mutate()}>{save.isPending ? '保存中…' : dirty ? '保存新版本' : '已保存'}</button></div>
    {save.isError && <div className="conflict-banner"><span>{save.error.message}</span><button onClick={reloadLatest}>载入服务器最新版本</button></div>}
    <div className="copilot-bar"><strong>AI Copilot</strong><button className="primary" disabled={generateDraft.isPending||draftTask.data?.status==='RUNNING'} onClick={()=>generateDraft.mutate()}><Sparkles size={13}/>{generateDraft.isPending||draftTask.data?.status==='RUNNING'?`正在生成正文 ${draftTask.data?.progress??0}%`:document.versionCount>1?'重新生成单篇':'生成单篇正文'}</button><span>{selection.end>selection.start?`已选择 ${selection.end-selection.start} 字符`:'未选择时处理全文'}</span>{[{action:'润色',tip:'优化语言、节奏和表达，不改变原意'},{action:'扩写',tip:'补充细节、描写和论述'},{action:'缩写',tip:'压缩内容并保留核心信息'},{action:'改写',tip:'用不同表达方式重写内容'},{action:'续写',tip:'依据上下文继续创作后续内容'},{action:'总结',tip:'提炼内容的主要观点和情节'},{action:'分析',tip:'分析结构、逻辑、风格和潜在问题'},{action:'检查',tip:'检查错字、语病、逻辑与一致性'}].map(({action,tip})=><button title={`${tip}；${selection.end>selection.start?'处理当前选区':'处理全文'}`} disabled={copilot.isPending||copilotTask.data?.status==='RUNNING'} onClick={()=>copilot.mutate(action)} key={action}>{action}</button>)}</div>
    {(generateDraft.isError||draftTask.data?.status==='FAILED')&&<p className="quality-error">{generateDraft.error?.message||draftTask.data?.error}</p>}
    {(copilot.isPending||copilotTask.data?.status==='RUNNING')&&<div className="copilot-status">AI 正在处理选中文本…</div>}{(copilot.isError||copilotTask.data?.status==='FAILED')&&<p className="quality-error">{copilot.error?.message||copilotTask.data?.error}</p>}
    <div className={versionsCollapsed?'editor-layout versions-collapsed':'editor-layout'}><div className="markdown-pane">{editorMode==='edit'?<MarkdownEditor value={content} onChange={setContent} onSelectionChange={(start,end)=>setSelection({start,end})}/>:<MarkdownPreview value={content}/>}<footer><span>{content.length} 字符</span><span>{editorMode==='preview'?'Markdown 预览':dirty ? '有未保存修改' : `当前版本 v${latest?.versionNumber ?? '—'}`}</span></footer></div><aside className="version-panel"><h3>版本历史 <span>{versions.data?.total ?? 0}</span></h3>{versions.data?.items.map((version) => <article className={version.id === baseVersionId ? 'current' : ''} key={version.id}><button onClick={() => { setContent(version.content); setBaseVersionId(latest?.id ?? version.id) }}><strong>v{version.versionNumber}</strong><span>{version.reason}</span><small>{new Date(version.createdAt).toLocaleString('zh-CN')}</small></button>{version.id !== latest?.id && <div className="version-actions"><button className="restore" onClick={() => setCompareFrom(version.id)}>与最新版比较</button><button className="restore" disabled={restore.isPending} onClick={() => restore.mutate(version.id)}>恢复</button></div>}</article>)}</aside><button className="version-panel-toggle" title={versionsCollapsed?'展开版本历史':'折叠版本历史'} aria-label={versionsCollapsed?'展开版本历史':'折叠版本历史'} onClick={()=>setVersionsCollapsed(value=>!value)}>{versionsCollapsed?<ChevronLeft/>:<ChevronRight/>}</button></div>
    <Dialog open={Boolean(compareFrom)} onOpenChange={open=>{if(!open)setCompareFrom('')}}><DialogContent className="dialog diff-dialog"><section className="diff-panel"><header><div><h3>版本差异</h3><small>{diff.data?`新增 ${diff.data.added} 行 · 删除 ${diff.data.deleted} 行`:'正在加载差异…'}</small></div></header>{diff.isError&&<p className="quality-error">{diff.error.message}</p>}{diff.data&&<pre>{diff.data.lines.map((line,index)=><span className={line.type.toLowerCase()} key={`${index}-${line.oldLine}-${line.newLine}`}><i>{line.oldLine??' '}</i><i>{line.newLine??' '}</i><b>{line.type==='ADDED'?'+':line.type==='DELETED'?'-':' '}</b>{line.content||' '}</span>)}</pre>}</section></DialogContent></Dialog>
  </section>
}
