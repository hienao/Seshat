import { Badge } from '@appica/ui-react/badge'
import { Activity, ArrowRight, Bell, FileText, Plug, ShieldCheck, Webhook } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'
import { useAuthStore } from '@/stores/auth'

const productFeatures = [
  { icon: Plug, title: '多 App 独立接入', description: '为不同 App 和环境创建独立 Webhook 地址，接入关系清晰可控。' },
  { icon: Bell, title: '按消息类型展示', description: '针对已知事件提供结构化展示，未知类型自动回退到原始消息内容。' },
  { icon: Activity, title: '完整排障链路', description: '从消息详情到接口与业务日志，快速定位签名、请求和服务端处理问题。' },
]

function PublicHome() {
  return (
    <div className="overflow-hidden px-4 pb-20 pt-14 sm:px-6 sm:pt-20">
      <div className="app-grid pointer-events-none absolute inset-x-0 top-0 -z-10 h-[680px]" />
      <section className="rise-in mx-auto max-w-6xl text-center">
        <Badge variant="soft" size="md"><Webhook size={15} />Webhook 消息管理</Badge>
        <h1 className="mx-auto mt-7 max-w-4xl text-5xl font-black tracking-[-0.055em] text-neutral-950 dark:text-white sm:text-7xl">
          接住每一条 Webhook，<span className="text-emerald-700 dark:text-emerald-400">看清每一次事件</span>
        </h1>
        <p className="mx-auto mt-6 max-w-2xl text-lg leading-8 text-neutral-600 dark:text-neutral-400">
          Seshat 统一接收不同 App 的 Webhook，根据消息类型呈现关键信息，并保留原始内容与接口日志，方便追踪和排查。
        </p>
        <div className="mt-9 flex flex-wrap justify-center gap-3">
          <AppButton render={<Link to="/login" />} size="lg">登录控制台<ArrowRight size={18} /></AppButton>
          <AppButton render={<a href="/swagger/index.html" target="_blank" rel="noreferrer" />} variant="outline" size="lg"><FileText size={18} />API 文档</AppButton>
        </div>
      </section>

      <section className="mx-auto mt-20 grid max-w-6xl gap-5 md:grid-cols-3" aria-label="产品能力">
        {productFeatures.map(({ icon: Icon, title, description }, index) => (
          <article key={title} className="app-panel rise-in p-6" style={{ animationDelay: `${120 + index * 90}ms` }}>
            <span className="grid size-12 place-items-center rounded-2xl bg-emerald-50 text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300"><Icon size={24} /></span>
            <h2 className="mt-5 text-xl font-bold tracking-tight">{title}</h2>
            <p className="mt-3 text-sm leading-7 text-neutral-600 dark:text-neutral-400">{description}</p>
          </article>
        ))}
      </section>
    </div>
  )
}

function MetricCard({ label, value, description, icon: Icon }: {
  label: string
  value: string | number
  description: string
  icon: typeof Plug
}) {
  return (
    <article className="app-panel p-5 sm:p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-neutral-500">{label}</p>
          <p className="mt-3 font-mono text-3xl font-bold tracking-tight text-neutral-950 dark:text-white">{value}</p>
        </div>
        <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-emerald-50 text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300"><Icon size={21} /></span>
      </div>
      <p className="mt-3 text-xs text-neutral-500">{description}</p>
    </article>
  )
}

function DashboardHome({ isAdmin }: { isAdmin: boolean }) {
  const integrations = useQuery({ queryKey: ['webhooks', 'integrations'], queryFn: api.integrations })
  const events = useQuery({ queryKey: ['webhooks', 'events', 'overview'], queryFn: () => api.events() })
  const apps = useQuery({ queryKey: ['webhooks', 'apps'], queryFn: api.webhookApps })
  const activeIntegrations = integrations.data?.filter((item) => item.enabled).length ?? 0
  const failedQuery = integrations.error || events.error || apps.error

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Webhook Operations" title="概览" description="查看当前接入规模、消息动态和常用操作。" action={<AppButton render={<Link to="/integrations" />} size="sm"><Plug size={16} />创建接入</AppButton>} />

      {failedQuery && <ErrorState message={errorMessage(failedQuery, '概览数据加载失败')} onRetry={() => { void integrations.refetch(); void events.refetch(); void apps.refetch() }} />}

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" aria-label="运行概况">
        <MetricCard label="接入实例" value={integrations.isPending ? '—' : integrations.data?.length ?? 0} description={`${activeIntegrations} 个正在接收消息`} icon={Plug} />
        <MetricCard label="Webhook 消息" value={events.isPending ? '—' : events.data?.total ?? 0} description="当前账户累计收到的消息" icon={Bell} />
        <MetricCard label="支持的 App" value={apps.isPending ? '—' : apps.data?.length ?? 0} description="可创建独立接入的 App 类型" icon={Webhook} />
      </section>

      <div className="grid gap-7 lg:grid-cols-[minmax(0,1.45fr)_minmax(260px,.55fr)]">
        <Panel title="最近消息" description={events.data?.total ? `共 ${events.data.total} 条消息` : '等待新的 Webhook 消息'} action={<AppButton render={<Link to="/events" />} size="sm" variant="outline">查看全部<ArrowRight size={15} /></AppButton>}>
          {events.isPending ? <div className="py-10 text-center text-sm text-neutral-500">正在加载消息…</div> : !events.data?.items.length ? <EmptyState title="还没有 Webhook 消息" description="创建接入实例并向对应地址发送消息后，最近消息会显示在这里。" /> : (
            <div className="divide-y divide-neutral-200/70 dark:divide-neutral-800/80">
              {events.data.items.slice(0, 5).map((event) => (
                <Link key={event.id} to={`/events/${event.id}`} className="flex items-start justify-between gap-4 rounded-xl px-2 py-4 transition hover:bg-neutral-50 dark:hover:bg-neutral-900/70">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="truncate font-semibold">{event.title || `${event.app_code} Webhook`}</h3>
                      {event.is_fallback && <Badge variant="outline" size="sm">默认展示</Badge>}
                    </div>
                    <p className="mt-1 truncate text-sm text-neutral-500">{event.summary || '无摘要'} · {event.app_code}</p>
                  </div>
                  <time className="shrink-0 text-xs text-neutral-400">{event.received_at ? new Date(event.received_at).toLocaleString() : '-'}</time>
                </Link>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="快捷入口" description="继续管理和排查 Webhook">
          <div className="space-y-2">
            <Link to="/integrations" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><Plug size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">管理接入实例</span><ArrowRight size={16} className="text-neutral-400" /></Link>
            <Link to="/events" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><Bell size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">查看消息流</span><ArrowRight size={16} className="text-neutral-400" /></Link>
            {isAdmin && <Link to="/admin/logs" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><ShieldCheck size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">排查接口日志</span><ArrowRight size={16} className="text-neutral-400" /></Link>}
            {isAdmin && <Link to="/admin/application-logs" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><Activity size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">查看业务日志</span><ArrowRight size={16} className="text-neutral-400" /></Link>}
          </div>
        </Panel>
      </div>
    </div>
  )
}

export function HomePage() {
  const user = useAuthStore((state) => state.user)
  return user ? <DashboardHome isAdmin={user.is_admin} /> : <PublicHome />
}
