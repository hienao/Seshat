import { EventCardView } from './event-card-view'
import { eventLabel, normalizedSummary, number, record, resolveCategory, safeWebUrl, tagsWithoutMediaType, text } from './model-utils'
import type { EventCardModel, EventCardProps } from './types'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'

export function JellyfinEventCard(props: EventCardProps) {
  const { t } = useTranslation()
  return <EventCardView {...props} model={jellyfinModel(props, t)} />
}

function jellyfinModel({ event }: EventCardProps, t: TFunction): EventCardModel {
  const data = record(event.presentation?.data)
  const media = record(data.media)
  const actor = record(data.actor)
  const playback = record(data.playback)
  const system = record(data.system)
  const category = resolveCategory(event, data)
  const label = eventLabel(event.display_event_type, event.is_fallback, t)
  let title = 'Jellyfin Webhook'
  if (category === 'media' || category === 'playback') title = text(media.display_name) || text(media.name) || title
  if (category === 'security' || category === 'user') title = text(actor.username) || title
  if (category === 'system') title = text(system.name) || title
  return {
    renderer: 'jellyfin', appName: 'Jellyfin', appInitial: 'J', category, label, title,
    summary: normalizedSummary(event.summary, t('events.genericEvent', { label }), category, media), imageUrl: safeWebUrl(text(media.image_url)),
    overview: text(media.overview),
    facts: event.presentation?.facts ?? [], wideFactLabels: [t('events.facts.external_ids'), '外部 ID'], tags: tagsWithoutMediaType(event.presentation?.tags ?? [], media), percent: number(playback.percent),
    positionLabel: text(playback.position_label), durationLabel: text(media.duration_label),
    metadataAttribution: text(media.metadata_source) === 'jellyfin' ? t('events.metadataBy', { provider: 'Jellyfin' }) : text(media.metadata_source) === 'tmdb' ? t('events.metadataBy', { provider: 'TMDB' }) : '', rawPreview: '',
  }
}
