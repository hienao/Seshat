import { RobotChannelForm } from '../shared/robot-form'
import type { NotificationChannelAdapter } from '../types'
import { channelPayload, credentialTextField, textField } from '../types'

export const dingTalkChannelAdapter: NotificationChannelAdapter = {
  type: 'dingtalk',
  label: 'DingTalk',
  defaultFields: () => ({ webhookUrl: '', signingSecret: '' }),
  fieldsFromChannel: (channel) => ({ webhookUrl: credentialTextField(channel, 'webhook_url'), signingSecret: credentialTextField(channel, 'signing_secret') }),
  toPayload: (common, fields) => {
    const webhookUrl = textField(fields, 'webhookUrl').trim()
    const signingSecret = textField(fields, 'signingSecret').trim()
    return channelPayload(common, {}, { webhook_url: webhookUrl, signing_secret: signingSecret })
  },
  Form: (props) => <RobotChannelForm providerName="DingTalk" {...props} />,
}
