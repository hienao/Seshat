import { Input } from '@appica/ui-react/input'
import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'
import { useTranslation } from 'react-i18next'
import { i18n } from '@/i18n'

export const wxPusherChannelAdapter: NotificationChannelAdapter = {
  type: 'wxpusher',
  label: 'WxPusher',
  defaultFields: () => ({ appToken: '', uids: '', topicIds: '' }),
  fieldsFromChannel: (channel) => ({ appToken: credentialTextField(channel, 'app_token'), uids: String(channel.config.uids ?? ''), topicIds: String(channel.config.topic_ids ?? '') }),
  toPayload: (common, fields) => {
    const appToken = textField(fields, 'appToken').trim()
    return channelPayload(common, { uids: textField(fields, 'uids').trim(), topic_ids: textField(fields, 'topicIds').trim() }, { app_token: appToken })
  },
  validate: (fields) => (textField(fields, 'uids').trim() || textField(fields, 'topicIds').trim() ? '' : i18n.t('notifications.forms.wxRecipientRequired')),
  Form: ({ fields, update }) => {
    const { t } = useTranslation()
    return <>
      <label className="block space-y-2 text-sm font-medium">
        <span>AppToken</span>
        <SecretInput revealLabel="AppToken" value={textField(fields, 'appToken')} onChange={(event) => update('appToken', event.target.value)} required />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.uidsOptional')}</span>
        <Input value={textField(fields, 'uids')} onChange={(event) => update('uids', event.target.value)} placeholder="UID_xxx, UID_yyy" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.topicsOptional')}</span>
        <Input value={textField(fields, 'topicIds')} onChange={(event) => update('topicIds', event.target.value)} placeholder="123, 456" />
        <span className="block text-xs font-normal text-neutral-500">{t('notifications.forms.wxRecipientHelp')}</span>
      </label>
    </>
  },
}
