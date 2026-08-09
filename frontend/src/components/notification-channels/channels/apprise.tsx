import { Input } from '@appica/ui-react/input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, secretLabel, textField } from '../types'

export const appriseChannelAdapter: NotificationChannelAdapter = {
  type: 'apprise',
  label: 'Apprise',
  defaultFields: () => ({ baseUrl: '', configId: '', tag: '' }),
  fieldsFromChannel: (channel) => ({ baseUrl: String(channel.config.base_url ?? ''), configId: '', tag: String(channel.config.tag ?? '') }),
  toPayload: (common, fields) => {
    const configId = textField(fields, 'configId').trim()
    return channelPayload(common, { base_url: textField(fields, 'baseUrl').trim(), tag: textField(fields, 'tag').trim() }, configId ? { config_id: configId } : undefined)
  },
  Form: ({ fields, keepsExistingCredential, update }) => (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>Apprise Base URL</span>
        <Input type="url" value={textField(fields, 'baseUrl')} onChange={(event) => update('baseUrl', event.target.value)} required placeholder="https://apprise.example.com" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('Config ID', keepsExistingCredential)}</span>
        <Input type="password" value={textField(fields, 'configId')} onChange={(event) => update('configId', event.target.value)} required={!keepsExistingCredential} maxLength={128} placeholder="apprise" />
        <span className="block text-xs text-neutral-500">Apprise 中保存通知配置时使用的 Config ID。</span>
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Tag（可选，留空推送到 Config ID 下的全部服务）</span>
        <Input value={textField(fields, 'tag')} onChange={(event) => update('tag', event.target.value)} placeholder="all" />
      </label>
    </>
  ),
}
