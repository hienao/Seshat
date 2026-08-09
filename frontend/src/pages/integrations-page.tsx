import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Bell, Book2, Check, Copy, Eye, EyeOff, Key, Link as LinkIcon, Plus, Refresh } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
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
      <PageHeader eyebrow="Webhook Sources" title="接入实例" description="为不同 App 创建独立的 Webhook 地址和签名密钥。" />
      {copyFeedback && <Message variant={copyFeedback} title={copyFeedback === 'success' ? '已复制到剪贴板' : '复制失败，请手动选中内容复制'} />}
      {created && <Message variant="success" title={credentialAction === 'created' ? '接入实例已创建' : credentialAction === 'endpoint_rotated' ? '接入地址已轮换' : 'Secret 已轮换'} description={credentialAction === 'created' ? createdUsesEndpointCredential ? '请将随机 Webhook 地址视为敏感凭据，可通过接入说明完成配置。' : '可随时在实例卡片中查看 Secret 和接入说明。' : credentialAction === 'endpoint_rotated' ? '旧 Webhook 地址已立即失效，请同步更新 Emby 配置。' : '旧 Secret 已立即失效，请同步更新对应 App 的配置。'} />}
      {created && <Panel title={credentialAction === 'created' ? '本次生成的凭据' : '轮换后的凭据'} description={createdUsesEndpointCredential ? '随机 Webhook 地址本身就是该实例的接入凭据。' : 'Secret 可在接入实例卡片中按需查看。'}><div className="space-y-4"><div><p className="mb-1 text-xs text-neutral-500">Webhook 地址</p><div className="flex gap-2"><Input readOnly value={`${window.location.origin}${created.webhook_path}`} /><AppButton variant="outline" onClick={() => void copy(`${window.location.origin}${created.webhook_path}`)}><Copy size={16} />复制</AppButton></div></div>{!createdUsesEndpointCredential && <div><p className="mb-1 text-xs text-neutral-500">Secret</p><div className="flex gap-2"><Input readOnly value={created.secret} /><AppButton variant="outline" onClick={() => void copy(created.secret)}><Copy size={16} />复制</AppButton></div></div>}</div></Panel>}
      <Panel title="创建接入" icon={<Plus size={20} />}>
        <form className="grid gap-5 md:grid-cols-[1fr_1fr_auto] md:items-end" onSubmit={(event) => void submit(event)}>
          <FormField label="App 类型"><select className="h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" value={appCode} onChange={(event) => setAppCode(event.target.value)} required><option value="">请选择 App</option>{(apps.data ?? []).map((app) => <option key={app.code} value={app.code}>{app.name}</option>)}</select></FormField>
          <FormField label="接入名称"><Input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：生产环境 GitHub" required maxLength={100} /></FormField>
          <AppButton type="submit" disabled={create.isPending || !appCode || !name.trim()}><Plus size={17} />创建</AppButton>
        </form>
        {create.error && <div className="mt-4"><Message variant="error" title={errorMessage(create.error, '创建接入失败')} /></div>}
      </Panel>
      <Panel title="已有接入" description={`共 ${integrations.data?.length ?? 0} 个`}>
        {integrations.isPending ? <div className="py-10 text-center text-sm text-neutral-500">正在加载接入…</div> : integrations.error ? <ErrorState message={errorMessage(integrations.error)} onRetry={() => void integrations.refetch()} /> : !integrations.data?.length ? <EmptyState title="还没有接入实例" description="选择一个 App 创建你的第一个 Webhook 地址。" /> : <div className="grid gap-4 md:grid-cols-2">{integrations.data.map((item) => {
          const secretVisible = visibleSecrets.includes(item.id)
          const appDefinition = apps.data?.find((app) => app.code === item.app_code)
          const usesEndpointCredential = appDefinition?.auth_mode === 'endpoint_url' || item.app_code === 'emby'
          const supportsMediaAPI = Boolean(mediaSettingsDefinition(item.app_code))
          return <article key={item.id} className="rounded-xl border border-neutral-200 p-5 dark:border-neutral-800"><div className="flex items-start justify-between gap-3"><div><h3 className="font-semibold">{item.name}</h3><p className="mt-1 text-sm text-neutral-500">{appDefinition?.name ?? item.app_code}</p>{supportsMediaAPI && <p className="mt-2 text-xs text-neutral-500">媒体 API：{item.media_api_configured ? '已配置' : '未配置，将使用 TMDB'}</p>}</div><Badge variant={item.enabled ? 'soft' : 'outline'}>{item.enabled ? '已启用' : '已停用'}</Badge></div><p className="mt-4 break-all rounded-lg bg-neutral-50 p-3 font-mono text-xs text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">{window.location.origin}{item.webhook_path}</p>{secretVisible && secrets[item.id] && !usesEndpointCredential && <div className="mt-3 flex items-start gap-2"><code className="min-w-0 flex-1 break-all rounded-lg bg-neutral-950 p-3 text-xs text-emerald-100">{secrets[item.id]}</code><AppButton size="sm" variant="outline" onClick={() => void copy(secrets[item.id])}><Copy size={15} />复制</AppButton></div>}<div className="mt-4 flex flex-wrap gap-2">{!usesEndpointCredential && <AppButton size="sm" variant="outline" disabled={reveal.isPending && reveal.variables === item.id} onClick={() => toggleSecret(item.id)}>{secretVisible ? <EyeOff size={15} /> : <Eye size={15} />}{reveal.isPending && reveal.variables === item.id ? '读取中…' : secretVisible ? '隐藏 Secret' : '查看 Secret'}</AppButton>}<ConfirmDialog title={usesEndpointCredential ? '确认轮换接入地址？' : '确认轮换 Secret？'} description={usesEndpointCredential ? '轮换后旧 Webhook 地址会立即失效，需要将新地址更新到 Emby 后才能继续接收消息。' : '轮换后旧 Secret 会立即失效，对应 App 在更新为新 Secret 前将无法继续发送消息。'} confirmLabel="确认轮换" destructive busy={rotate.isPending} onConfirm={() => rotate.mutate(item.id)} trigger={<AppButton size="sm" variant="outline" disabled={rotate.isPending}><Refresh size={15} />{usesEndpointCredential ? '轮换接入地址' : '轮换 Secret'}</AppButton>} />{supportsMediaAPI && <AppButton size="sm" variant="outline" onClick={() => setMediaSettingsIntegration(item)}><Key size={15} />媒体 API</AppButton>}<AppButton size="sm" variant="outline" onClick={() => setGuideIntegration(item)}><Book2 size={15} />接入说明</AppButton><AppButton render={<Link to={`/notification-channels?integration=${item.id}`} />} size="sm" variant="outline"><Bell size={15} />通知设置</AppButton></div></article>
        })}</div>}
      </Panel>
      {reveal.error && <Message variant="error" title={errorMessage(reveal.error, '读取 Secret 失败')} />}
      <div className="flex items-center gap-2 text-xs text-neutral-500"><LinkIcon size={15} />支持通用 Webhook、GitHub、Jellyfin 和 Emby；未知事件类型会使用对应 App 的默认类型展示。</div>
      <IntegrationGuideDialog integration={guideIntegration} app={guideApp} secret={guideIntegration ? secrets[guideIntegration.id] : undefined} secretLoading={reveal.isPending && reveal.variables === guideIntegration?.id} open={Boolean(guideIntegration)} onOpenChange={(open) => { if (!open) setGuideIntegration(null) }} onRevealSecret={() => { if (guideIntegration) reveal.mutate(guideIntegration.id) }} onCopy={(value) => void copy(value)} />
      <MediaSettingsDialog integration={mediaSettingsIntegration} open={Boolean(mediaSettingsIntegration)} onOpenChange={(open) => { if (!open) setMediaSettingsIntegration(null) }} onSaved={() => void client.invalidateQueries({ queryKey: ['webhooks', 'integrations'] })} />
    </div>
  )
}
