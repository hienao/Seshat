import { cleanup, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/api/services'
import { AdminPage } from './admin-page'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
})

function renderAdminPage(publicBaseURL: string) {
  vi.spyOn(api, 'users').mockResolvedValue([])
  vi.spyOn(api, 'systemSettings').mockResolvedValue({
    allow_register: false,
    api_log_retention_days: 7,
    http_proxy_configured: false,
    http_proxy_url: '',
    tmdb_configured: false,
    tmdb_api_key: '',
    tmdb_use_proxy: false,
    public_base_url: publicBaseURL,
  })
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><AdminPage /></QueryClientProvider>)
}

describe('AdminPage public base URL', () => {
  it('does not present the browser origin as a saved setting', async () => {
    const { container } = renderAdminPage('')

    await screen.findByText(/输入框占位地址仅为建议值/)
    const input = container.querySelector<HTMLInputElement>('input[type="url"]')
    expect(input).not.toBeNull()
    expect(input).toHaveValue('')
    expect(input).toHaveAttribute('placeholder', window.location.origin)
    expect(screen.getByText(/输入框占位地址仅为建议值/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '保存访问地址' })).toBeDisabled()
  })

  it('shows the address returned by the server as the saved value', async () => {
    const { container } = renderAdminPage('https://seshat.example.com')

    await screen.findByText(/输入框占位地址仅为建议值/)
    expect(container.querySelector<HTMLInputElement>('input[type="url"]')).toHaveValue('https://seshat.example.com')
  })
})
