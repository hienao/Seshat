import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { booleanField, channelPayload, credentialTextField, textField } from '../types'
import { useTranslation } from 'react-i18next'

export const telegramChannelAdapter: NotificationChannelAdapter = {
  type: 'telegram',
  label: 'Telegram',
  defaultFields: () => ({ botToken: '', chatId: '', threadId: '', silent: false }),
  fieldsFromChannel: (channel) => ({ botToken: credentialTextField(channel, 'bot_token'), chatId: String(channel.config.chat_id ?? ''), threadId: String(channel.config.message_thread_id ?? ''), silent: Boolean(channel.config.silent) }),
  toPayload: (common, fields) => {
    const token = textField(fields, 'botToken').trim()
    const threadId = textField(fields, 'threadId').trim()
    return channelPayload(common, { chat_id: textField(fields, 'chatId').trim(), silent: booleanField(fields, 'silent'), ...(threadId ? { message_thread_id: Number(threadId) } : {}) }, { bot_token: token })
  },
  Form: ({ fields, update }) => {
    const { t } = useTranslation()
    return <>
      <label className="block space-y-2 text-sm font-medium">
        <span>Bot Token</span>
        <SecretInput revealLabel="Bot Token" value={textField(fields, 'botToken')} onChange={(event) => update('botToken', event.target.value)} required />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Chat ID</span>
        <Input value={textField(fields, 'chatId')} onChange={(event) => update('chatId', event.target.value)} required />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.threadId')}</span>
        <Input type="number" value={textField(fields, 'threadId')} onChange={(event) => update('threadId', event.target.value)} />
      </label>
      <div className="flex items-center justify-between gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
        <p className="text-sm font-medium">{t('notifications.forms.silent')}</p>
        <Switch checked={booleanField(fields, 'silent')} onCheckedChange={(checked) => update('silent', checked)} />
      </div>
    </>
  },
}
