import type { DisplayWebhookEvent } from '@/api/types'
import type { TFunction } from 'i18next'
import type { DataRecord, EventCategory } from './types'

export const appNames: Record<string, string> = { jellyfin: 'Jellyfin', emby: 'Emby', github: 'GitHub', generic: 'Generic Webhook' }

export function resolveCategory(event: DisplayWebhookEvent, data: DataRecord): EventCategory {
  const explicit = text(data.category)
  if (['media', 'playback', 'security', 'user', 'system', 'raw'].includes(explicit)) return explicit as EventCategory
  if (event.is_fallback || event.display_event_type === '__default__') return 'raw'
  if (event.display_event_type.startsWith('media_')) return 'media'
  if (event.display_event_type.startsWith('playback_')) return 'playback'
  if (event.display_event_type.startsWith('authentication_') || event.display_event_type === 'user_locked_out') return 'security'
  if (event.display_event_type.startsWith('user_')) return 'user'
  return 'system'
}

export function eventLabel(eventType: string, fallback: boolean, t: TFunction) {
  return t(fallback ? 'events.otherMessage' : `events.types.${eventType}`, { defaultValue: eventType })
}

export function record(value: unknown): DataRecord {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as DataRecord : {}
}

export function text(value: unknown) {
  return typeof value === 'string' ? value : ''
}

export function number(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

export function normalizedSummary(summary: string, fallback: string, category: EventCategory, media: DataRecord) {
  const value = summary.trim()
  if (/^收到\s.+事件$/.test(value)) return fallback
  if (category === 'media' && value && value === text(media.type)) return fallback
  return value || fallback
}

export function tagsWithoutMediaType(tags: string[], media: DataRecord) {
  const mediaType = text(media.type)
  return tags.filter((tag, index) => tag && tag !== mediaType && tags.indexOf(tag) === index)
}

export function safeWebUrl(value: string) {
	if (value.startsWith('/api/public/media-images/') && !value.startsWith('//')) return value
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : ''
  } catch { return '' }
}

export function formatRawPreview(value: string) {
  try { return JSON.stringify(JSON.parse(value), null, 2) }
  catch { return value }
}
