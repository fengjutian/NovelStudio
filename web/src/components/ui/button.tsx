import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'
const variants=cva('inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',{variants:{variant:{default:'bg-primary text-primary-foreground hover:opacity-90',secondary:'border border-border bg-card hover:bg-black/5',ghost:'hover:bg-black/5',destructive:'bg-red-700 text-white'},size:{default:'h-9 px-4 py-2',sm:'h-8 px-3 text-xs',icon:'size-9'}},defaultVariants:{variant:'default',size:'default'}})
export function Button({className,variant,size,asChild=false,...props}:React.ComponentProps<'button'>&VariantProps<typeof variants>&{asChild?:boolean}){const Comp=asChild?Slot:'button';return <Comp data-slot="button" className={cn(variants({variant,size}),className)} {...props}/>}
