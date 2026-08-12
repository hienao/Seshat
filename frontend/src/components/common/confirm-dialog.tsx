import {
  AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from '@appica/ui-react/alert-dialog'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { AppButton } from './app-button'

export function ConfirmDialog({ trigger, title, description, confirmLabel, destructive, busy, onConfirm }: {
  trigger: ReactNode
  title: string
  description: string
  confirmLabel?: string
  destructive?: boolean
  busy?: boolean
  onConfirm: () => void
}) {
  const { t } = useTranslation()
  return (
    <AlertDialog>
      <AlertDialogTrigger render={trigger as React.ReactElement} />
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose render={<AppButton variant="ghost">{t('common.actions.cancel')}</AppButton>} />
          <AlertDialogClose render={<AppButton variant={destructive ? 'destructive' : 'primary'} disabled={busy} onClick={onConfirm}>{busy ? t('common.actions.processing') : confirmLabel ?? t('common.actions.confirm')}</AppButton>} />
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
