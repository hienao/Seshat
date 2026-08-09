import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import type { WebhookEvent } from '@/api/types'
import { EventCard } from './event-card'

afterEach(cleanup)

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
      facts: [{ label: '用户', value: 'mark' }, { label: '客户端', value: 'Jellyfin Web' }, { label: '设备', value: 'Living Room' }, { label: '媒体信息', value: '1080p H264 SDR' }],
      data: {
        category: 'playback',
        event_label: '播放进度',
        media: { display_name: 'Severance · S01E01 · Pilot', type: 'Episode', duration_label: '60:00', overview: '团队发现了新的线索。', image_url: 'https://image.tmdb.org/t/p/w342/poster.jpg', metadata_source: 'tmdb' },
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
    expect(screen.getByTestId('event-card')).toHaveAttribute('data-renderer', 'jellyfin')
    expect(screen.getByText('播放进度')).toBeInTheDocument()
    expect(screen.getByText('Severance · S01E01 · Pilot')).toBeInTheDocument()
    expect(screen.getByLabelText('播放进度 25%')).toBeInTheDocument()
    expect(screen.getByText('mark')).toBeInTheDocument()
    expect(screen.getByText('Jellyfin Web')).toBeInTheDocument()
    expect(screen.getByText('Living Room')).toBeInTheDocument()
    expect(screen.getByText('1080p H264 SDR')).toBeInTheDocument()
    expect(screen.getByText('团队发现了新的线索。')).toBeInTheDocument()
    expect(screen.getByText('媒体资料由 TMDB 提供')).toBeInTheDocument()
  })

  it('does not repeat a legacy media type in the summary and tags', () => {
    render(<EventCard event={event({
      source_event_type: 'ItemAdded',
      display_event_type: 'media_added',
      summary: 'Episode',
      presentation: {
        schema_version: 1,
        title: 'Jellyfin · 新增媒体 · 我的团长我的团 · S01E04',
        summary: 'Episode',
        severity: 'success',
        facts: [{ label: '类型', value: 'Episode' }],
        tags: ['Episode'],
        data: {
          category: 'media',
          event_label: '新增媒体',
          media: { display_name: '我的团长我的团 · S01E04', type: 'Episode' },
        },
      },
    })} />)
    expect(screen.getByText('新增媒体事件')).toBeInTheDocument()
    expect(screen.getAllByText('Episode')).toHaveLength(1)
  })

  it('uses a larger landscape image and shows complete external IDs in detail mode', () => {
    const externalIDs = 'TMDB: 1452857 · TVDB: 6493171 · IMDb: tt8150114'
    const item = event()
    item.presentation.facts = [...(item.presentation.facts ?? []), { label: '外部 ID', value: externalIDs }]
    const { container } = render(<EventCard event={item} detail />)

    const image = container.querySelector('img')
    expect(image?.parentElement).toHaveClass('aspect-video', 'w-40', 'sm:w-64')
    const externalIDValue = screen.getByText(externalIDs)
    expect(externalIDValue).toHaveClass('whitespace-normal', 'break-words')
    expect(externalIDValue).not.toHaveClass('truncate')
    expect(externalIDValue.parentElement).toHaveClass('sm:col-span-2', 'xl:col-span-3')
  })

  it('accepts cached media image paths and attributes Jellyfin metadata', () => {
		const item = event()
		const data = item.presentation.data as Record<string, unknown>
		data.media = { display_name: 'Jellyfin item', type: 'Episode', image_url: '/api/public/media-images/random-token', metadata_source: 'jellyfin' }
		const { container } = render(<EventCard event={item} />)
		expect(container.querySelector('img')).toHaveAttribute('src', '/api/public/media-images/random-token')
		expect(screen.getByText('媒体资料由 Jellyfin 提供')).toBeInTheDocument()
	})

  it('renders unknown events as raw cards', () => {
    render(<EventCard event={event({
      source_event_type: 'PluginCustomEvent',
      display_event_type: '__default__',
      is_fallback: true,
      raw_body: '{"hello":"world"}',
      presentation: { schema_version: 1, title: '其他消息', severity: 'info', data: { category: 'raw' } },
    })} detail />)
    expect(screen.getByTestId('event-card')).toHaveAttribute('data-renderer', 'raw:jellyfin')
    expect(screen.getByText('默认类型')).toBeInTheDocument()
    expect(screen.getByText('PluginCustomEvent')).toBeInTheDocument()
    expect(screen.getByText(/"hello": "world"/)).toBeInTheDocument()
  })

  it('routes Emby events to the independent Emby renderer', () => {
		const item = event({ app_code: 'emby', title: 'Emby · 播放进度 · Severance' })
		const data = item.presentation.data as Record<string, unknown>
		data.media = { display_name: 'Emby item', type: 'Episode', metadata_source: 'emby' }
		render(<EventCard event={item} />)
    expect(screen.getByTestId('event-card')).toHaveAttribute('data-renderer', 'emby')
    expect(screen.getByText('Emby')).toBeInTheDocument()
		expect(screen.getByText('媒体资料由 Emby 提供')).toBeInTheDocument()
  })

  it('uses the default renderer for apps without a dedicated renderer', () => {
    render(<EventCard event={event({ app_code: 'github', display_event_type: 'push', source_event_type: 'push' })} />)
    expect(screen.getByTestId('event-card')).toHaveAttribute('data-renderer', 'default:github')
  })
})
