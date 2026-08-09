import { Input } from '@appica/ui-react/input'
import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'

export const barkChannelAdapter: NotificationChannelAdapter = {
  type: 'bark',
  label: 'Bark',
  defaultFields: () => ({ baseUrl: 'https://api.day.app', deviceKey: '', group: 'Seshat', sound: '' }),
  fieldsFromChannel: (channel) => ({ baseUrl: String(channel.config.base_url ?? 'https://api.day.app'), deviceKey: credentialTextField(channel, 'device_key'), group: String(channel.config.group ?? 'Seshat'), sound: String(channel.config.sound ?? '') }),
  toPayload: (common, fields) => {
    const deviceKey = textField(fields, 'deviceKey').trim()
    return channelPayload(common, { base_url: textField(fields, 'baseUrl').trim(), group: textField(fields, 'group').trim(), sound: textField(fields, 'sound').trim() }, { device_key: deviceKey })
  },
  Form: ({ fields, update }) => (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>Bark 服务地址</span>
        <Input type="url" value={textField(fields, 'baseUrl')} onChange={(event) => update('baseUrl', event.target.value)} required placeholder="https://api.day.app" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Device Key</span>
        <SecretInput revealLabel="Device Key" value={textField(fields, 'deviceKey')} onChange={(event) => update('deviceKey', event.target.value)} required />
      </label>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block space-y-2 text-sm font-medium">
          <span>分组（可选）</span>
          <Input value={textField(fields, 'group')} onChange={(event) => update('group', event.target.value)} />
        </label>
        <label className="block space-y-2 text-sm font-medium">
          <span>声音（可选）</span>
          <Input value={textField(fields, 'sound')} onChange={(event) => update('sound', event.target.value)} />
        </label>
      </div>
    </>
  ),
}
