import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { NotificationChannel, NotificationChannelType } from '@/api/types'
import { notificationChannelAdapter, notificationChannelAdapters } from './registry'
import { notificationChannelBindingName, type ChannelCommonValues } from './types'

beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({ matches: false, media: query, onchange: null, addEventListener: vi.fn(), removeEventListener: vi.fn(), addListener: vi.fn(), removeListener: vi.fn(), dispatchEvent: vi.fn() })),
  })
})

function common(type: NotificationChannelType): ChannelCommonValues {
  return { name: '测试渠道', type, enabled: true, useProxy: true }
}

function channel(type: NotificationChannelType, config: Record<string, unknown>, credentials: Record<string, unknown>): NotificationChannel {
  return {
    id: 1,
    name: type,
    type,
    enabled: true,
    config,
    credentials,
    has_credentials: Object.keys(credentials).length > 0,
    binding_count: 0,
    created_at: '2026-08-09T00:00:00Z',
    updated_at: '2026-08-09T00:00:00Z',
  }
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

    const apprise = notificationChannelAdapter('apprise').toPayload(common('apprise'), { baseUrl: 'https://apprise.example.com', configId: 'config-main', tag: 'all' })
    expect(apprise.config).toEqual({ base_url: 'https://apprise.example.com', tag: 'all', use_proxy: true })
    expect(apprise.credentials).toEqual({ config_id: 'config-main' })

    const dingTalk = notificationChannelAdapter('dingtalk').toPayload(common('dingtalk'), { secret: 'SEC-test', token: 'ding-token', targets: '13800138000, 13900139000', messageFormat: 'markdown', includeImage: false })
    expect(dingTalk.config).toEqual({ message_format: 'markdown', include_image: false, use_proxy: true })
    expect(dingTalk.credentials).toEqual({ secret: 'SEC-test', token: 'ding-token', targets: '13800138000, 13900139000' })
    expect(dingTalk.credentials).not.toHaveProperty('webhook_url')
  })

  it('hydrates only fields owned by the selected adapter', () => {
    const channel: NotificationChannel = {
      id: 1,
      name: 'Apprise',
      type: 'apprise',
      enabled: true,
      config: { base_url: 'https://apprise.example.com', tag: 'media', chat_id: 'should-not-leak' },
      credentials: { config_id: 'config-main', bot_token: 'should-not-leak' },
      has_credentials: true,
      binding_count: 0,
      created_at: '2026-08-09T00:00:00Z',
      updated_at: '2026-08-09T00:00:00Z',
    }
    expect(notificationChannelAdapter('apprise').fieldsFromChannel(channel)).toEqual({ baseUrl: 'https://apprise.example.com', configId: 'config-main', tag: 'media' })
    expect(notificationChannelBindingName(channel.id, [channel], false)).toBe('Apprise')
    expect(notificationChannelBindingName(undefined, [channel], false)).toBe('未绑定渠道')
  })

  it('hydrates every channel credential into its own adapter fields', () => {
    expect(notificationChannelAdapter('webhook').fieldsFromChannel(channel('webhook', {}, { url: 'https://example.com/hook', headers: { Authorization: 'Bearer token' } }))).toEqual({ url: 'https://example.com/hook', headers: '{\n  "Authorization": "Bearer token"\n}' })
    expect(notificationChannelAdapter('telegram').fieldsFromChannel(channel('telegram', { chat_id: '42' }, { bot_token: 'telegram-token' })).botToken).toBe('telegram-token')
    expect(notificationChannelAdapter('email').fieldsFromChannel(channel('email', {}, { username: 'mailer', password: 'mail-password' }))).toMatchObject({ username: 'mailer', password: 'mail-password' })
    expect(notificationChannelAdapter('serverchan').fieldsFromChannel(channel('serverchan', {}, { send_key: 'SCT-key' })).sendKey).toBe('SCT-key')
    expect(notificationChannelAdapter('bark').fieldsFromChannel(channel('bark', {}, { device_key: 'device-key' })).deviceKey).toBe('device-key')
    expect(notificationChannelAdapter('dingtalk').fieldsFromChannel(channel('dingtalk', { message_format: 'plain_text', include_image: false }, { secret: 'SEC-ding', token: 'ding-token', targets: '13800138000' }))).toEqual({ secret: 'SEC-ding', token: 'ding-token', targets: '13800138000', messageFormat: 'plain_text', includeImage: false })
    expect(notificationChannelAdapter('dingtalk').fieldsFromChannel(channel('dingtalk', {}, { webhook_url: 'https://oapi.dingtalk.com/robot/send?access_token=legacy-token', signing_secret: 'legacy-secret' }))).toEqual({ secret: 'legacy-secret', token: 'legacy-token', targets: '', messageFormat: 'auto', includeImage: true })
    expect(notificationChannelAdapter('feishu').fieldsFromChannel(channel('feishu', {}, { webhook_url: 'https://feishu.example/hook', signing_secret: 'feishu-secret' }))).toEqual({ webhookUrl: 'https://feishu.example/hook', signingSecret: 'feishu-secret' })
    expect(notificationChannelAdapter('whatsapp').fieldsFromChannel(channel('whatsapp', {}, { access_token: 'wa-token', phone_number_id: '123', recipient: '86138' }))).toMatchObject({ token: 'wa-token', phoneNumberId: '123', recipient: '86138' })
    expect(notificationChannelAdapter('wxpusher').fieldsFromChannel(channel('wxpusher', {}, { app_token: 'wx-token' })).appToken).toBe('wx-token')
  })

  it('requires a valid Apprise Config ID', () => {
    const adapter = notificationChannelAdapter('apprise')
    const fields = { baseUrl: 'https://apprise.example.com', configId: '', tag: '' }
    expect(adapter.validate?.(fields)).toBe('Apprise Config ID 不能为空')
    expect(adapter.validate?.({ ...fields, configId: 'invalid id' })).toContain('字母、数字')
    expect(adapter.validate?.({ ...fields, configId: 'config-main' })).toContain('Tag 不能为空')
    expect(adapter.validate?.({ ...fields, configId: 'config-main', tag: 'all' })).toBe('')
    expect(adapter.defaultFields().tag).toBe('all')
    expect(adapter.fieldsFromChannel(channel('apprise', { base_url: 'https://apprise.example.com', tag: '' }, { config_id: 'config-main' })).tag).toBe('all')
  })

  it('requires a DingTalk token and validates optional targets', () => {
    const adapter = notificationChannelAdapter('dingtalk')
    expect(adapter.validate?.({ secret: '', token: '', targets: '' })).toBe('钉钉 Token 不能为空')
    expect(adapter.validate?.({ secret: '', token: 'token', targets: 'invalid' })).toContain('11 到 14 位手机号')
    expect(adapter.validate?.({ secret: '', token: 'token', targets: '13800138000, +86 13900139000' })).toBe('')
  })

  it('renders only the selected channel form', () => {
    const WebhookForm = notificationChannelAdapter('webhook').Form
    render(<WebhookForm fields={notificationChannelAdapter('webhook').defaultFields()} update={vi.fn()} />)
    expect(screen.getByText('目标 URL')).toBeInTheDocument()
    expect(screen.queryByText('Bark 服务地址')).not.toBeInTheDocument()
  })

  it('renders DingTalk token, secret, and targets fields', () => {
    const DingTalkForm = notificationChannelAdapter('dingtalk').Form
    render(<DingTalkForm fields={notificationChannelAdapter('dingtalk').defaultFields()} update={vi.fn()} />)
    expect(screen.getByText('钉钉 Token')).toBeInTheDocument()
    expect(screen.getByText('钉钉 Secret（可选）')).toBeInTheDocument()
    expect(screen.getByText('Targets（可选）')).toBeInTheDocument()
    expect(screen.getByText('消息格式')).toBeInTheDocument()
    expect(screen.getByText('附带媒体图片')).toBeInTheDocument()
    expect(screen.queryByText('DingTalk机器人 Webhook URL')).not.toBeInTheDocument()
  })

  it('conceals hydrated channel credentials until the reveal button is clicked', () => {
    const AppriseForm = notificationChannelAdapter('apprise').Form
    render(<AppriseForm fields={{ baseUrl: 'https://apprise.example.com', configId: 'config-main', tag: 'all' }} update={vi.fn()} />)
    const input = screen.getByPlaceholderText('apprise')
    expect(input).toHaveAttribute('type', 'password')
    expect(screen.getByDisplayValue('all')).toBeRequired()
    fireEvent.click(screen.getByRole('button', { name: '显示 Config ID' }))
    expect(input).toHaveAttribute('type', 'text')
  })
})
