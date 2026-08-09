import { Input } from '@appica/ui-react/input'
import { Message } from '@/components/common/feedback'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, secretLabel, textField } from '../types'

export const whatsAppChannelAdapter: NotificationChannelAdapter = {
  type: 'whatsapp',
  label: 'WhatsApp',
  defaultFields: () => ({ version: 'v25.0', token: '', phoneNumberId: '', recipient: '' }),
  fieldsFromChannel: (channel) => ({ version: String(channel.config.api_version ?? 'v25.0'), token: '', phoneNumberId: '', recipient: '' }),
  toPayload: (common, fields) => {
    const token = textField(fields, 'token').trim()
    const phoneNumberId = textField(fields, 'phoneNumberId').trim()
    const recipient = textField(fields, 'recipient').trim()
    return channelPayload(
      common,
      { api_version: textField(fields, 'version').trim() },
      token || phoneNumberId || recipient ? { ...(token ? { access_token: token } : {}), ...(phoneNumberId ? { phone_number_id: phoneNumberId } : {}), ...(recipient ? { recipient } : {}) } : undefined,
    )
  },
  Form: ({ fields, keepsExistingCredential, update }) => (
    <>
      <Message variant="warning" title="当前发送普通文本消息" description="需要满足 WhatsApp Cloud API 的会话窗口要求；超出窗口时应使用已审核的消息模板。" />
      <label className="block space-y-2 text-sm font-medium">
        <span>Graph API 版本</span>
        <Input value={textField(fields, 'version')} onChange={(event) => update('version', event.target.value)} required pattern="v[0-9]+\.[0-9]+" placeholder="v25.0" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('Access Token', keepsExistingCredential)}</span>
        <Input type="password" value={textField(fields, 'token')} onChange={(event) => update('token', event.target.value)} required={!keepsExistingCredential} />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('Phone Number ID', keepsExistingCredential)}</span>
        <Input type="password" inputMode="numeric" value={textField(fields, 'phoneNumberId')} onChange={(event) => update('phoneNumberId', event.target.value)} required={!keepsExistingCredential} />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('收件号码', keepsExistingCredential)}</span>
        <Input type="password" inputMode="tel" value={textField(fields, 'recipient')} onChange={(event) => update('recipient', event.target.value)} required={!keepsExistingCredential} placeholder="国家码 + 手机号" />
      </label>
    </>
  ),
}
