import { Badge } from '@appica/ui-react/badge'
import { Dialog, DialogBody, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@appica/ui-react/dialog'
import { Copy, Key, Link as LinkIcon } from '@appica/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import type { AppDefinition, Integration } from '@/api/types'
import { AppButton } from '@/components/common/app-button'

export function IntegrationGuideDialog({ integration, app, secret, secretLoading, open, onOpenChange, onRevealSecret, onCopy }: {
  integration: Integration | null
  app?: AppDefinition
  secret?: string
  secretLoading?: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  onRevealSecret: () => void
  onCopy: (value: string) => void
}) {
  const { t } = useTranslation()
  if (!integration) return null
  const webhookUrl = `${window.location.origin}${integration.webhook_path}`
  const needsSecret = app?.auth_mode ? app.auth_mode !== 'endpoint_url' : integration.app_code !== 'emby'
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('integrations.guide.title', { app: app?.name ?? integration.app_code })}</DialogTitle>
          <DialogDescription>{t('integrations.guide.description', { name: integration.name })}</DialogDescription>
        </DialogHeader>
        <DialogBody className="max-h-[70vh] space-y-5 overflow-y-auto">
          <GuideValue icon={<LinkIcon size={16} />} label={t('integrations.webhookUrl')} value={webhookUrl} onCopy={() => onCopy(webhookUrl)} />
          {needsSecret ? (
            secret ? <GuideValue icon={<Key size={16} />} label="Webhook Secret" value={secret} onCopy={() => onCopy(secret)} sensitive /> : (
              <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950/40">
                <p className="text-sm font-medium text-amber-900 dark:text-amber-200">{t('integrations.guide.currentSecretRequired')}</p>
                <AppButton className="mt-3" size="sm" variant="outline" disabled={secretLoading} onClick={onRevealSecret}>{t(secretLoading ? 'integrations.readingSecret' : 'integrations.guide.showSecret')}</AppButton>
              </div>
            )
          ) : <p className="rounded-xl bg-sky-50 p-4 text-sm text-sky-800 dark:bg-sky-950/50 dark:text-sky-200">{t('integrations.guide.endpointSensitive')}</p>}
          <GuideSteps appCode={integration.app_code} webhookUrl={webhookUrl} secret={secret} t={t} />
        </DialogBody>
        <DialogFooter><DialogClose render={<AppButton>{t('common.actions.done')}</AppButton>} /></DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function GuideValue({ icon, label, value, sensitive, onCopy }: { icon: ReactNode; label: string; value: string; sensitive?: boolean; onCopy: () => void }) {
  const { t } = useTranslation()
  return (
    <div>
      <p className="mb-2 flex items-center gap-2 text-sm font-medium">{icon}{label}{sensitive && <Badge variant="warning" size="sm">{t('integrations.sensitive')}</Badge>}</p>
      <div className="flex items-start gap-2"><code className="min-w-0 flex-1 break-all rounded-xl bg-neutral-950 p-3 text-xs leading-5 text-emerald-100">{value}</code><AppButton size="sm" variant="outline" onClick={onCopy}><Copy size={15} />{t('common.actions.copy')}</AppButton></div>
    </div>
  )
}

function GuideSteps({ appCode, webhookUrl, secret, t }: { appCode: string; webhookUrl: string; secret?: string; t: TFunction }) {
  const currentSecret = secret || t('integrations.guide.secretAbove')
  const steps = (prefix: string, count: number) => ['one', 'two', 'three', 'four', 'five', 'six'].slice(0, count).map((index) => t(`integrations.guide.${prefix}.${index}`, { url: webhookUrl, secret: currentSecret }))
  if (appCode === 'jellyfin') return <Instruction title={t('integrations.guide.jellyfinTitle')} steps={steps('jellyfinSteps', 6)} />
  if (appCode === 'emby') return <Instruction title={t('integrations.guide.embyTitle')} steps={steps('embySteps', 5)} />
  if (appCode === 'github') return <Instruction title={t('integrations.guide.githubTitle')} steps={steps('githubSteps', 5)} />
  return <Instruction title={t('integrations.guide.genericTitle')} steps={steps('genericSteps', 4)} />
}

function Instruction({ title, steps }: { title: string; steps: string[] }) {
  return <section className="rounded-xl border border-neutral-200 p-4 dark:border-neutral-800"><h3 className="font-semibold">{title}</h3><ol className="mt-3 space-y-3">{steps.map((step, index) => <li key={step} className="flex gap-3 text-sm leading-6 text-neutral-600 dark:text-neutral-300"><span className="grid size-6 shrink-0 place-items-center rounded-full bg-emerald-50 text-xs font-bold text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300">{index + 1}</span><span className="min-w-0 break-words">{step}</span></li>)}</ol></section>
}
