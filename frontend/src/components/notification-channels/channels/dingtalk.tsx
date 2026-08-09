import { RobotChannelForm } from '../shared/robot-form'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, textField } from '../types'

export const dingTalkChannelAdapter: NotificationChannelAdapter = {
  type: 'dingtalk',
  label: 'DingTalk',
  defaultFields: () => ({ webhookUrl: '', signingSecret: '' }),
  fieldsFromChannel: () => ({ webhookUrl: '', signingSecret: '' }),
  toPayload: (common, fields) => {
    const webhookUrl = textField(fields, 'webhookUrl').trim()
    const signingSecret = textField(fields, 'signingSecret').trim()
    return channelPayload(common, {}, webhookUrl || signingSecret ? { ...(webhookUrl ? { webhook_url: webhookUrl } : {}), ...(signingSecret ? { signing_secret: signingSecret } : {}) } : undefined)
  },
  Form: (props) => <RobotChannelForm providerName="钉钉" {...props} />,
}
