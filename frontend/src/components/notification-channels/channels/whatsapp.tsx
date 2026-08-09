import { Input } from '@appica/ui-react/input'
import { Message } from '@/components/common/feedback'
import { SecretInput } from '../secret-input'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'

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
  Form: ({ fields, update }) => (
    <>
      <Message variant="warning" title="当前发送普通文本消息" description="需要满足 WhatsApp Cloud API 的会话窗口要求；超出窗口时应使用已审核的消息模板。" />
      <label className="block space-y-2 text-sm font-medium">
        <span>Graph API 版本</span>
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
        <span>收件号码</span>
        <SecretInput revealLabel="收件号码" inputMode="tel" value={textField(fields, 'recipient')} onChange={(event) => update('recipient', event.target.value)} required placeholder="国家码 + 手机号" />
      </label>
    </>
  ),
}
