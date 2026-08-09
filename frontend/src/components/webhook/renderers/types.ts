import type { DisplayWebhookEvent } from '@/api/types'

export type DataRecord = Record<string, unknown>
export type EventCategory = 'media' | 'playback' | 'security' | 'user' | 'system' | 'raw'

export interface EventCardProps {
  event: DisplayWebhookEvent
  compact?: boolean
  detail?: boolean
}

export interface EventCardModel {
  renderer: string
  appName: string
  appInitial: string
  category: EventCategory
  label: string
  title: string
  summary: string
  imageUrl: string
  imageLayout?: 'portrait' | 'landscape'
  overview: string
  facts: Array<Record<string, string>>
  wideFactLabels?: string[]
  tags: string[]
  percent?: number
  positionLabel: string
  durationLabel: string
  metadataAttribution: string
  rawPreview: string
}
