import { SecretInput } from '../secret-input'
import type { ChannelFormProps } from '../types'
import { textField } from '../types'
import { useTranslation } from 'react-i18next'

export function RobotChannelForm({ providerName, fields, update }: ChannelFormProps & { providerName: string }) {
  const { t } = useTranslation()
  return (
    <>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.robotUrl', { provider: providerName })}</span>
        <SecretInput revealLabel={t('notifications.forms.robotUrl', { provider: providerName })} value={textField(fields, 'webhookUrl')} onChange={(event) => update('webhookUrl', event.target.value)} required autoComplete="off" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{t('notifications.forms.signingSecret')}</span>
        <SecretInput revealLabel={t('notifications.forms.signingSecret')} value={textField(fields, 'signingSecret')} onChange={(event) => update('signingSecret', event.target.value)} autoComplete="new-password" />
      </label>
    </>
  )
}
