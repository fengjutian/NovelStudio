import {cn} from '@/lib/utils'
export function Card({className,...props}:React.ComponentProps<'article'>){return <article data-slot="card" className={cn('rounded-xl border border-border bg-card text-card-foreground shadow-sm',className)} {...props}/>}
