import { Input } from '@appica/ui-react/input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, secretLabel, textField } from '../types'

export const wxPusherChannelAdapter: NotificationChannelAdapter = {
  type: 'wxpusher',
  label: 'WxPusher',
  defaultFields: () => ({ appToken: '', uids: '', topicIds: '' }),
  fieldsFromChannel: (channel) => ({ appToken: '', uids: String(channel.config.uids ?? ''), topicIds: String(channel.config.topic_ids ?? '') }),
  toPayload: (common, fields) => {
    const appToken = textField(fields, 'appToken').trim()
    return channelPayload(common, { uids: textField(fields, 'uids').trim(), topic_ids: textField(fields, 'topicIds').trim() }, appToken ? { app_token: appToken } : undefined)
  },
  validate: (fields) => (textField(fields, 'uids').trim() || textField(fields, 'topicIds').trim() ? '' : 'UIDs 和 Topic IDs 至少填写一项'),
  Form: ({ fields, keepsExistingCredential, update }) => (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('AppToken', keepsExistingCredential)}</span>
        <Input type="password" value={textField(fields, 'appToken')} onChange={(event) => update('appToken', event.target.value)} required={!keepsExistingCredential} />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>UIDs（可选）</span>
        <Input value={textField(fields, 'uids')} onChange={(event) => update('uids', event.target.value)} placeholder="UID_xxx, UID_yyy" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Topic IDs（可选）</span>
        <Input value={textField(fields, 'topicIds')} onChange={(event) => update('topicIds', event.target.value)} placeholder="123, 456" />
        <span className="block text-xs font-normal text-neutral-500">UIDs 和 Topic IDs 至少填写一项。</span>
      </label>
    </>
  ),
}
