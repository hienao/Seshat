import {
  AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from '@appica/ui-react/alert-dialog'
import type { ReactNode } from 'react'
import { AppButton } from './app-button'

export function ConfirmDialog({ trigger, title, description, confirmLabel = '确认', destructive, busy, onConfirm }: {
  trigger: ReactNode
  title: string
  description: string
  confirmLabel?: string
  destructive?: boolean
  busy?: boolean
  onConfirm: () => void
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger render={trigger as React.ReactElement} />
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose render={<AppButton variant="ghost">取消</AppButton>} />
          <AlertDialogClose render={<AppButton variant={destructive ? 'destructive' : 'primary'} disabled={busy} onClick={onConfirm}>{busy ? '处理中…' : confirmLabel}</AppButton>} />
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
