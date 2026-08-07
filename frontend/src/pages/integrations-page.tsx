import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Bell, Check, Copy, Link as LinkIcon, Plus, Refresh } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState, Message } from '@/components/common/feedback'
import { FormField } from '@/components/common/form-field'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'
import type { CreatedIntegration } from '@/api/types'

export function IntegrationsPage() {
  const client = useQueryClient()
  const apps = useQuery({ queryKey: ['webhooks', 'apps'], queryFn: api.webhookApps })
  const integrations = useQuery({ queryKey: ['webhooks', 'integrations'], queryFn: api.integrations })
  const [appCode, setAppCode] = useState('')
  const [name, setName] = useState('')
  const [created, setCreated] = useState<CreatedIntegration | null>(null)
  const create = useMutation({ mutationFn: () => api.createIntegration(appCode, name.trim()), onSuccess: (data) => { setCreated(data); setName(''); void client.invalidateQueries({ queryKey: ['webhooks', 'integrations'] }) } })
  const rotate = useMutation({ mutationFn: (id: number) => api.rotateIntegrationSecret(id), onSuccess: setCreated })

  function submit(event: FormEvent) { event.preventDefault(); if (appCode && name.trim()) create.mutate() }
  async function copy(value: string) { await navigator.clipboard?.writeText(value) }

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Webhook Sources" title="接入实例" description="为不同 App 创建独立的 Webhook 地址和签名密钥。" />
      {created && <Message variant="success" title="接入实例已创建" description="Secret 只在本次显示，请立即复制并保存。" />}
      {created && <Panel title="本次生成的凭据" description="Secret 轮换后旧密钥立即失效。"><div className="space-y-4"><div><p className="mb-1 text-xs text-neutral-500">Webhook 地址</p><div className="flex gap-2"><Input readOnly value={`${window.location.origin}${created.webhook_path}`} /><AppButton variant="outline" onClick={() => void copy(`${window.location.origin}${created.webhook_path}`)}><Copy size={16} />复制</AppButton></div></div><div><p className="mb-1 text-xs text-neutral-500">Secret</p><div className="flex gap-2"><Input readOnly value={created.secret} /><AppButton variant="outline" onClick={() => void copy(created.secret)}><Copy size={16} />复制</AppButton></div></div></div></Panel>}
      <Panel title="创建接入" icon={<Plus size={20} />}>
        <form className="grid gap-5 md:grid-cols-[1fr_1fr_auto] md:items-end" onSubmit={(event) => void submit(event)}>
          <FormField label="App 类型"><select className="h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" value={appCode} onChange={(event) => setAppCode(event.target.value)} required><option value="">请选择 App</option>{(apps.data ?? []).map((app) => <option key={app.code} value={app.code}>{app.name}</option>)}</select></FormField>
          <FormField label="接入名称"><Input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：生产环境 GitHub" required maxLength={100} /></FormField>
          <AppButton type="submit" disabled={create.isPending || !appCode || !name.trim()}><Plus size={17} />创建</AppButton>
        </form>
        {create.error && <div className="mt-4"><Message variant="error" title={errorMessage(create.error, '创建接入失败')} /></div>}
      </Panel>
      <Panel title="已有接入" description={`共 ${integrations.data?.length ?? 0} 个`}>
        {integrations.isPending ? <div className="py-10 text-center text-sm text-neutral-500">正在加载接入…</div> : integrations.error ? <ErrorState message={errorMessage(integrations.error)} onRetry={() => void integrations.refetch()} /> : !integrations.data?.length ? <EmptyState title="还没有接入实例" description="选择一个 App 创建你的第一个 Webhook 地址。" /> : <div className="grid gap-4 md:grid-cols-2">{integrations.data.map((item) => <article key={item.id} className="rounded-xl border border-neutral-200 p-5 dark:border-neutral-800"><div className="flex items-start justify-between gap-3"><div><h3 className="font-semibold">{item.name}</h3><p className="mt-1 text-sm text-neutral-500">{item.app_code}</p></div><Badge variant={item.enabled ? 'soft' : 'outline'}>{item.enabled ? '已启用' : '已停用'}</Badge></div><p className="mt-4 break-all rounded-lg bg-neutral-50 p-3 font-mono text-xs text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">{window.location.origin}{item.webhook_path}</p><div className="mt-4 flex flex-wrap gap-2"><AppButton size="sm" variant="outline" disabled={rotate.isPending} onClick={() => rotate.mutate(item.id)}><Refresh size={15} />轮换 Secret</AppButton><AppButton render={<Link to={`/notification-channels?integration=${item.id}`} />} size="sm" variant="outline"><Bell size={15} />通知设置</AppButton></div></article>)}</div>}
      </Panel>
      <div className="flex items-center gap-2 text-xs text-neutral-500"><LinkIcon size={15} />支持通用 Webhook、GitHub 和 Jellyfin；未知事件类型会使用对应 App 的默认类型展示。</div>
    </div>
  )
}
