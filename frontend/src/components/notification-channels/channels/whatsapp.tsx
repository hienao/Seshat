import { Input } from '@appica/ui-react/input'
import { Message } from '@/components/common/feedback'
import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'
import { useTranslation } from 'react-i18next'

export const whatsAppChannelAdapter: NotificationChannelAdapter = {
  type: 'whatsapp',
  label: 'WhatsApp',
  defaultFields: () => ({ version: 'v25.0', token: '', phoneNumberId: '', recipient: '' }),
  fieldsFromChannel: (channel) => ({ version: String(channel.config.api_version ?? 'v25.0'), token: credentialTextField(channel, 'access_token'), phoneNumberId: credentialTextField(channel, 'phone_number_id'), recipient: credentialTextField(channel, 'recipient') }),
  toPayload: (common, fields) => {
    const token = textField(fields, 'token').trim()
    const phoneNumberId = textField(fields, 'phoneNumberId').trim()
    const recipient = textField(fields, 'recipient').trim()
    return channelPayload(common, { api_version: textField(fields, 'version').trim() }, { access_token: token, phone_number_id: phoneNumberId, recipient })
  },
  Form: ({ fields, update }) => {
    const { t } = useTranslation()
    return <>
      <Message variant="warning" title={t('notifications.forms.whatsappWarning')} description={t('notifications.forms.whatsappWarningDescription')} />
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.graphVersion')}</span>
        <Input value={textField(fields, 'version')} onChange={(event) => update('version', event.target.value)} required pattern="v[0-9]+\.[0-9]+" placeholder="v25.0" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Access Token</span>
        <SecretInput revealLabel="Access Token" value={textField(fields, 'token')} onChange={(event) => update('token', event.target.value)} required />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Phone Number ID</span>
        <SecretInput revealLabel="Phone Number ID" inputMode="numeric" value={textField(fields, 'phoneNumberId')} onChange={(event) => update('phoneNumberId', event.target.value)} required />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.recipientNumber')}</span>
        <SecretInput revealLabel={t('notifications.forms.recipientNumber')} inputMode="tel" value={textField(fields, 'recipient')} onChange={(event) => update('recipient', event.target.value)} required placeholder={t('notifications.forms.recipientPlaceholder')} />
      </label>
    </>
  },
}
