import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Bell, Book2, Check, Copy, Eye, EyeOff, Key, Link as LinkIcon, Plus, Refresh } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { EmptyState, ErrorState, Message } from '@/components/common/feedback'
import { FormField } from '@/components/common/form-field'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { IntegrationGuideDialog } from '@/components/webhook/integration-guide-dialog'
import { MediaSettingsDialog } from '@/components/webhook/media-settings/media-settings-dialog'
import { mediaSettingsDefinition } from '@/components/webhook/media-settings/registry'
import { copyTextToClipboard } from '@/lib/clipboard'
import { errorMessage } from '@/lib/error-message'
import type { CreatedIntegration, Integration } from '@/api/types'

export function IntegrationsPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const apps = useQuery({ queryKey: ['webhooks', 'apps'], queryFn: api.webhookApps })
  const integrations = useQuery({ queryKey: ['webhooks', 'integrations'], queryFn: api.integrations })
  const [appCode, setAppCode] = useState('')
  const [name, setName] = useState('')
  const [created, setCreated] = useState<CreatedIntegration | null>(null)
  const [credentialAction, setCredentialAction] = useState<'created' | 'rotated' | 'endpoint_rotated'>('created')
  const [secrets, setSecrets] = useState<Record<number, string>>({})
  const [visibleSecrets, setVisibleSecrets] = useState<number[]>([])
  const [guideIntegration, setGuideIntegration] = useState<Integration | null>(null)
  const [mediaSettingsIntegration, setMediaSettingsIntegration] = useState<Integration | null>(null)
  const [copyFeedback, setCopyFeedback] = useState<'success' | 'error' | null>(null)
  const create = useMutation({ mutationFn: () => api.createIntegration(appCode, name.trim()), onSuccess: (data) => { setCreated(data); setCredentialAction('created'); setSecrets((current) => ({ ...current, [data.id]: data.secret })); setVisibleSecrets((current) => [...new Set([...current, data.id])]); setName(''); void client.invalidateQueries({ queryKey: ['webhooks', 'integrations'] }) } })
  const rotate = useMutation({ mutationFn: (id: number) => api.rotateIntegrationSecret(id), onSuccess: (data, id) => { setCreated(data); setCredentialAction(integrations.data?.find((item) => item.id === id)?.app_code === 'emby' ? 'endpoint_rotated' : 'rotated'); setSecrets((current) => ({ ...current, [data.id]: data.secret })); setVisibleSecrets((current) => [...new Set([...current, data.id])]); void client.invalidateQueries({ queryKey: ['webhooks', 'integrations'] }) } })
  const reveal = useMutation({ mutationFn: (id: number) => api.integrationSecret(id), onSuccess: (data, id) => { setSecrets((current) => ({ ...current, [id]: data.secret })); setVisibleSecrets((current) => [...new Set([...current, id])]) } })

  function submit(event: FormEvent) { event.preventDefault(); if (appCode && name.trim()) create.mutate() }
  async function copy(value: string) {
    try {
      await copyTextToClipboard(value)
      setCopyFeedback('success')
    } catch {
      setCopyFeedback('error')
    }
  }
  function toggleSecret(id: number) {
    if (!secrets[id]) { reveal.mutate(id); return }
    setVisibleSecrets((current) => current.includes(id) ? current.filter((item) => item !== id) : [...current, id])
  }
  const guideApp = apps.data?.find((app) => app.code === guideIntegration?.app_code)
  const createdUsesEndpointCredential = created?.app_code === 'emby' || apps.data?.find((app) => app.code === created?.app_code)?.auth_mode === 'endpoint_url'

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow={t('integrations.eyebrow')} title={t('integrations.title')} description={t('integrations.description')} />
      {copyFeedback && <Message variant={copyFeedback} title={t(copyFeedback === 'success' ? 'common.feedback.copySuccess' : 'common.feedback.copyFailed')} />}
      {created && <Message variant="success" title={t(credentialAction === 'created' ? 'integrations.created' : credentialAction === 'endpoint_rotated' ? 'integrations.endpointRotated' : 'integrations.secretRotated')} description={t(credentialAction === 'created' ? createdUsesEndpointCredential ? 'integrations.endpointCredentialDescription' : 'integrations.secretCredentialDescription' : credentialAction === 'endpoint_rotated' ? 'integrations.endpointRotatedDescription' : 'integrations.secretRotatedDescription')} />}
      {created && <Panel title={t(credentialAction === 'created' ? 'integrations.generatedCredential' : 'integrations.rotatedCredential')} description={t(createdUsesEndpointCredential ? 'integrations.endpointIsCredential' : 'integrations.secretAvailable')}><div className="space-y-4"><div><p className="mb-1 text-xs text-neutral-500">{t('integrations.webhookUrl')}</p><div className="flex gap-2"><Input readOnly value={`${window.location.origin}${created.webhook_path}`} /><AppButton variant="outline" onClick={() => void copy(`${window.location.origin}${created.webhook_path}`)}><Copy size={16} />{t('common.actions.copy')}</AppButton></div></div>{!createdUsesEndpointCredential && <div><p className="mb-1 text-xs text-neutral-500">Secret</p><div className="flex gap-2"><Input readOnly value={created.secret} /><AppButton variant="outline" onClick={() => void copy(created.secret)}><Copy size={16} />{t('common.actions.copy')}</AppButton></div></div>}</div></Panel>}
      <Panel title={t('integrations.createTitle')} icon={<Plus size={20} />}>
        <form className="grid gap-5 md:grid-cols-[1fr_1fr_auto] md:items-end" onSubmit={(event) => void submit(event)}>
          <FormField label={t('integrations.appType')}><select className="h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" value={appCode} onChange={(event) => setAppCode(event.target.value)} required><option value="">{t('integrations.selectApp')}</option>{(apps.data ?? []).map((app) => <option key={app.code} value={app.code}>{app.name}</option>)}</select></FormField>
          <FormField label={t('integrations.instanceName')}><Input value={name} onChange={(event) => setName(event.target.value)} placeholder={t('integrations.instancePlaceholder')} required maxLength={100} /></FormField>
          <AppButton type="submit" disabled={create.isPending || !appCode || !name.trim()}><Plus size={17} />{t('common.actions.create')}</AppButton>
        </form>
        {create.error && <div className="mt-4"><Message variant="error" title={errorMessage(create.error, t('integrations.createFailed'))} /></div>}
      </Panel>
      <Panel title={t('integrations.existing')} description={t('common.count', { count: integrations.data?.length ?? 0 })}>
        {integrations.isPending ? <div className="py-10 text-center text-sm text-neutral-500">{t('integrations.loading')}</div> : integrations.error ? <ErrorState message={errorMessage(integrations.error)} onRetry={() => void integrations.refetch()} /> : !integrations.data?.length ? <EmptyState title={t('integrations.empty')} description={t('integrations.emptyDescription')} /> : <div className="grid gap-4 md:grid-cols-2">{integrations.data.map((item) => {
          const secretVisible = visibleSecrets.includes(item.id)
          const appDefinition = apps.data?.find((app) => app.code === item.app_code)
          const usesEndpointCredential = appDefinition?.auth_mode === 'endpoint_url' || item.app_code === 'emby'
          const supportsMediaAPI = Boolean(mediaSettingsDefinition(item.app_code))
          return <article key={item.id} className="rounded-xl border border-neutral-200 p-5 dark:border-neutral-800"><div className="flex items-start justify-between gap-3"><div><h3 className="font-semibold">{item.name}</h3><p className="mt-1 text-sm text-neutral-500">{appDefinition?.name ?? item.app_code}</p>{supportsMediaAPI && <p className="mt-2 text-xs text-neutral-500">{t('integrations.mediaApi')}：{t(item.media_api_configured ? 'integrations.mediaApiConfigured' : 'integrations.mediaApiUnconfigured')}</p>}</div><Badge variant={item.enabled ? 'soft' : 'outline'}>{t(item.enabled ? 'common.states.enabled' : 'common.states.disabled')}</Badge></div><p className="mt-4 break-all rounded-lg bg-neutral-50 p-3 font-mono text-xs text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">{window.location.origin}{item.webhook_path}</p>{secretVisible && secrets[item.id] && !usesEndpointCredential && <div className="mt-3 flex items-start gap-2"><code className="min-w-0 flex-1 break-all rounded-lg bg-neutral-950 p-3 text-xs text-emerald-100">{secrets[item.id]}</code><AppButton size="sm" variant="outline" onClick={() => void copy(secrets[item.id])}><Copy size={15} />{t('common.actions.copy')}</AppButton></div>}<div className="mt-4 flex flex-wrap gap-2">{!usesEndpointCredential && <AppButton size="sm" variant="outline" disabled={reveal.isPending && reveal.variables === item.id} onClick={() => toggleSecret(item.id)}>{secretVisible ? <EyeOff size={15} /> : <Eye size={15} />}{t(reveal.isPending && reveal.variables === item.id ? 'integrations.readingSecret' : secretVisible ? 'integrations.hideSecret' : 'integrations.revealSecret')}</AppButton>}<ConfirmDialog title={t(usesEndpointCredential ? 'integrations.confirmRotateEndpoint' : 'integrations.confirmRotateSecret')} description={t(usesEndpointCredential ? 'integrations.rotateEndpointDescription' : 'integrations.rotateSecretDescription')} confirmLabel={t('integrations.confirmRotate')} destructive busy={rotate.isPending} onConfirm={() => rotate.mutate(item.id)} trigger={<AppButton size="sm" variant="outline" disabled={rotate.isPending}><Refresh size={15} />{t(usesEndpointCredential ? 'integrations.rotateEndpoint' : 'integrations.rotateSecret')}</AppButton>} />{supportsMediaAPI && <AppButton size="sm" variant="outline" onClick={() => setMediaSettingsIntegration(item)}><Key size={15} />{t('integrations.mediaApi')}</AppButton>}<AppButton size="sm" variant="outline" onClick={() => setGuideIntegration(item)}><Book2 size={15} />{t('integrations.setupGuide')}</AppButton><AppButton render={<Link to={`/notification-channels?integration=${item.id}`} />} size="sm" variant="outline"><Bell size={15} />{t('integrations.notificationSettings')}</AppButton></div></article>
        })}</div>}
      </Panel>
      {reveal.error && <Message variant="error" title={errorMessage(reveal.error, t('integrations.readSecretFailed'))} />}
      <div className="flex items-center gap-2 text-xs text-neutral-500"><LinkIcon size={15} />{t('integrations.supports')}</div>
      <IntegrationGuideDialog integration={guideIntegration} app={guideApp} secret={guideIntegration ? secrets[guideIntegration.id] : undefined} secretLoading={reveal.isPending && reveal.variables === guideIntegration?.id} open={Boolean(guideIntegration)} onOpenChange={(open) => { if (!open) setGuideIntegration(null) }} onRevealSecret={() => { if (guideIntegration) reveal.mutate(guideIntegration.id) }} onCopy={(value) => void copy(value)} />
      <MediaSettingsDialog integration={mediaSettingsIntegration} open={Boolean(mediaSettingsIntegration)} onOpenChange={(open) => { if (!open) setMediaSettingsIntegration(null) }} onSaved={() => void client.invalidateQueries({ queryKey: ['webhooks', 'integrations'] })} />
    </div>
  )
}
