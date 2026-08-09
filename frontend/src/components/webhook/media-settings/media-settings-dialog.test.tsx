import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/api/services'
import type { Integration } from '@/api/types'
import { MediaSettingsDialog } from './media-settings-dialog'

afterEach(() => {
	cleanup()
	vi.restoreAllMocks()
})

beforeEach(() => {
	Object.defineProperty(window, 'matchMedia', {
		configurable: true,
		value: vi.fn().mockReturnValue({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }),
	})
})

const integration: Integration = {
	id: 3, owner_id: 7, app_code: 'jellyfin', name: '家庭影院', endpoint_key: 'endpoint', webhook_path: '/hooks/v1/endpoint',
	config: {}, enabled: true, media_api_configured: true, created_at: '2026-08-09T00:00:00Z', updated_at: '2026-08-09T00:00:00Z',
}

describe('MediaSettingsDialog', () => {
	it('loads the original API key, supports reveal, test and save', async () => {
		vi.spyOn(api, 'integrationMediaSettings').mockResolvedValue({ server_url: 'http://jellyfin:8096', api_key: 'original-key', configured: true })
		const test = vi.spyOn(api, 'testIntegrationMediaSettings').mockResolvedValue({ message: '连接成功' })
		const save = vi.spyOn(api, 'updateIntegrationMediaSettings').mockResolvedValue({ server_url: 'http://jellyfin:8096', api_key: 'original-key', configured: true })
		const onSaved = vi.fn()
		const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
		render(<QueryClientProvider client={queryClient}><MediaSettingsDialog integration={integration} open onOpenChange={vi.fn()} onSaved={onSaved} /></QueryClientProvider>)

		const keyInput = await screen.findByLabelText('API Key')
		expect(keyInput).toHaveValue('original-key')
		expect(keyInput).toHaveAttribute('type', 'password')
		fireEvent.click(screen.getByRole('button', { name: '显示 Jellyfin API Key' }))
		expect(keyInput).toHaveAttribute('type', 'text')
		fireEvent.click(screen.getByRole('button', { name: '测试连接' }))
		await waitFor(() => expect(test).toHaveBeenCalledWith(3, { server_url: 'http://jellyfin:8096', api_key: 'original-key' }))
		expect(await screen.findByText('连接成功')).toBeInTheDocument()
		fireEvent.click(screen.getByRole('button', { name: '保存' }))
		await waitFor(() => expect(save).toHaveBeenCalledWith(3, { server_url: 'http://jellyfin:8096', api_key: 'original-key' }))
		expect(onSaved).toHaveBeenCalled()
	})
})
