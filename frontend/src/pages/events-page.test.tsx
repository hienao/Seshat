import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { api } from '@/api/services'
import type { WebhookEvent } from '@/api/types'
import { EventsPage } from './events-page'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const event: WebhookEvent = {
  id: 1,
  integration_id: 11,
  app_code: 'jellyfin',
  source_event_type: 'PlaybackProgress',
  display_event_type: 'playback_progress',
  external_event_id: '',
  status: 'processed',
  is_fallback: false,
  title: '播放进度',
  summary: '测试消息',
  severity: 'info',
  presentation_version: 1,
  presentation: { schema_version: 1, title: '播放进度', summary: '测试消息', severity: 'info', data: { category: 'playback', event_label: '播放进度', media: { display_name: '测试剧集' } } },
  content_type: 'application/json',
  received_at: '2026-08-09T09:54:47Z',
}

describe('EventsPage', () => {
  it('filters by App integration and its event type, then loads the next page', async () => {
    vi.spyOn(api, 'webhookApps').mockResolvedValue([{
      code: 'jellyfin', name: 'Jellyfin', description: '', auth_mode: 'secret_header', default_event_type: '__default__',
      event_types: [{ code: 'playback_progress', name: '播放进度', render_mode: 'custom' }, { code: '__default__', name: '其他消息', render_mode: 'raw' }],
    }])
    vi.spyOn(api, 'integrations').mockResolvedValue([{
      id: 11, owner_id: 7, app_code: 'jellyfin', name: '客厅影院', endpoint_key: 'living-room', webhook_path: '/hooks/v1/living-room', config: {}, enabled: true,
      created_at: '2026-08-09T00:00:00Z', updated_at: '2026-08-09T00:00:00Z',
    }])
    const events = vi.spyOn(api, 'events').mockImplementation(async (params = {}) => ({
      items: [{ ...event, id: params.offset ? 2 : 1 }], total: 21, limit: 20, offset: params.offset ?? 0, has_more: !params.offset,
    }))
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={queryClient}><MemoryRouter><EventsPage /></MemoryRouter></QueryClientProvider>)

    const appSelect = screen.getByLabelText('App')
    expect(await screen.findByRole('option', { name: '客厅影院 · Jellyfin' })).toBeInTheDocument()
    fireEvent.change(appSelect, { target: { value: '11' } })
    const typeSelect = screen.getByLabelText('消息类型')
    await waitFor(() => expect(typeSelect).not.toBeDisabled())
    fireEvent.change(typeSelect, { target: { value: 'playback_progress' } })
    await waitFor(() => expect(events).toHaveBeenCalledWith(expect.objectContaining({ integrationId: 11, eventType: 'playback_progress', limit: 20, offset: 0 })))

    fireEvent.click(await screen.findByRole('button', { name: '下一页' }))
    await waitFor(() => expect(events).toHaveBeenCalledWith(expect.objectContaining({ integrationId: 11, eventType: 'playback_progress', limit: 20, offset: 20 })))
    expect(await screen.findByText('第 2 / 2 页 · 显示第 21–21 条')).toBeInTheDocument()
  })

  it('shows explicit pagination controls and loads a selected page by offset', async () => {
    vi.spyOn(api, 'webhookApps').mockResolvedValue([])
    vi.spyOn(api, 'integrations').mockResolvedValue([])
    const events = vi.spyOn(api, 'events').mockImplementation(async (params = {}) => ({
      items: [{ ...event, id: (params.offset ?? 0) + 1 }], total: 45, limit: 20, offset: params.offset ?? 0, has_more: (params.offset ?? 0) < 40,
    }))
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={queryClient}><MemoryRouter><EventsPage /></MemoryRouter></QueryClientProvider>)

    expect(await screen.findByRole('navigation', { name: '消息分页' })).toBeInTheDocument()
    expect(screen.getByText('第 1 / 3 页 · 显示第 1–20 条')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '第 3 页' }))

    await waitFor(() => expect(events).toHaveBeenCalledWith(expect.objectContaining({ limit: 20, offset: 40 })))
    expect(await screen.findByText('第 3 / 3 页 · 显示第 41–45 条')).toBeInTheDocument()
  })
})
