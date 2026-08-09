import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { AppDefinition, Integration } from '@/api/types'
import { IntegrationGuideDialog } from './integration-guide-dialog'

afterEach(cleanup)

const integration: Integration = {
  id: 1, owner_id: 7, app_code: 'jellyfin', name: '家庭影院', endpoint_key: 'endpoint', webhook_path: '/hooks/v1/endpoint', config: {}, enabled: true, media_api_configured: false,
  created_at: '2026-08-08T00:00:00Z', updated_at: '2026-08-08T00:00:00Z',
}

function app(code: string, authMode: AppDefinition['auth_mode']): AppDefinition {
  return { code, name: code === 'emby' ? 'Emby' : 'Jellyfin', description: '', auth_mode: authMode, default_event_type: '__default__', event_types: [] }
}

describe('IntegrationGuideDialog', () => {
  it('shows the Jellyfin header and current secret instructions', () => {
    render(<IntegrationGuideDialog integration={integration} app={app('jellyfin', 'secret_header')} secret="current-secret" open onOpenChange={vi.fn()} onRevealSecret={vi.fn()} onCopy={vi.fn()} />)
    expect(screen.getByText('Jellyfin 接入说明')).toBeInTheDocument()
    expect(screen.getAllByText('current-secret').length).toBeGreaterThan(0)
    expect(screen.getByText(/Add Generic Destination/)).toBeInTheDocument()
    expect(screen.getByText(/X-Webhook-Secret: current-secret/)).toBeInTheDocument()
  })

  it('explains that Emby authenticates with the endpoint URL', () => {
    render(<IntegrationGuideDialog integration={{ ...integration, app_code: 'emby' }} app={app('emby', 'endpoint_url')} open onOpenChange={vi.fn()} onRevealSecret={vi.fn()} onCopy={vi.fn()} />)
    expect(screen.getByText('Emby 接入说明')).toBeInTheDocument()
    expect(screen.getByText(/随机 Webhook 地址本身就是接入凭据/)).toBeInTheDocument()
    expect(screen.queryByText('Webhook Secret')).not.toBeInTheDocument()
  })
})
