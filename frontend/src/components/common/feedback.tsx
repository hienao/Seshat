import { Alert, AlertDescription, AlertIcon, AlertTitle } from '@appica/ui-react/alert'
import { AlertCircle, CircleCheck, Refresh } from '@appica/icons-react'
import { AppButton } from './app-button'
import { useTranslation } from 'react-i18next'

export function Message({ variant, title, description }: {
  variant: 'error' | 'success' | 'info' | 'warning'
  title: string
  description?: string
}) {
  const Icon = variant === 'success' ? CircleCheck : AlertCircle
  return (
    <Alert variant={variant}>
      <AlertIcon><Icon size={19} /></AlertIcon>
      <AlertTitle>{title}</AlertTitle>
      {description && <AlertDescription>{description}</AlertDescription>}
    </Alert>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="grid place-items-center py-10 text-center">
      <AlertCircle size={34} className="text-red-500" />
      <p className="mt-3 font-medium">{t('common.feedback.loadFailed')}</p>
      <p className="mt-1 text-sm text-neutral-500">{message}</p>
      {onRetry && <AppButton className="mt-4" variant="outline" size="sm" onClick={onRetry}><Refresh size={16} />{t('common.actions.reload')}</AppButton>}
    </div>
  )
}

export function EmptyState({ title, description }: { title?: string; description?: string }) {
  const { t } = useTranslation()
  return (
    <div className="grid place-items-center py-12 text-center">
      <div className="grid size-12 place-items-center rounded-full bg-neutral-100 text-neutral-400 dark:bg-neutral-800"><span className="text-xl">0</span></div>
      <p className="mt-3 font-medium">{title ?? t('common.feedback.noData')}</p>
      {description && <p className="mt-1 text-sm text-neutral-500">{description}</p>}
    </div>
  )
}
