import { SecretInput, SecretTextarea } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'

export const webhookChannelAdapter: NotificationChannelAdapter = {
  type: 'webhook',
  label: 'Webhook',
  defaultFields: () => ({ url: '', headers: '' }),
  fieldsFromChannel: (channel) => ({
    url: credentialTextField(channel, 'url'),
    headers: channel.credentials.headers && typeof channel.credentials.headers === 'object' ? JSON.stringify(channel.credentials.headers, null, 2) : '',
  }),
  toPayload: (common, fields) => {
    let headers: Record<string, string> = {}
    if (textField(fields, 'headers').trim()) headers = JSON.parse(textField(fields, 'headers')) as Record<string, string>
    const url = textField(fields, 'url').trim()
    return channelPayload(common, {}, { ...(url ? { url } : {}), ...(Object.keys(headers).length ? { headers } : {}) })
  },
  Form: ({ fields, update }) => (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>目标 URL</span>
        <SecretInput revealLabel="目标 URL" value={textField(fields, 'url')} onChange={(event) => update('url', event.target.value)} required placeholder="https://example.com/notify" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>自定义请求头 JSON（可选）</span>
        <SecretTextarea
          revealLabel="自定义请求头"
          className="min-h-28 w-full rounded-lg border border-neutral-300 bg-white p-3 font-mono text-xs dark:border-neutral-700 dark:bg-neutral-900"
          value={textField(fields, 'headers')}
          onChange={(event) => update('headers', event.target.value)}
          placeholder={'{"Authorization":"Bearer ..."}'}
        />
      </label>
    </>
  ),
}
