import { EventCardView } from './event-card-view'
import { appNames, eventLabel, number, record, resolveCategory, safeWebUrl, text } from './model-utils'
import type { EventCardModel, EventCardProps } from './types'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'

export function DefaultEventCard(props: EventCardProps) {
  const { t } = useTranslation()
  return <EventCardView {...props} model={defaultModel(props, t)} />
}

function defaultModel({ event }: EventCardProps, t: TFunction): EventCardModel {
  const data = record(event.presentation?.data)
  const media = record(data.media)
  const actor = record(data.actor)
  const playback = record(data.playback)
  const system = record(data.system)
  const category = resolveCategory(event, data)
  const label = eventLabel(event.display_event_type, event.is_fallback, t)
  let title = `${event.app_code} Webhook`
  if (category === 'media' || category === 'playback') title = text(media.display_name) || text(media.name) || title
  if (category === 'security' || category === 'user') title = text(actor.username) || title
  if (category === 'system') title = text(system.name) || title
  const appName = appNames[event.app_code] ?? event.app_code
  return {
    renderer: `default:${event.app_code}`, appName, appInitial: appName.slice(0, 1).toUpperCase() || 'W', category, label, title,
    summary: event.summary || t('events.genericEvent', { label }), imageUrl: safeWebUrl(text(media.image_url)), overview: text(media.overview),
    facts: event.presentation?.facts ?? [], tags: event.presentation?.tags ?? [], percent: number(playback.percent),
    positionLabel: text(playback.position_label), durationLabel: text(media.duration_label), metadataAttribution: '', rawPreview: '',
  }
}
