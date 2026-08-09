import { RobotChannelForm } from '../shared/robot-form'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, textField } from '../types'

export const feishuChannelAdapter: NotificationChannelAdapter = {
  type: 'feishu',
  label: 'Feishu',
  defaultFields: () => ({ webhookUrl: '', signingSecret: '' }),
  fieldsFromChannel: () => ({ webhookUrl: '', signingSecret: '' }),
  toPayload: (common, fields) => {
    const webhookUrl = textField(fields, 'webhookUrl').trim()
    const signingSecret = textField(fields, 'signingSecret').trim()
    return channelPayload(common, {}, webhookUrl || signingSecret ? { ...(webhookUrl ? { webhook_url: webhookUrl } : {}), ...(signingSecret ? { signing_secret: signingSecret } : {}) } : undefined)
  },
  Form: (props) => <RobotChannelForm providerName="飞书" {...props} />,
}
