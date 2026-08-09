import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'

export const serverChanChannelAdapter: NotificationChannelAdapter = {
  type: 'serverchan',
  label: 'Server酱',
  defaultFields: () => ({ sendKey: '' }),
  fieldsFromChannel: (channel) => ({ sendKey: credentialTextField(channel, 'send_key') }),
  toPayload: (common, fields) => {
    const sendKey = textField(fields, 'sendKey').trim()
    return channelPayload(common, {}, { send_key: sendKey })
  },
  Form: ({ fields, update }) => (
    <label className="block space-y-2 text-sm font-medium">
      <span>SendKey</span>
      <SecretInput revealLabel="SendKey" value={textField(fields, 'sendKey')} onChange={(event) => update('sendKey', event.target.value)} required placeholder="SCT... 或 sctp..." />
    </label>
  ),
}
