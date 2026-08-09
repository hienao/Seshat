import { Badge } from '@appica/ui-react/badge'
import { Activity, ArrowRight, Bell, FileText, Plug, ShieldCheck, Webhook } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { EventCard } from '@/components/webhook/event-card'
import { errorMessage } from '@/lib/error-message'
import { useAuthStore } from '@/stores/auth'

function PublicHome() {
  const { t } = useTranslation()
  const productFeatures = [
    { icon: Plug, title: t('home.public.independentTitle'), description: t('home.public.independentDescription') },
    { icon: Bell, title: t('home.public.typedTitle'), description: t('home.public.typedDescription') },
    { icon: Activity, title: t('home.public.diagnosticsTitle'), description: t('home.public.diagnosticsDescription') },
  ]
  return (
    <div className="overflow-hidden px-4 pb-20 pt-14 sm:px-6 sm:pt-20">
      <div className="app-grid pointer-events-none absolute inset-x-0 top-0 -z-10 h-[680px]" />
      <section className="rise-in mx-auto max-w-6xl text-center">
        <Badge variant="soft" size="md"><Webhook size={15} />{t('home.public.badge')}</Badge>
        <h1 className="mx-auto mt-7 max-w-4xl text-5xl font-black tracking-[-0.055em] text-neutral-950 dark:text-white sm:text-7xl">
          {t('home.public.headline')} <span className="text-emerald-700 dark:text-emerald-400">{t('home.public.headlineAccent')}</span>
        </h1>
        <p className="mx-auto mt-6 max-w-2xl text-lg leading-8 text-neutral-600 dark:text-neutral-400">
          {t('home.public.description')}
        </p>
        <div className="mt-9 flex flex-wrap justify-center gap-3">
          <AppButton render={<Link to="/login" />} size="lg">{t('home.public.console')}<ArrowRight size={18} /></AppButton>
          <AppButton render={<a href="/swagger/index.html" target="_blank" rel="noreferrer" />} variant="outline" size="lg"><FileText size={18} />{t('home.public.apiDocs')}</AppButton>
        </div>
      </section>

      <section className="mx-auto mt-20 grid max-w-6xl gap-5 md:grid-cols-3" aria-label={t('home.public.capabilities')}>
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
  const { t } = useTranslation()
  const integrations = useQuery({ queryKey: ['webhooks', 'integrations'], queryFn: api.integrations })
  const events = useQuery({ queryKey: ['webhooks', 'events', 'overview'], queryFn: () => api.events() })
  const apps = useQuery({ queryKey: ['webhooks', 'apps'], queryFn: api.webhookApps })
  const activeIntegrations = integrations.data?.filter((item) => item.enabled).length ?? 0
  const failedQuery = integrations.error || events.error || apps.error

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow={t('home.eyebrow')} title={t('home.title')} description={t('home.description')} action={<AppButton render={<Link to="/integrations" />} size="sm"><Plug size={16} />{t('home.createIntegration')}</AppButton>} />

      {failedQuery && <ErrorState message={errorMessage(failedQuery, t('home.loadFailed'))} onRetry={() => { void integrations.refetch(); void events.refetch(); void apps.refetch() }} />}

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" aria-label={t('home.metricsAria')}>
        <MetricCard label={t('home.integrations')} value={integrations.isPending ? '—' : integrations.data?.length ?? 0} description={t('home.activeIntegrations', { count: activeIntegrations })} icon={Plug} />
        <MetricCard label={t('home.totalMessages')} value={events.isPending ? '—' : events.data?.total ?? 0} description={t('home.totalMessagesDescription')} icon={Bell} />
        <MetricCard label={t('home.supportedApps')} value={apps.isPending ? '—' : apps.data?.length ?? 0} description={t('home.supportedAppsDescription')} icon={Webhook} />
      </section>

      <div className="grid gap-7 lg:grid-cols-[minmax(0,1.45fr)_minmax(260px,.55fr)]">
        <Panel title={t('home.recentMessages')} description={events.data?.total ? t('home.recentDescription', { count: events.data.total }) : t('home.waitingMessages')} action={<AppButton render={<Link to="/events" />} size="sm" variant="outline">{t('home.viewAll')}<ArrowRight size={15} /></AppButton>}>
          {events.isPending ? <div className="py-10 text-center text-sm text-neutral-500">{t('home.loadingMessages')}</div> : !events.data?.items.length ? <EmptyState title={t('home.noMessages')} description={t('home.noMessagesDescription')} /> : (
            <div className="space-y-3">
              {events.data.items.slice(0, 5).map((event) => (
                <Link key={event.id} to={`/events/${event.id}`} className="block rounded-2xl focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"><EventCard event={event} compact /></Link>
              ))}
            </div>
          )}
        </Panel>

        <Panel title={t('home.quickActions')} description={t('home.quickDescription')}>
          <div className="space-y-2">
            <Link to="/integrations" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><Plug size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">{t('home.manageIntegrations')}</span><ArrowRight size={16} className="text-neutral-400" /></Link>
            <Link to="/events" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><Bell size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">{t('home.manageChannels')}</span><ArrowRight size={16} className="text-neutral-400" /></Link>
            {isAdmin && <Link to="/admin/logs" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><ShieldCheck size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">{t('home.viewLogs')}</span><ArrowRight size={16} className="text-neutral-400" /></Link>}
            {isAdmin && <Link to="/admin/application-logs" className="flex items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-sm font-medium transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-neutral-800 dark:hover:border-emerald-800 dark:hover:bg-emerald-950/40"><Activity size={18} className="text-emerald-700 dark:text-emerald-300" /><span className="flex-1">{t('home.viewApplicationLogs')}</span><ArrowRight size={16} className="text-neutral-400" /></Link>}
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
