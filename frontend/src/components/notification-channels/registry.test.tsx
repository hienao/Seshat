import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { NotificationChannel, NotificationChannelType } from '@/api/types'
import { notificationChannelAdapter, notificationChannelAdapters } from './registry'
import type { ChannelCommonValues } from './types'

function common(type: NotificationChannelType): ChannelCommonValues {
  return { name: '测试渠道', type, enabled: true, useProxy: true }
}

describe('notification channel adapter registry', () => {
  it('registers every channel type exactly once', () => {
    const types = notificationChannelAdapters().map((adapter) => adapter.type)
    expect(types).toEqual(['webhook', 'telegram', 'apprise', 'email', 'serverchan', 'bark', 'dingtalk', 'feishu', 'whatsapp', 'wxpusher'])
    expect(new Set(types).size).toBe(types.length)
  })

  it('keeps channel payload fields inside their own adapters', () => {
    const telegram = notificationChannelAdapter('telegram').toPayload(common('telegram'), { botToken: 'token', chatId: '42', threadId: '', silent: false })
    expect(telegram.config).toEqual({ chat_id: '42', silent: false, use_proxy: true })
    expect(telegram.credentials).toEqual({ bot_token: 'token' })
    expect(telegram.config).not.toHaveProperty('base_url')

    const bark = notificationChannelAdapter('bark').toPayload(common('bark'), { baseUrl: 'https://api.day.app', deviceKey: 'device', group: 'Seshat', sound: '' })
    expect(bark.config).toEqual({ base_url: 'https://api.day.app', group: 'Seshat', sound: '', use_proxy: true })
    expect(bark.credentials).toEqual({ device_key: 'device' })
    expect(bark.credentials).not.toHaveProperty('bot_token')
  })

  it('hydrates only fields owned by the selected adapter', () => {
    const channel: NotificationChannel = {
      id: 1,
      name: 'Apprise',
      type: 'apprise',
      enabled: true,
      config: { base_url: 'https://apprise.example.com', tag: 'media', chat_id: 'should-not-leak' },
      has_credentials: true,
      binding_count: 0,
      created_at: '2026-08-09T00:00:00Z',
      updated_at: '2026-08-09T00:00:00Z',
    }
    expect(notificationChannelAdapter('apprise').fieldsFromChannel(channel)).toEqual({ baseUrl: 'https://apprise.example.com', configId: '', tag: 'media' })
  })

  it('renders only the selected channel form', () => {
    const WebhookForm = notificationChannelAdapter('webhook').Form
    render(<WebhookForm fields={notificationChannelAdapter('webhook').defaultFields()} keepsExistingCredential={false} update={vi.fn()} />)
    expect(screen.getByText('目标 URL')).toBeInTheDocument()
    expect(screen.queryByText('Bark 服务地址')).not.toBeInTheDocument()
  })
})
