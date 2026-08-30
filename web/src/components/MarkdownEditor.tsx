import {useEffect,useRef} from 'react'
import {EditorState} from '@codemirror/state'
import {EditorView,keymap,lineNumbers,highlightActiveLine,drawSelection} from '@codemirror/view'
import {defaultKeymap,history,historyKeymap} from '@codemirror/commands'
import {bracketMatching,syntaxHighlighting,defaultHighlightStyle} from '@codemirror/language'
import {markdown} from '@codemirror/lang-markdown'

export function MarkdownEditor({value,onChange,onSelectionChange}:{value:string;onChange:(value:string)=>void;onSelectionChange?:(start:number,end:number)=>void}){
  const host=useRef<HTMLDivElement|null>(null);const view=useRef<EditorView|null>(null);const changing=useRef(false)
  useEffect(()=>{if(!host.current)return;view.current=new EditorView({parent:host.current,state:EditorState.create({doc:value,extensions:[lineNumbers(),history(),drawSelection(),highlightActiveLine(),bracketMatching(),syntaxHighlighting(defaultHighlightStyle,{fallback:true}),markdown(),keymap.of([...defaultKeymap,...historyKeymap]),EditorView.lineWrapping,EditorView.theme({'&':{height:'100%',fontSize:'14px'},'.cm-scroller':{fontFamily:"'Noto Serif SC',serif",lineHeight:'1.72'},'.cm-content':{padding:'18px 22px'}}),EditorView.updateListener.of(update=>{if(update.docChanged){changing.current=true;onChange(update.state.doc.toString());changing.current=false}if(update.selectionSet){const range=update.state.selection.main;onSelectionChange?.(range.from,range.to)}})]})});return()=>{view.current?.destroy();view.current=null}},[])
  useEffect(()=>{const editor=view.current;if(!editor||changing.current||editor.state.doc.toString()===value)return;editor.dispatch({changes:{from:0,to:editor.state.doc.length,insert:value}})},[value])
  return <div className="codemirror-host" ref={host}/>
}
