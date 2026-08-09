import { Input } from '@appica/ui-react/input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, secretLabel, textField } from '../types'

export const serverChanChannelAdapter: NotificationChannelAdapter = {
  type: 'serverchan',
  label: 'Server酱',
  defaultFields: () => ({ sendKey: '' }),
  fieldsFromChannel: () => ({ sendKey: '' }),
  toPayload: (common, fields) => {
    const sendKey = textField(fields, 'sendKey').trim()
    return channelPayload(common, {}, sendKey ? { send_key: sendKey } : undefined)
  },
  Form: ({ fields, keepsExistingCredential, update }) => (
    <label className="block space-y-2 text-sm font-medium">
      <span>{secretLabel('SendKey', keepsExistingCredential)}</span>
      <Input type="password" value={textField(fields, 'sendKey')} onChange={(event) => update('sendKey', event.target.value)} required={!keepsExistingCredential} placeholder="SCT... 或 sctp..." />
    </label>
  ),
}
