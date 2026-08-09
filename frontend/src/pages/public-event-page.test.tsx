import { cleanup, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { api } from '@/api/services'
import type { DisplayWebhookEvent } from '@/api/types'
import { PublicEventPage } from './public-event-page'

beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({ matches: false, media: query, onchange: null, addEventListener: vi.fn(), removeEventListener: vi.fn(), addListener: vi.fn(), removeListener: vi.fn(), dispatchEvent: vi.fn() })),
  })
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const publicEvent: DisplayWebhookEvent = {
  app_code: 'jellyfin', source_event_type: 'PlaybackProgress', display_event_type: 'playback_progress', is_fallback: false,
  title: 'Jellyfin · 播放进度', summary: 'mark · Jellyfin Web', severity: 'info', presentation_version: 1,
  presentation: { schema_version: 1, title: 'Jellyfin · 播放进度', summary: 'mark · Jellyfin Web', severity: 'info', data: { category: 'playback', event_label: '播放进度', media: { display_name: 'Severance · S01E01 · Pilot' }, playback: { percent: 25 } } },
  received_at: '2026-08-09T00:00:00Z',
}

describe('PublicEventPage', () => {
  it('renders only the standardized event card without authenticated page sections', async () => {
    vi.spyOn(api, 'publicEvent').mockResolvedValue(publicEvent)
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={['/public/events/abcdefghijklmnopqrstuvwxyz123456']}>
          <Routes><Route path="/public/events/:token" element={<PublicEventPage />} /></Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(await screen.findByText('Severance · S01E01 · Pilot')).toBeInTheDocument()
    expect(screen.queryByText('原始消息')).not.toBeInTheDocument()
    expect(screen.queryByText('推送状态')).not.toBeInTheDocument()
    expect(screen.queryByText('登录')).not.toBeInTheDocument()
  })
})
