import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { useTranslation } from 'react-i18next'
import { SecretInput } from '../secret-input'
import type { ChannelFormProps, NotificationChannelAdapter } from '../types'
import { booleanField, channelInputClass, channelPayload, credentialTextField, textField } from '../types'
import { i18n } from '@/i18n'

export const dingTalkChannelAdapter: NotificationChannelAdapter = {
  type: 'dingtalk',
  label: 'DingTalk',
  defaultFields: () => ({ secret: '', token: '', targets: '', messageFormat: 'auto', includeImage: true }),
  fieldsFromChannel: (channel) => ({
    secret: credentialTextField(channel, 'secret') || credentialTextField(channel, 'signing_secret'),
    token: credentialTextField(channel, 'token') || legacyDingTalkToken(credentialTextField(channel, 'webhook_url')),
    targets: credentialTextField(channel, 'targets'),
    messageFormat: notificationMessageFormat(channel.config.message_format),
    includeImage: channel.config.include_image !== false,
  }),
  toPayload: (common, fields) => {
    const secret = textField(fields, 'secret').trim()
    const token = textField(fields, 'token').trim()
    const targets = textField(fields, 'targets').trim()
    return channelPayload(common, { message_format: notificationMessageFormat(fields.messageFormat), include_image: booleanField(fields, 'includeImage') }, { secret, token, targets })
  },
  validate: (fields) => {
    if (!textField(fields, 'token').trim()) return i18n.t('notifications.forms.dingTalkTokenRequired')
    if (invalidDingTalkTarget(textField(fields, 'targets'))) return i18n.t('notifications.forms.dingTalkTargetsInvalid')
    return ''
  },
  Form: DingTalkChannelForm,
}

function DingTalkChannelForm({ fields, update }: ChannelFormProps) {
  const { t } = useTranslation()
  return (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.dingTalkToken')}</span>
        <SecretInput revealLabel={t('notifications.forms.dingTalkToken')} value={textField(fields, 'token')} onChange={(event) => update('token', event.target.value)} required autoComplete="off" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.messageFormat')}</span>
        <select className={channelInputClass} value={textField(fields, 'messageFormat')} onChange={(event) => update('messageFormat', event.target.value)}>
          <option value="auto">{t('notifications.forms.messageFormatAuto')}</option>
          <option value="markdown">Markdown</option>
          <option value="plain_text">{t('notifications.forms.messageFormatPlainText')}</option>
        </select>
        <span className="block text-xs font-normal text-neutral-500">{t('notifications.forms.messageFormatHelp')}</span>
      </label>
      <div className="flex items-center justify-between gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
        <div>
          <p className="text-sm font-medium">{t('notifications.forms.includeImage')}</p>
          <p className="mt-1 text-xs font-normal text-neutral-500">{t('notifications.forms.includeImageHelp')}</p>
        </div>
        <Switch checked={booleanField(fields, 'includeImage')} onCheckedChange={(checked) => update('includeImage', checked)} />
      </div>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.dingTalkSecret')}</span>
        <SecretInput revealLabel={t('notifications.forms.dingTalkSecret')} value={textField(fields, 'secret')} onChange={(event) => update('secret', event.target.value)} autoComplete="new-password" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.dingTalkTargets')}</span>
        <Input value={textField(fields, 'targets')} onChange={(event) => update('targets', event.target.value)} placeholder="13800138000, 13900139000" />
        <span className="block text-xs font-normal text-neutral-500">{t('notifications.forms.dingTalkTargetsHelp')}</span>
      </label>
    </>
  )
}

function notificationMessageFormat(value: unknown) {
  return value === 'markdown' || value === 'plain_text' ? value : 'auto'
}

function legacyDingTalkToken(webhookURL: string) {
  try {
    return new URL(webhookURL).searchParams.get('access_token')?.trim() ?? ''
  } catch {
    return ''
  }
}

function invalidDingTalkTarget(value: string) {
  return value.split(/[,;\r\n]+/).map((target) => target.trim()).filter(Boolean).some((part) => {
    const target = part.replace(/\D/g, '')
    return target.length < 11 || target.length > 14
  })
}
