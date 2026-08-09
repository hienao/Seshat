import { Input } from '@appica/ui-react/input'
import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'

const configIDPattern = /^[A-Za-z0-9_-]{1,128}$/

export const appriseChannelAdapter: NotificationChannelAdapter = {
  type: 'apprise',
  label: 'Apprise',
  defaultFields: () => ({ baseUrl: '', configId: '', tag: 'all' }),
  fieldsFromChannel: (channel) => ({ baseUrl: String(channel.config.base_url ?? ''), configId: credentialTextField(channel, 'config_id'), tag: String(channel.config.tag ?? '').trim() || 'all' }),
  toPayload: (common, fields) => {
    const configId = textField(fields, 'configId').trim()
    return channelPayload(common, { base_url: textField(fields, 'baseUrl').trim(), tag: textField(fields, 'tag').trim() }, { config_id: configId })
  },
  validate: (fields) => {
    const configId = textField(fields, 'configId').trim()
    const tag = textField(fields, 'tag').trim()
    if (!configId) return 'Apprise Config ID 不能为空'
    if (configId && !configIDPattern.test(configId)) return 'Apprise Config ID 必须为 1 到 128 位字母、数字、下划线或连字符'
    if (!tag) return 'Apprise Tag 不能为空，请使用 all 推送到全部服务'
    return ''
  },
  Form: ({ fields, update }) => (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>Apprise Base URL</span>
        <Input type="url" value={textField(fields, 'baseUrl')} onChange={(event) => update('baseUrl', event.target.value)} required placeholder="https://apprise.example.com" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Config ID</span>
        <SecretInput revealLabel="Config ID" value={textField(fields, 'configId')} onChange={(event) => update('configId', event.target.value)} required maxLength={128} placeholder="apprise" />
        <span className="block text-xs text-neutral-500">Apprise 中保存通知配置时使用的 Config ID。</span>
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>Tag</span>
        <Input value={textField(fields, 'tag')} onChange={(event) => update('tag', event.target.value)} required placeholder="all" />
        <span className="block text-xs text-neutral-500">填写 all 推送到 Config ID 下的全部服务，也可以填写指定服务的 Tag。</span>
      </label>
    </>
  ),
}
