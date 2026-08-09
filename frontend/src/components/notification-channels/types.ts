import type { ComponentType } from 'react'
import type { NotificationChannel, NotificationChannelInput, NotificationChannelType } from '@/api/types'

export type ChannelFieldValue = string | boolean
export type ChannelFields = Record<string, ChannelFieldValue>

export interface ChannelCommonValues {
  name: string
  type: NotificationChannelType
  enabled: boolean
  useProxy: boolean
}

export interface ChannelFormProps {
  fields: ChannelFields
  update: (key: string, value: ChannelFieldValue) => void
}

export interface NotificationChannelAdapter {
  type: NotificationChannelType
  label: string
  defaultFields: () => ChannelFields
  fieldsFromChannel: (channel: NotificationChannel) => ChannelFields
  toPayload: (common: ChannelCommonValues, fields: ChannelFields) => NotificationChannelInput
  Form: ComponentType<ChannelFormProps>
  validate?: (fields: ChannelFields) => string
}

export const channelInputClass = 'h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900'

export function textField(fields: ChannelFields, key: string): string {
  const value = fields[key]
  return typeof value === 'string' ? value : ''
}

export function booleanField(fields: ChannelFields, key: string): boolean {
  return fields[key] === true
}

export function credentialTextField(channel: NotificationChannel, key: string): string {
  const value = channel.credentials[key]
  return typeof value === 'string' ? value : ''
}

export function notificationChannelBindingName(channelId: number | undefined, channels: NotificationChannel[], loading: boolean) {
  if (!channelId) return '未绑定渠道'
  return channels.find((channel) => channel.id === channelId)?.name ?? (loading ? '正在加载渠道…' : '绑定渠道不可用')
}

export function channelPayload(common: ChannelCommonValues, config: Record<string, unknown>, credentials?: Record<string, unknown>): NotificationChannelInput {
  return {
    name: common.name.trim(),
    type: common.type,
    enabled: common.enabled,
    config: { ...config, use_proxy: common.useProxy },
    credentials: credentials && Object.keys(credentials).length ? credentials : undefined,
  }
}
