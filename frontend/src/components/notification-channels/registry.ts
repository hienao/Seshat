import type { NotificationChannelType } from '@/api/types'
import { appriseChannelAdapter } from './channels/apprise'
import { barkChannelAdapter } from './channels/bark'
import { dingTalkChannelAdapter } from './channels/dingtalk'
import { emailChannelAdapter } from './channels/email'
import { feishuChannelAdapter } from './channels/feishu'
import { serverChanChannelAdapter } from './channels/serverchan'
import { telegramChannelAdapter } from './channels/telegram'
import { webhookChannelAdapter } from './channels/webhook'
import { whatsAppChannelAdapter } from './channels/whatsapp'
import { wxPusherChannelAdapter } from './channels/wxpusher'
import type { NotificationChannelAdapter } from './types'
import { i18n } from '@/i18n'

const adapters: NotificationChannelAdapter[] = [
  webhookChannelAdapter,
  telegramChannelAdapter,
  appriseChannelAdapter,
  emailChannelAdapter,
  serverChanChannelAdapter,
  barkChannelAdapter,
  dingTalkChannelAdapter,
  feishuChannelAdapter,
  whatsAppChannelAdapter,
  wxPusherChannelAdapter,
]

const adapterRegistry = new Map<NotificationChannelType, NotificationChannelAdapter>()
for (const adapter of adapters) {
  if (adapterRegistry.has(adapter.type)) throw new Error(`Duplicate notification channel adapter: ${adapter.type}`)
  adapterRegistry.set(adapter.type, adapter)
}

export function notificationChannelAdapters(): NotificationChannelAdapter[] {
  return [...adapters]
}

export function notificationChannelAdapter(type: NotificationChannelType): NotificationChannelAdapter {
  const adapter = adapterRegistry.get(type)
  if (!adapter) throw new Error(`Unsupported notification channel adapter: ${type}`)
  return adapter
}

export function notificationChannelLabel(type: NotificationChannelType): string {
  return i18n.t(`notifications.types.${type}`, { defaultValue: notificationChannelAdapter(type).label })
}
