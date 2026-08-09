import { SecretInput } from '../secret-input'
import type { ChannelFormProps } from '../types'
import { textField } from '../types'

export function RobotChannelForm({ providerName, fields, update }: ChannelFormProps & { providerName: string }) {
  return (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>{providerName}机器人 Webhook URL</span>
        <SecretInput revealLabel={`${providerName}机器人 Webhook URL`} value={textField(fields, 'webhookUrl')} onChange={(event) => update('webhookUrl', event.target.value)} required autoComplete="off" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>签名密钥（可选）</span>
        <SecretInput revealLabel="签名密钥" value={textField(fields, 'signingSecret')} onChange={(event) => update('signingSecret', event.target.value)} autoComplete="new-password" />
      </label>
    </>
  )
}
