import * as DialogPrimitive from '@radix-ui/react-dialog'
import {X} from 'lucide-react'
import {cn} from '@/lib/utils'
export const Dialog=DialogPrimitive.Root;export const DialogTrigger=DialogPrimitive.Trigger;export const DialogClose=DialogPrimitive.Close
export function DialogContent({className,children,...props}:React.ComponentProps<typeof DialogPrimitive.Content>){return <DialogPrimitive.Portal><DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/45"/><DialogPrimitive.Content className={cn('fixed left-1/2 top-1/2 z-50 w-[min(92vw,520px)] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-card p-6 shadow-xl',className)} {...props}>{children}<DialogPrimitive.Close className="absolute right-4 top-4"><X className="size-4"/></DialogPrimitive.Close></DialogPrimitive.Content></DialogPrimitive.Portal>}
export const DialogHeader=({className,...props}:React.ComponentProps<'div'>)=><div className={cn('mb-4 grid gap-1',className)} {...props}/>;export const DialogTitle=DialogPrimitive.Title;export const DialogDescription=DialogPrimitive.Description
