import type { EventCardProps } from './renderers/types'
import { resolveEventRenderer } from './renderers/registry'

export function EventCard(props: EventCardProps) {
  const Renderer = resolveEventRenderer(props.event)
  return <Renderer {...props} />
}
