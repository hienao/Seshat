import { EventCardView } from './event-card-view'
import { appNames, eventLabel, number, record, resolveCategory, safeWebUrl, text } from './model-utils'
import type { EventCardModel, EventCardProps } from './types'

export function DefaultEventCard(props: EventCardProps) {
  return <EventCardView {...props} model={defaultModel(props)} />
}

function defaultModel({ event }: EventCardProps): EventCardModel {
  const data = record(event.presentation?.data)
  const media = record(data.media)
  const actor = record(data.actor)
  const playback = record(data.playback)
  const system = record(data.system)
  const category = resolveCategory(event, data)
  const label = text(data.event_label) || eventLabel(event.display_event_type, event.is_fallback)
  let title = event.title || `${event.app_code} Webhook`
  if (category === 'media' || category === 'playback') title = text(media.display_name) || text(media.name) || title
  if (category === 'security' || category === 'user') title = text(actor.username) || title
  if (category === 'system') title = text(system.name) || title
  const appName = appNames[event.app_code] ?? event.app_code
  return {
    renderer: `default:${event.app_code}`, appName, appInitial: appName.slice(0, 1).toUpperCase() || 'W', category, label, title,
    summary: event.summary || `${label}事件`, imageUrl: safeWebUrl(text(media.image_url)), overview: text(media.overview),
    facts: event.presentation?.facts ?? [], tags: event.presentation?.tags ?? [], percent: number(playback.percent),
    positionLabel: text(playback.position_label), durationLabel: text(media.duration_label), metadataAttribution: '', rawPreview: '',
  }
}
