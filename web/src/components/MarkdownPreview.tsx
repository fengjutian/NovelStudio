import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

export function MarkdownPreview({value}:{value:string}){
  return <div className="markdown-preview">
    {value.trim()?<ReactMarkdown remarkPlugins={[remarkGfm]}>{value}</ReactMarkdown>:<p className="markdown-preview-empty">暂无可预览内容</p>}
  </div>
}
