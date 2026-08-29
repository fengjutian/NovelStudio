import {cn} from '@/lib/utils'
export function Badge({className,...props}:React.ComponentProps<'span'>){return <span data-slot="badge" className={cn('inline-flex items-center rounded-full border border-border px-2 py-0.5 text-xs font-medium',className)} {...props}/>}
