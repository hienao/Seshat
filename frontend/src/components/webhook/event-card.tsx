import { Badge } from '@appica/ui-react/badge'
import { AlertCircle, ExternalLink, FileText, Movie, Package, PlayerPlay, Server, Shield, User } from '@appica/icons-react'
import type { ComponentType } from 'react'
import type { WebhookEvent } from '@/api/types'

type DataRecord = Record<string, unknown>
type EventCategory = 'media' | 'playback' | 'security' | 'user' | 'system' | 'raw'

interface EventCardProps {
  event: WebhookEvent
  compact?: boolean
  detail?: boolean
}

const appNames: Record<string, string> = { jellyfin: 'Jellyfin', emby: 'Emby', github: 'GitHub', generic: '通用 Webhook' }

export function EventCard({ event, compact = false, detail = false }: EventCardProps) {
  const data = record(event.presentation?.data)
  const media = record(data.media)
  const actor = record(data.actor)
  const playback = record(data.playback)
  const system = record(data.system)
  const category = resolveCategory(event, data)
  const Icon = categoryIcon(category, event.display_event_type)
  const imageUrl = safeWebUrl(text(media.image_url))
  const label = text(data.event_label) || eventLabel(event.display_event_type, event.is_fallback)
  const title = cardTitle(event, category, media, actor, system)
  const facts = (event.presentation?.facts ?? []).filter((fact) => fact.label && fact.value).slice(0, detail ? 6 : compact ? 2 : 4)
  const percent = number(playback.percent)
  const rawContent = event.raw_body || text(data.raw_preview)
  const rawPreview = category === 'raw' && rawContent ? formatRawPreview(rawContent) : ''

  return (
    <article className={`overflow-hidden rounded-2xl border bg-white/70 transition dark:bg-neutral-950/50 ${toneBorder(event.severity)} ${compact ? 'p-4' : 'p-5 sm:p-6'}`} data-testid="event-card">
      <header className="flex flex-wrap items-center gap-2">
        <span className="grid size-7 place-items-center rounded-lg bg-neutral-950 text-[11px] font-bold text-white dark:bg-white dark:text-neutral-950" aria-hidden="true">{appInitial(event.app_code)}</span>
        <span className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">{appNames[event.app_code] ?? event.app_code}</span>
        <Badge variant={badgeVariant(event.severity, event.is_fallback)} size="sm">{event.is_fallback ? '默认类型' : label}</Badge>
        <time className="ml-auto text-xs text-neutral-400">{formatTime(event.received_at)}</time>
      </header>

      <div className={`mt-4 flex ${compact ? 'gap-3' : 'gap-4 sm:gap-5'}`}>
        <div className={`relative grid shrink-0 place-items-center overflow-hidden rounded-xl ${visualTone(category, event.severity)} ${compact ? 'size-11' : imageUrl ? 'h-24 w-16 sm:h-28 sm:w-20' : 'size-14 sm:size-16'}`}>
          {imageUrl && !compact ? <img src={imageUrl} alt="" className="size-full object-cover" loading="lazy" referrerPolicy="no-referrer" /> : <Icon size={compact ? 22 : 28} />}
        </div>

        <div className="min-w-0 flex-1">
          <h3 className={`${compact ? 'text-sm' : 'text-base sm:text-lg'} truncate font-semibold text-neutral-950 dark:text-white`}>{title}</h3>
          <p className={`${compact ? 'mt-1 line-clamp-1' : 'mt-1.5 line-clamp-2'} text-sm text-neutral-500 dark:text-neutral-400`}>{event.summary || `${label}事件`}</p>

          {category === 'playback' && percent !== undefined && (
            <div className="mt-3" aria-label={`播放进度 ${percent}%`}>
              <div className="mb-1 flex justify-between text-[11px] text-neutral-500"><span>{text(playback.position_label) || '播放进度'}</span><span>{text(media.duration_label) || `${percent}%`}</span></div>
              <div className="h-1.5 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div className="h-full rounded-full bg-sky-500" style={{ width: `${Math.max(0, Math.min(100, percent))}%` }} /></div>
            </div>
          )}

          {!compact && facts.length > 0 && (
            <dl className="mt-4 grid gap-x-5 gap-y-3 sm:grid-cols-2 xl:grid-cols-3">
              {facts.map((fact, index) => <div key={`${fact.label}-${index}`} className="min-w-0"><dt className="text-[11px] text-neutral-400">{fact.label}</dt><dd className="mt-0.5 truncate text-sm font-medium text-neutral-700 dark:text-neutral-200">{maskedFact(fact.label, fact.value)}</dd></div>)}
            </dl>
          )}

          {!compact && event.presentation?.tags && event.presentation.tags.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-2">{event.presentation.tags.map((tag) => <span key={tag} className="rounded-md bg-neutral-100 px-2 py-1 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{tag}</span>)}</div>
          )}
        </div>
      </div>

      {rawPreview && <pre className="mt-4 max-h-44 overflow-auto rounded-xl bg-neutral-950 p-4 text-xs leading-5 text-neutral-200">{rawPreview}</pre>}

      {detail && event.presentation?.links && event.presentation.links.length > 0 && (
        <footer className="mt-5 flex flex-wrap gap-2 border-t border-neutral-200/70 pt-4 dark:border-neutral-800">
          {event.presentation.links.map((link) => safeWebUrl(link.url) && <a key={link.url} href={link.url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-sm font-medium text-neutral-700 transition hover:border-emerald-300 hover:text-emerald-700 dark:border-neutral-700 dark:text-neutral-200 dark:hover:border-emerald-700 dark:hover:text-emerald-300">{link.label || '打开链接'}<ExternalLink size={14} /></a>)}
        </footer>
      )}
    </article>
  )
}

function resolveCategory(event: WebhookEvent, data: DataRecord): EventCategory {
  const explicit = text(data.category)
  if (['media', 'playback', 'security', 'user', 'system', 'raw'].includes(explicit)) return explicit as EventCategory
  if (event.is_fallback || event.display_event_type === '__default__') return 'raw'
  if (event.display_event_type.startsWith('media_')) return 'media'
  if (event.display_event_type.startsWith('playback_')) return 'playback'
  if (event.display_event_type.startsWith('authentication_') || event.display_event_type === 'user_locked_out') return 'security'
  if (event.display_event_type.startsWith('user_')) return 'user'
  return 'system'
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

function cardTitle(event: WebhookEvent, category: EventCategory, media: DataRecord, actor: DataRecord, system: DataRecord) {
  if (category === 'media' || category === 'playback') return text(media.display_name) || text(media.name) || event.title || `${event.app_code} Webhook`
  if (category === 'security' || category === 'user') return text(actor.username) || event.title || `${event.app_code} Webhook`
  if (category === 'system') return text(system.name) || event.title || `${event.app_code} Webhook`
  return event.source_event_type && event.source_event_type !== 'unknown' ? event.source_event_type : event.title || '原始 Webhook 消息'
}

function eventLabel(eventType: string, fallback: boolean) {
  if (fallback) return '其他消息'
  const labels: Record<string, string> = {
    media_added: '新增媒体', media_deleted: '删除媒体', playback_started: '开始播放', playback_progress: '播放进度', playback_stopped: '停止播放',
    authentication_success: '登录成功', authentication_failure: '登录失败', session_started: '会话开始', server_restart_required: '服务等待重启',
    task_completed: '计划任务完成', subtitle_download_failed: '字幕下载失败', plugin_installing: '正在安装插件', plugin_installed: '插件安装完成',
    plugin_install_failed: '插件安装失败', plugin_install_cancelled: '插件安装取消', plugin_updated: '插件已更新', plugin_uninstalled: '插件已卸载',
    user_created: '用户已创建', user_updated: '用户信息更新', user_deleted: '用户已删除', user_locked_out: '用户被锁定',
    user_password_changed: '用户密码已修改', user_data_saved: '用户数据已保存', generic: '通用通知',
  }
  return labels[eventType] ?? eventType
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

function appInitial(appCode: string) {
  return (appNames[appCode] ?? appCode).slice(0, 1).toUpperCase() || 'W'
}

function formatTime(value?: string) {
  return value ? new Date(value).toLocaleString() : '-'
}

function formatRawPreview(value: string) {
  try { return JSON.stringify(JSON.parse(value), null, 2) }
  catch { return value }
}

function safeWebUrl(value: string) {
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : ''
  } catch { return '' }
}

function record(value: unknown): DataRecord {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as DataRecord : {}
}

function text(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function number(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}
