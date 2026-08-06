import { Badge } from '@appica/ui-react/badge'
import { Bell, Refresh } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useState } from 'react'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'

export function EventsPage() {
  const [appCode, setAppCode] = useState('')
  const apps = useQuery({ queryKey: ['webhooks', 'apps'], queryFn: api.webhookApps })
  const events = useQuery({ queryKey: ['webhooks', 'events', appCode], queryFn: () => api.events({ appCode }) })

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Webhook Inbox" title="消息流" description="查看所有接入 App 推送的 Webhook 消息。未知类型会自动使用 App 默认展示。" action={<AppButton size="sm" variant="outline" disabled={events.isFetching} onClick={() => void events.refetch()}><Refresh size={16} />刷新</AppButton>} />
      <Panel title="消息筛选" icon={<Bell size={20} />}>
        <div className="max-w-sm">
          <label className="mb-2 block text-sm font-medium">App 类型</label>
          <select className="h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" value={appCode} onChange={(event) => setAppCode(event.target.value)}><option value="">全部 App</option>{(apps.data ?? []).map((app) => <option key={app.code} value={app.code}>{app.name}</option>)}</select>
        </div>
      </Panel>
      <Panel title="最近消息" description={`共 ${events.data?.total ?? 0} 条`}>
        {events.isPending ? <div className="py-12 text-center text-sm text-neutral-500">正在加载消息…</div> : events.error ? <ErrorState message={errorMessage(events.error)} onRetry={() => void events.refetch()} /> : !events.data?.items.length ? <EmptyState title="还没有 Webhook 消息" description="创建接入实例并向对应地址发送消息后，这里会显示内容。" /> : (
          <div className="divide-y divide-neutral-200/70 dark:divide-neutral-800/80">
            {events.data.items.map((event) => (
              <Link key={event.id} to={`/events/${event.id}`} className="block rounded-xl px-2 py-4 transition hover:bg-neutral-50 dark:hover:bg-neutral-900/70">
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0"><div className="flex flex-wrap items-center gap-2"><h3 className="truncate font-semibold">{event.title || `${event.app_code} Webhook`}</h3><Badge variant={event.is_fallback ? 'outline' : 'soft'} size="sm">{event.is_fallback ? '默认类型' : event.display_event_type}</Badge></div><p className="mt-1 text-sm text-neutral-500">{event.summary || '无摘要'} · {event.app_code}</p></div>
                  <time className="shrink-0 text-xs text-neutral-400">{event.received_at ? new Date(event.received_at).toLocaleString() : '-'}</time>
                </div>
              </Link>
            ))}
          </div>
        )}
      </Panel>
    </div>
  )
}
