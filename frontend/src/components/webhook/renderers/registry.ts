import type { ComponentType } from 'react'
import type { DisplayWebhookEvent } from '@/api/types'
import { DefaultEventCard } from './default-event-card'
import { EmbyEventCard } from './emby-event-card'
import { JellyfinEventCard } from './jellyfin-event-card'
import { RawEventCard } from './raw-event-card'
import type { EventCardProps } from './types'

type EventRenderer = ComponentType<EventCardProps>

interface AppRendererDefinition {
  defaultRenderer: EventRenderer
  eventRenderers?: Record<string, EventRenderer>
}

const rendererRegistry: Record<string, AppRendererDefinition> = {
  jellyfin: { defaultRenderer: JellyfinEventCard, eventRenderers: {} },
  emby: { defaultRenderer: EmbyEventCard, eventRenderers: {} },
  github: { defaultRenderer: DefaultEventCard, eventRenderers: {} },
  generic: { defaultRenderer: DefaultEventCard, eventRenderers: {} },
}

export function resolveEventRenderer(event: DisplayWebhookEvent): EventRenderer {
  if (event.is_fallback || event.display_event_type === '__default__') return RawEventCard
  const app = rendererRegistry[event.app_code]
  if (!app) return DefaultEventCard
  return app.eventRenderers?.[event.display_event_type] ?? app.defaultRenderer
}
