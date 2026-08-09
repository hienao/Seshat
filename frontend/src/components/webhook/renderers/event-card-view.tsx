import { Badge } from '@appica/ui-react/badge'
import { AlertCircle, ExternalLink, FileText, Movie, Package, PlayerPlay, Server, Shield, User } from '@appica/icons-react'
import type { ComponentType } from 'react'
import type { EventCardModel, EventCardProps, EventCategory } from './types'
import { safeWebUrl } from './model-utils'

interface EventCardViewProps extends EventCardProps {
  model: EventCardModel
}

export function EventCardView({ event, model, compact = false, detail = false }: EventCardViewProps) {
  const Icon = categoryIcon(model.category, event.display_event_type)
  const facts = model.facts.filter((fact) => fact.label && fact.value).slice(0, detail ? 8 : compact ? 2 : 6)

  return (
    <article className={`overflow-hidden rounded-2xl border bg-white/70 transition dark:bg-neutral-950/50 ${toneBorder(event.severity)} ${compact ? 'p-4' : 'p-5 sm:p-6'}`} data-testid="event-card" data-renderer={model.renderer}>
      <header className="flex flex-wrap items-center gap-2">
        <span className="grid size-7 place-items-center rounded-lg bg-neutral-950 text-[11px] font-bold text-white dark:bg-white dark:text-neutral-950" aria-hidden="true">{model.appInitial}</span>
        <span className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">{model.appName}</span>
        <Badge variant={badgeVariant(event.severity, event.is_fallback)} size="sm">{event.is_fallback ? '默认类型' : model.label}</Badge>
        <time className="ml-auto text-xs text-neutral-400">{formatTime(event.received_at)}</time>
      </header>

      <div className={`mt-4 flex ${compact ? 'gap-3' : 'gap-4 sm:gap-5'}`}>
        <div className={`relative grid shrink-0 place-items-center overflow-hidden rounded-xl ${visualTone(model.category, event.severity)} ${visualSize(model, compact, detail)}`}>
          {model.imageUrl && !compact ? <img src={model.imageUrl} alt="" className="size-full object-cover" loading="lazy" referrerPolicy="no-referrer" /> : <Icon size={compact ? 22 : 28} />}
        </div>

        <div className="min-w-0 flex-1">
          <h3 className={`${compact ? 'text-sm' : 'text-base sm:text-lg'} truncate font-semibold text-neutral-950 dark:text-white`}>{model.title}</h3>
          <p className={`${compact ? 'mt-1 line-clamp-1' : 'mt-1.5 line-clamp-2'} text-sm text-neutral-500 dark:text-neutral-400`}>{model.summary}</p>

          {!compact && model.overview && <p className="mt-3 line-clamp-3 text-sm leading-6 text-neutral-600 dark:text-neutral-300">{model.overview}</p>}

          {model.category === 'playback' && model.percent !== undefined && (
            <div className="mt-3" aria-label={`播放进度 ${model.percent}%`}>
              <div className="mb-1 flex justify-between text-[11px] text-neutral-500"><span>{model.positionLabel || '播放进度'}</span><span>{model.durationLabel || `${model.percent}%`}</span></div>
              <div className="h-1.5 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div className="h-full rounded-full bg-sky-500" style={{ width: `${Math.max(0, Math.min(100, model.percent))}%` }} /></div>
            </div>
          )}

          {!compact && facts.length > 0 && (
            <dl className="mt-4 grid gap-x-5 gap-y-3 sm:grid-cols-2 xl:grid-cols-3">
              {facts.map((fact, index) => {
                const wide = model.wideFactLabels?.includes(fact.label)
                return <div key={`${fact.label}-${index}`} className={`min-w-0 ${wide ? 'sm:col-span-2 xl:col-span-3' : ''}`}><dt className="text-[11px] text-neutral-400">{fact.label}</dt><dd className={`mt-0.5 text-sm font-medium text-neutral-700 dark:text-neutral-200 ${wide ? 'whitespace-normal break-words' : 'truncate'}`}>{maskedFact(fact.label, fact.value)}</dd></div>
              })}
            </dl>
          )}

          {!compact && model.tags.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-2">{model.tags.map((tag) => <span key={tag} className="rounded-md bg-neutral-100 px-2 py-1 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{tag}</span>)}</div>
          )}
          {!compact && model.metadataAttribution && <p className="mt-3 text-[11px] text-neutral-400">{model.metadataAttribution}</p>}
        </div>
      </div>

      {model.rawPreview && <pre className="mt-4 max-h-44 overflow-auto rounded-xl bg-neutral-950 p-4 text-xs leading-5 text-neutral-200">{model.rawPreview}</pre>}

      {detail && event.presentation?.links && event.presentation.links.length > 0 && (
        <footer className="mt-5 flex flex-wrap gap-2 border-t border-neutral-200/70 pt-4 dark:border-neutral-800">
          {event.presentation.links.map((link) => safeWebUrl(link.url) && <a key={link.url} href={link.url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-sm font-medium text-neutral-700 transition hover:border-emerald-300 hover:text-emerald-700 dark:border-neutral-700 dark:text-neutral-200 dark:hover:border-emerald-700 dark:hover:text-emerald-300">{link.label || '打开链接'}<ExternalLink size={14} /></a>)}
        </footer>
      )}
    </article>
  )
}

function visualSize(model: EventCardModel, compact: boolean, detail: boolean) {
  if (compact) return 'size-11'
  if (!model.imageUrl) return 'size-14 sm:size-16'
  if (model.imageLayout === 'landscape') return detail ? 'aspect-video w-40 sm:w-64' : 'aspect-video w-28 sm:w-44'
  return detail ? 'h-44 w-28 sm:h-56 sm:w-40' : 'h-24 w-16 sm:h-28 sm:w-20'
}

function categoryIcon(category: EventCategory, eventType: string): ComponentType<{ size?: number }> {
  if (category === 'media') return Movie
  if (category === 'playback') return PlayerPlay
  if (category === 'security') return Shield
  if (category === 'user') return User
  if (category === 'raw') return FileText
  if (eventType.startsWith('plugin_')) return Package
  if (eventType.includes('failed')) return AlertCircle
  return Server
}

function badgeVariant(severity: string, fallback: boolean): 'outline' | 'success' | 'error' | 'warning' | 'info' | 'soft' {
  if (fallback) return 'outline'
  if (severity === 'success') return 'success'
  if (severity === 'error') return 'error'
  if (severity === 'warning') return 'warning'
  if (severity === 'info') return 'info'
  return 'soft'
}

function toneBorder(severity: string) {
  if (severity === 'error') return 'border-red-200/90 hover:border-red-300 dark:border-red-950 dark:hover:border-red-800'
  if (severity === 'warning') return 'border-amber-200/90 hover:border-amber-300 dark:border-amber-950 dark:hover:border-amber-800'
  if (severity === 'success') return 'border-emerald-200/90 hover:border-emerald-300 dark:border-emerald-950 dark:hover:border-emerald-800'
  return 'border-neutral-200/80 hover:border-sky-300 dark:border-neutral-800 dark:hover:border-sky-800'
}

function visualTone(category: EventCategory, severity: string) {
  if (severity === 'error') return 'bg-red-50 text-red-700 dark:bg-red-950/70 dark:text-red-300'
  if (severity === 'warning') return 'bg-amber-50 text-amber-700 dark:bg-amber-950/70 dark:text-amber-300'
  if (category === 'playback') return 'bg-sky-50 text-sky-700 dark:bg-sky-950/70 dark:text-sky-300'
  if (category === 'media' || severity === 'success') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300'
  return 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300'
}

function maskedFact(label: string, value: string) {
  if (label !== '来源') return value
  if (value.includes(':') && !value.includes('.')) return `${value.slice(0, 6)}…`
  const parts = value.split('.')
  return parts.length === 4 ? `${parts[0]}.${parts[1]}.*.*` : value
}

function formatTime(value?: string) {
  return value ? new Date(value).toLocaleString() : '-'
}
