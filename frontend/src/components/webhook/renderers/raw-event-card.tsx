import { EventCardView } from './event-card-view'
import { appNames, formatRawPreview, record, text } from './model-utils'
import type { EventCardModel, EventCardProps } from './types'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'

export function RawEventCard(props: EventCardProps) {
  const { t } = useTranslation()
  return <EventCardView {...props} model={rawModel(props, t)} />
}

function rawModel({ event }: EventCardProps, t: TFunction): EventCardModel {
  const data = record(event.presentation?.data)
  const appName = appNames[event.app_code] ?? event.app_code
  const sourceType = event.source_event_type && event.source_event_type !== 'unknown' ? event.source_event_type : ''
  const rawContent = 'raw_body' in event && typeof event.raw_body === 'string' ? event.raw_body : text(data.raw_preview)
  return {
    renderer: `raw:${event.app_code}`, appName, appInitial: appName.slice(0, 1).toUpperCase() || 'W', category: 'raw',
    label: t('events.otherMessage'), title: sourceType || event.title || t('events.defaultTitle'), summary: event.summary || t('events.defaultDescription'),
    imageUrl: '', overview: '', facts: event.presentation?.facts ?? [], tags: event.presentation?.tags ?? [],
    positionLabel: '', durationLabel: '', metadataAttribution: '', rawPreview: rawContent ? formatRawPreview(rawContent) : '',
  }
}
