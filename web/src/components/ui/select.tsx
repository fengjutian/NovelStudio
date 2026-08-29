import {cn} from '@/lib/utils'
export function Select({className,...props}:React.ComponentProps<'select'>){return <select data-slot="select" className={cn('h-9 w-full rounded-md border border-input bg-white px-3 text-sm outline-none focus:ring-2 focus:ring-ring/30',className)} {...props}/>}
