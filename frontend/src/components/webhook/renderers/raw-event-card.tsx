import { EventCardView } from './event-card-view'
import { appNames, formatRawPreview, record, text } from './model-utils'
import type { EventCardModel, EventCardProps } from './types'

export function RawEventCard(props: EventCardProps) {
  return <EventCardView {...props} model={rawModel(props)} />
}

function rawModel({ event }: EventCardProps): EventCardModel {
  const data = record(event.presentation?.data)
  const appName = appNames[event.app_code] ?? event.app_code
  const sourceType = event.source_event_type && event.source_event_type !== 'unknown' ? event.source_event_type : ''
  const rawContent = 'raw_body' in event && typeof event.raw_body === 'string' ? event.raw_body : text(data.raw_preview)
  return {
    renderer: `raw:${event.app_code}`, appName, appInitial: appName.slice(0, 1).toUpperCase() || 'W', category: 'raw',
    label: '其他消息', title: sourceType || event.title || '原始 Webhook 消息', summary: event.summary || '收到一条未知类型的 Webhook 消息',
    imageUrl: '', overview: '', facts: event.presentation?.facts ?? [], tags: event.presentation?.tags ?? [],
    positionLabel: '', durationLabel: '', metadataAttribution: '', rawPreview: rawContent ? formatRawPreview(rawContent) : '',
  }
}
