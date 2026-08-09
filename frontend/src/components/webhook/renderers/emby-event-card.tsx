import { EventCardView } from './event-card-view'
import { eventLabel, normalizedSummary, number, record, resolveCategory, safeWebUrl, tagsWithoutMediaType, text } from './model-utils'
import type { EventCardModel, EventCardProps } from './types'

export function EmbyEventCard(props: EventCardProps) {
  return <EventCardView {...props} model={embyModel(props)} />
}

function embyModel({ event }: EventCardProps): EventCardModel {
  const data = record(event.presentation?.data)
  const media = record(data.media)
  const actor = record(data.actor)
  const playback = record(data.playback)
  const system = record(data.system)
  const category = resolveCategory(event, data)
  const label = text(data.event_label) || eventLabel(event.display_event_type, event.is_fallback)
  let title = event.title || 'Emby Webhook'
  if (category === 'media' || category === 'playback') title = text(media.display_name) || text(media.name) || title
  if (category === 'security' || category === 'user') title = text(actor.username) || title
  if (category === 'system') title = text(system.name) || title
  return {
    renderer: 'emby', appName: 'Emby', appInitial: 'E', category, label, title,
    summary: normalizedSummary(event.summary, `${label}事件`, category, media), imageUrl: safeWebUrl(text(media.image_url)),
    imageLayout: text(media.type).toLowerCase() === 'episode' ? 'landscape' : 'portrait', overview: text(media.overview),
    facts: event.presentation?.facts ?? [], wideFactLabels: ['外部 ID'], tags: tagsWithoutMediaType(event.presentation?.tags ?? [], media), percent: number(playback.percent),
    positionLabel: text(playback.position_label), durationLabel: text(media.duration_label),
    metadataAttribution: text(media.metadata_source) === 'emby' ? '媒体资料由 Emby 提供' : text(media.metadata_source) === 'tmdb' ? '媒体资料由 TMDB 提供' : '', rawPreview: '',
  }
}
