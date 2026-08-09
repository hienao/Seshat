import { Badge } from '@appica/ui-react/badge'
import { ArrowLeft, Bell, FileText } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { EventCard } from '@/components/webhook/event-card'
import { errorMessage } from '@/lib/error-message'
import { formatDateTime } from '@/i18n/format'

export function EventDetailPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const event = useQuery({ queryKey: ['webhooks', 'event', id], queryFn: () => api.event(id), enabled: Number.isFinite(id) && id > 0 })
  const notificationStatus = useQuery({ queryKey: ['notifications', 'event', id], queryFn: () => api.eventNotificationStatus(id), enabled: Number.isFinite(id) && id > 0 })
  if (event.isPending) return <div className="grid min-h-[50vh] place-items-center text-sm text-neutral-500">{t('events.loading')}</div>
  if (event.error || !event.data) return <div className="mx-auto max-w-4xl px-4 py-10"><ErrorState message={errorMessage(event.error, t('events.missing'))} /></div>
  const item = event.data
  return (
    <div className="mx-auto max-w-5xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Webhook Event" title={item.title || t('events.defaultTitle')} description={item.summary || t('events.defaultDescription')} action={<AppButton render={<Link to="/events" />} size="sm" variant="outline"><ArrowLeft size={16} />{t('events.backToStream')}</AppButton>} />
      <Panel title={t('events.information')} icon={<FileText size={20} />}>
        <dl className="grid gap-4 sm:grid-cols-4">
          <div><dt className="text-xs text-neutral-500">App</dt><dd className="mt-1 font-semibold">{item.app_code}</dd></div>
          <div><dt className="text-xs text-neutral-500">{t('events.sourceType')}</dt><dd className="mt-1 font-mono text-sm">{item.source_event_type || 'unknown'}</dd></div>
          <div><dt className="text-xs text-neutral-500">{t('events.displayType')}</dt><dd className="mt-1"><Badge variant={item.is_fallback ? 'outline' : 'soft'}>{item.is_fallback ? t('events.defaultType') : t(`events.types.${item.display_event_type}`, { defaultValue: item.display_event_type })}</Badge></dd></div>
          <div><dt className="text-xs text-neutral-500">{t('events.receivedAt')}</dt><dd className="mt-1 text-sm">{formatDateTime(item.received_at)}</dd></div>
        </dl>
      </Panel>
      <Panel title={t('events.notificationStatus')} description={t('events.notificationStatusDescription')} icon={<Bell size={20} />}>
        {notificationStatus.isPending ? <p className="text-sm text-neutral-500">{t('events.notificationStatusLoading')}</p> : notificationStatus.error ? <ErrorState message={errorMessage(notificationStatus.error, t('events.notificationStatusUnavailable'))} onRetry={() => void notificationStatus.refetch()} /> : notificationStatus.data && <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div><Badge variant={notificationStatus.data.state === 'succeeded' ? 'success' : notificationStatus.data.state === 'failed' ? 'error' : 'outline'}>{notificationStateLabel(notificationStatus.data.state, t)}</Badge><p className="mt-2 text-sm text-neutral-500">{notificationStatus.data.reason || notificationStatus.data.delivery?.last_error || t('events.notificationQueued')}</p></div>{notificationStatus.data.delivery && <div className="text-xs text-neutral-500">{t('events.attemptSummary', { count: notificationStatus.data.delivery.attempt_count })}{notificationStatus.data.delivery.sent_at ? ` · ${formatDateTime(notificationStatus.data.delivery.sent_at)}` : ''}</div>}</div>}
      </Panel>
      <Panel title={t('events.standardized')} description={t('events.standardizedDescription')}>
        <EventCard event={item} detail />
      </Panel>
      <Panel title={t('events.rawMessage')} description={t('events.rawDescription')}>
        <pre className="max-h-[620px] overflow-auto rounded-xl bg-neutral-950 p-5 text-sm leading-6 text-neutral-200">{item.raw_body || t('events.emptyBody')}</pre>
      </Panel>
    </div>
  )
}

function notificationStateLabel(state: string, t: TFunction) {
  return t(`events.notificationStates.${state}`, { defaultValue: state })
}
