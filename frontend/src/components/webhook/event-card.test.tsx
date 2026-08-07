import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { WebhookEvent } from '@/api/types'
import { EventCard } from './event-card'

function event(overrides: Partial<WebhookEvent> = {}): WebhookEvent {
  return {
    id: 1,
    integration_id: 1,
    app_code: 'jellyfin',
    source_event_type: 'PlaybackProgress',
    display_event_type: 'playback_progress',
    external_event_id: '',
    status: 'processed',
    is_fallback: false,
    title: 'Jellyfin · 播放进度 · Severance',
    summary: 'mark · Living Room · Transcode',
    severity: 'info',
    presentation_version: 1,
    presentation: {
      schema_version: 1,
      title: 'Jellyfin · 播放进度 · Severance',
      summary: 'mark · Living Room · Transcode',
      severity: 'info',
      facts: [{ label: '用户', value: 'mark' }],
      data: {
        category: 'playback',
        event_label: '播放进度',
        media: { display_name: 'Severance · S01E01 · Pilot', duration_label: '60:00' },
        playback: { percent: 25, position_label: '15:00', method: 'Transcode' },
      },
    },
    content_type: 'application/json',
    received_at: '2026-08-08T02:00:00Z',
    ...overrides,
  }
}

describe('EventCard', () => {
  it('renders a normalized playback card and progress', () => {
    render(<EventCard event={event()} />)
    expect(screen.getByText('播放进度')).toBeInTheDocument()
    expect(screen.getByText('Severance · S01E01 · Pilot')).toBeInTheDocument()
    expect(screen.getByLabelText('播放进度 25%')).toBeInTheDocument()
    expect(screen.getByText('mark')).toBeInTheDocument()
  })

  it('renders unknown events as raw cards', () => {
    render(<EventCard event={event({
      source_event_type: 'PluginCustomEvent',
      display_event_type: '__default__',
      is_fallback: true,
      raw_body: '{"hello":"world"}',
      presentation: { schema_version: 1, title: '其他消息', severity: 'info', data: { category: 'raw' } },
    })} detail />)
    expect(screen.getByText('默认类型')).toBeInTheDocument()
    expect(screen.getByText('PluginCustomEvent')).toBeInTheDocument()
    expect(screen.getByText(/"hello": "world"/)).toBeInTheDocument()
  })
})
