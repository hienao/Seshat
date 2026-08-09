import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/api/services'
import type { UpdateStatus } from '@/api/types'
import { i18n } from '@/i18n'
import { AppVersionStatus } from './app-version-status'

const status: UpdateStatus = {
  supported: true,
  current: { version: 'v0.7.0', channel: 'beta', commit: 'abc', build_time: '2026-08-09T00:00:00Z' },
  latest_version: 'v0.9.0',
  update_available: true,
  image_tag: 'beta-v0.9.0',
  checked_at: '2026-08-09T10:00:00Z',
  releases: [
    {
      schema_version: 1,
      channel: 'beta',
      version: 'v0.8.0',
      summary: { en: 'English summary', 'zh-CN': '中文摘要' },
      changes: [{ id: 'feature', type: 'feature', text: { en: 'English change', 'zh-CN': '中文更新' } }],
      upgrade_notes: { en: [], 'zh-CN': [] },
      image_tag: 'beta-v0.8.0',
    },
    {
      schema_version: 1,
      channel: 'beta',
      version: 'v0.9.0',
      summary: { en: 'Latest English summary', 'zh-CN': '最新中文摘要' },
      changes: [{ id: 'fix', type: 'fix', text: { en: 'English fix', 'zh-CN': '中文修复' } }],
      upgrade_notes: { en: ['Restart the container.'], 'zh-CN': ['重启容器。'] },
      image_tag: 'beta-v0.9.0',
      release_url: 'https://github.com/hienao/Seshat/releases/tag/beta-v0.9.0',
    },
  ],
}

function renderStatus() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<QueryClientProvider client={client}><AppVersionStatus isAdmin /></QueryClientProvider>)
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('AppVersionStatus', () => {
  it('shows the current version and cumulative beta updates in Chinese', async () => {
    vi.spyOn(api, 'version').mockResolvedValue(status.current)
    vi.spyOn(api, 'updates').mockResolvedValue(status)
    renderStatus()

    expect(await screen.findByText('v0.7.0 · Beta')).toBeInTheDocument()
    expect(await screen.findByText('发现新版本 v0.9.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '打开版本和更新详情' }))
    expect(await screen.findByText('中文摘要')).toBeInTheDocument()
    expect(screen.getByText('最新中文摘要')).toBeInTheDocument()
    expect(screen.getByText('中文更新')).toBeInTheDocument()
    expect(screen.getByText('重启容器。')).toBeInTheDocument()
  })

  it('switches release-note content with the interface language', async () => {
    await i18n.changeLanguage('en')
    vi.spyOn(api, 'version').mockResolvedValue(status.current)
    vi.spyOn(api, 'updates').mockResolvedValue(status)
    renderStatus()

    fireEvent.click(await screen.findByRole('button', { name: 'Open version and update details' }))
    expect(await screen.findByText('English summary')).toBeInTheDocument()
    expect(screen.getByText('Latest English summary')).toBeInTheDocument()
    expect(screen.queryByText('中文摘要')).not.toBeInTheDocument()
  })

  it('does not request remote updates for non-admin users', async () => {
    const updateSpy = vi.spyOn(api, 'updates').mockResolvedValue(status)
    vi.spyOn(api, 'version').mockResolvedValue({ version: 'dev', channel: 'dev', commit: 'unknown', build_time: 'unknown' })
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><AppVersionStatus isAdmin={false} /></QueryClientProvider>)
    expect(await screen.findByText('dev · 开发版')).toBeInTheDocument()
    await waitFor(() => expect(updateSpy).not.toHaveBeenCalled())
    expect(screen.getByText('开发构建 · 不检查更新')).toBeInTheDocument()
  })
})
