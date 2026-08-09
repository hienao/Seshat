import { Input } from '@appica/ui-react/input'
import type { ChannelFormProps } from '../types'
import { secretLabel, textField } from '../types'

export function RobotChannelForm({ providerName, fields, keepsExistingCredential, update }: ChannelFormProps & { providerName: string }) {
  return (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel(`${providerName}机器人 Webhook URL`, keepsExistingCredential)}</span>
        <Input type="password" value={textField(fields, 'webhookUrl')} onChange={(event) => update('webhookUrl', event.target.value)} required={!keepsExistingCredential} autoComplete="off" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('签名密钥（可选）', keepsExistingCredential)}</span>
        <Input type="password" value={textField(fields, 'signingSecret')} onChange={(event) => update('signingSecret', event.target.value)} autoComplete="new-password" />
      </label>
    </>
  )
}
