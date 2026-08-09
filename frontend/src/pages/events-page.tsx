import { Bell, Refresh } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useEffect, useMemo, useState } from 'react'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { EventCard } from '@/components/webhook/event-card'
import { errorMessage } from '@/lib/error-message'

export function EventsPage() {
  const pageSize = 20
  const [integrationID, setIntegrationID] = useState('')
  const [eventType, setEventType] = useState('')
  const [page, setPage] = useState(0)
  const apps = useQuery({ queryKey: ['webhooks', 'apps'], queryFn: api.webhookApps })
  const integrations = useQuery({ queryKey: ['webhooks', 'integrations'], queryFn: api.integrations })
  const selectedIntegration = integrations.data?.find((item) => item.id === Number(integrationID))
  const selectedApp = apps.data?.find((app) => app.code === selectedIntegration?.app_code)
  const eventTypes = useMemo(() => selectedApp?.event_types ?? [], [selectedApp])
  const events = useQuery({
    queryKey: ['webhooks', 'events', integrationID, eventType, page],
    queryFn: () => api.events({ integrationId: selectedIntegration?.id, eventType, limit: pageSize, offset: page * pageSize }),
  })
  const total = events.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  useEffect(() => {
    if (events.data && page >= totalPages) setPage(totalPages - 1)
  }, [events.data, page, totalPages])

  const selectIntegration = (value: string) => {
    setIntegrationID(value)
    setEventType('')
    setPage(0)
  }

  const selectEventType = (value: string) => {
    setEventType(value)
    setPage(0)
  }

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Webhook Inbox" title="消息流" description="查看所有接入 App 推送的 Webhook 消息。未知类型会自动使用 App 默认展示。" action={<AppButton size="sm" variant="outline" disabled={events.isFetching} onClick={() => void events.refetch()}><Refresh size={16} />刷新</AppButton>} />
      <Panel title="消息筛选" icon={<Bell size={20} />}>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block space-y-2 text-sm font-medium">
            <span>App</span>
            <select aria-label="App" className="h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" value={integrationID} onChange={(event) => selectIntegration(event.target.value)}>
              <option value="">全部 App</option>
              {(integrations.data ?? []).map((integration) => <option key={integration.id} value={integration.id}>{integration.name} · {apps.data?.find((app) => app.code === integration.app_code)?.name ?? integration.app_code}</option>)}
            </select>
          </label>
          <label className="block space-y-2 text-sm font-medium">
            <span>消息类型</span>
            <select aria-label="消息类型" className="h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-900" value={eventType} disabled={!selectedIntegration} onChange={(event) => selectEventType(event.target.value)}>
              <option value="">{selectedIntegration ? '全部消息类型' : '请先选择 App'}</option>
              {eventTypes.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
            </select>
          </label>
        </div>
      </Panel>
      <Panel title="最近消息" description={`共 ${total} 条`}>
        {events.isPending ? <div className="py-12 text-center text-sm text-neutral-500">正在加载消息…</div> : events.error ? <ErrorState message={errorMessage(events.error)} onRetry={() => void events.refetch()} /> : !events.data?.items.length ? <EmptyState title="还没有 Webhook 消息" description="创建接入实例并向对应地址发送消息后，这里会显示内容。" /> : (
          <div className="space-y-4">
            {events.data.items.map((event) => (
              <Link key={event.id} to={`/events/${event.id}`} className="block rounded-2xl focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"><EventCard event={event} /></Link>
            ))}
            {totalPages > 1 && <div className="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-200 pt-4 dark:border-neutral-800"><p className="text-sm text-neutral-500">第 {page + 1} / {totalPages} 页</p><div className="flex gap-2"><AppButton size="sm" variant="outline" disabled={page === 0 || events.isFetching} onClick={() => setPage((current) => Math.max(0, current - 1))}>上一页</AppButton><AppButton size="sm" variant="outline" disabled={!events.data.has_more || events.isFetching} onClick={() => setPage((current) => current + 1)}>下一页</AppButton></div></div>}
          </div>
        )}
      </Panel>
    </div>
  )
}
