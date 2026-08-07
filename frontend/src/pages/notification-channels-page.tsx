import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { AlertCircle, Bell, Check, Plus, Refresh, Settings, Trash } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState, type Dispatch, type FormEvent, type SetStateAction } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '@/api/services'
import type { Integration, NotificationChannel, NotificationChannelInput, NotificationChannelType } from '@/api/types'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState, Message } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'

const inputClass = 'h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900'
const channelLabels: Record<NotificationChannelType, string> = {
  webhook: 'Webhook',
  telegram: 'Telegram',
  apprise: 'Apprise',
  email: '邮箱通知',
  serverchan: 'Server酱',
  bark: 'Bark',
  dingtalk: 'DingTalk',
  feishu: 'Feishu',
  whatsapp: 'WhatsApp',
  wxpusher: 'WxPusher',
}

type ChannelFormState = {
  name: string
  type: NotificationChannelType
  enabled: boolean
  useProxy: boolean
  url: string
  headers: string
  botToken: string
  chatId: string
  threadId: string
  silent: boolean
  baseUrl: string
  appriseMode: 'stateful' | 'stateless'
  appriseKey: string
  appriseUrls: string
  tag: string
  smtpHost: string
  smtpPort: string
  smtpEncryption: 'starttls' | 'tls' | 'none'
  smtpFrom: string
  smtpTo: string
  smtpUsername: string
  smtpPassword: string
  serverChanSendKey: string
  barkBaseUrl: string
  barkDeviceKey: string
  barkGroup: string
  barkSound: string
  robotWebhookUrl: string
  robotSigningSecret: string
  whatsappVersion: string
  whatsappToken: string
  whatsappPhoneNumberId: string
  whatsappRecipient: string
  wxPusherAppToken: string
  wxPusherUids: string
  wxPusherTopicIds: string
}

function emptyChannelForm(): ChannelFormState {
  return {
    name: '',
    type: 'webhook',
    enabled: false,
    useProxy: false,
    url: '',
    headers: '',
    botToken: '',
    chatId: '',
    threadId: '',
    silent: false,
    baseUrl: '',
    appriseMode: 'stateful',
    appriseKey: '',
    appriseUrls: '',
    tag: '',
    smtpHost: '',
    smtpPort: '587',
    smtpEncryption: 'starttls',
    smtpFrom: '',
    smtpTo: '',
    smtpUsername: '',
    smtpPassword: '',
    serverChanSendKey: '',
    barkBaseUrl: 'https://api.day.app',
    barkDeviceKey: '',
    barkGroup: 'Seshat',
    barkSound: '',
    robotWebhookUrl: '',
    robotSigningSecret: '',
    whatsappVersion: 'v25.0',
    whatsappToken: '',
    whatsappPhoneNumberId: '',
    whatsappRecipient: '',
    wxPusherAppToken: '',
    wxPusherUids: '',
    wxPusherTopicIds: '',
  }
}

function formFromChannel(channel: NotificationChannel): ChannelFormState {
  return {
    ...emptyChannelForm(),
    name: channel.name,
    type: channel.type,
    enabled: channel.enabled,
    useProxy: Boolean(channel.config.use_proxy),
    chatId: String(channel.config.chat_id ?? ''),
    threadId: String(channel.config.message_thread_id ?? ''),
    silent: Boolean(channel.config.silent),
    baseUrl: String(channel.config.base_url ?? ''),
    appriseMode: channel.config.mode === 'stateless' ? 'stateless' : 'stateful',
    tag: String(channel.config.tag ?? ''),
    smtpHost: String(channel.config.smtp_host ?? ''),
    smtpPort: String(channel.config.smtp_port ?? '587'),
    smtpEncryption: channel.config.encryption === 'tls' || channel.config.encryption === 'none' ? channel.config.encryption : 'starttls',
    smtpFrom: String(channel.config.from ?? ''),
    smtpTo: String(channel.config.to ?? ''),
    barkBaseUrl: String(channel.config.base_url ?? 'https://api.day.app'),
    barkGroup: String(channel.config.group ?? 'Seshat'),
    barkSound: String(channel.config.sound ?? ''),
    whatsappVersion: String(channel.config.api_version ?? 'v25.0'),
    wxPusherUids: String(channel.config.uids ?? ''),
    wxPusherTopicIds: String(channel.config.topic_ids ?? ''),
  }
}

function channelPayload(form: ChannelFormState): NotificationChannelInput {
  if (form.type === 'webhook') {
    let headers: Record<string, string> = {}
    if (form.headers.trim()) headers = JSON.parse(form.headers) as Record<string, string>
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: { use_proxy: form.useProxy },
      credentials:
        form.url.trim() || Object.keys(headers).length
          ? {
              ...(form.url.trim() ? { url: form.url.trim() } : {}),
              ...(Object.keys(headers).length ? { headers } : {}),
            }
          : undefined,
    }
  }
  if (form.type === 'telegram') {
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: {
        chat_id: form.chatId.trim(),
        silent: form.silent,
        use_proxy: form.useProxy,
        ...(form.threadId.trim() ? { message_thread_id: Number(form.threadId) } : {}),
      },
      credentials: form.botToken.trim() ? { bot_token: form.botToken.trim() } : undefined,
    }
  }
  if (form.type === 'apprise')
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: {
        base_url: form.baseUrl.trim(),
        mode: form.appriseMode,
        tag: form.tag.trim(),
        use_proxy: form.useProxy,
      },
      credentials: form.appriseMode === 'stateless' ? (form.appriseUrls.trim() ? { urls: form.appriseUrls.trim() } : undefined) : form.appriseKey.trim() ? { key: form.appriseKey.trim() } : undefined,
    }
  if (form.type === 'email')
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: {
        smtp_host: form.smtpHost.trim(),
        smtp_port: Number(form.smtpPort),
        encryption: form.smtpEncryption,
        from: form.smtpFrom.trim(),
        to: form.smtpTo.trim(),
        use_proxy: form.useProxy,
      },
      credentials:
        form.smtpUsername.trim() || form.smtpPassword
          ? {
              ...(form.smtpUsername.trim() ? { username: form.smtpUsername.trim() } : {}),
              ...(form.smtpPassword ? { password: form.smtpPassword } : {}),
            }
          : undefined,
    }
  if (form.type === 'serverchan')
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: { use_proxy: form.useProxy },
      credentials: form.serverChanSendKey.trim() ? { send_key: form.serverChanSendKey.trim() } : undefined,
    }
  if (form.type === 'bark')
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: {
        base_url: form.barkBaseUrl.trim(),
        group: form.barkGroup.trim(),
        sound: form.barkSound.trim(),
        use_proxy: form.useProxy,
      },
      credentials: form.barkDeviceKey.trim() ? { device_key: form.barkDeviceKey.trim() } : undefined,
    }
  if (form.type === 'dingtalk' || form.type === 'feishu')
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: { use_proxy: form.useProxy },
      credentials:
        form.robotWebhookUrl.trim() || form.robotSigningSecret.trim()
          ? {
              ...(form.robotWebhookUrl.trim() ? { webhook_url: form.robotWebhookUrl.trim() } : {}),
              ...(form.robotSigningSecret.trim() ? { signing_secret: form.robotSigningSecret.trim() } : {}),
            }
          : undefined,
    }
  if (form.type === 'whatsapp')
    return {
      name: form.name.trim(),
      type: form.type,
      enabled: form.enabled,
      config: {
        api_version: form.whatsappVersion.trim(),
        use_proxy: form.useProxy,
      },
      credentials:
        form.whatsappToken.trim() || form.whatsappPhoneNumberId.trim() || form.whatsappRecipient.trim()
          ? {
              ...(form.whatsappToken.trim() ? { access_token: form.whatsappToken.trim() } : {}),
              ...(form.whatsappPhoneNumberId.trim() ? { phone_number_id: form.whatsappPhoneNumberId.trim() } : {}),
              ...(form.whatsappRecipient.trim() ? { recipient: form.whatsappRecipient.trim() } : {}),
            }
          : undefined,
    }
  return {
    name: form.name.trim(),
    type: form.type,
    enabled: form.enabled,
    config: {
      uids: form.wxPusherUids.trim(),
      topic_ids: form.wxPusherTopicIds.trim(),
      use_proxy: form.useProxy,
    },
    credentials: form.wxPusherAppToken.trim() ? { app_token: form.wxPusherAppToken.trim() } : undefined,
  }
}

export function NotificationChannelsPage() {
  const client = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const channels = useQuery({
    queryKey: ['notifications', 'channels'],
    queryFn: api.notificationChannels,
  })
  const integrations = useQuery({
    queryKey: ['webhooks', 'integrations'],
    queryFn: api.integrations,
  })
  const deliveries = useQuery({
    queryKey: ['notifications', 'deliveries'],
    queryFn: () => api.notificationDeliveries(),
  })
  const [editing, setEditing] = useState<NotificationChannel | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [form, setForm] = useState<ChannelFormState>(emptyChannelForm)
  const [formError, setFormError] = useState('')
  const [selectedIntegration, setSelectedIntegration] = useState<Integration | null>(null)
  useEffect(() => {
    const integrationId = Number(searchParams.get('integration'))
    if (!selectedIntegration && integrationId > 0 && integrations.data) {
      setSelectedIntegration(integrations.data.find((item) => item.id === integrationId) ?? null)
    }
  }, [integrations.data, searchParams, selectedIntegration])

  const save = useMutation({
    mutationFn: (payload: NotificationChannelInput) => (editing ? api.updateNotificationChannel(editing.id, payload) : api.createNotificationChannel(payload)),
    onSuccess: () => {
      closeForm()
      void client.invalidateQueries({
        queryKey: ['notifications', 'channels'],
      })
    },
  })
  const test = useMutation({
    mutationFn: api.testNotificationChannel,
    onSuccess: () =>
      void client.invalidateQueries({
        queryKey: ['notifications', 'channels'],
      }),
  })
  const remove = useMutation({
    mutationFn: api.deleteNotificationChannel,
    onSuccess: () =>
      void client.invalidateQueries({
        queryKey: ['notifications', 'channels'],
      }),
  })
  const retry = useMutation({
    mutationFn: api.retryNotificationDelivery,
    onSuccess: () =>
      void client.invalidateQueries({
        queryKey: ['notifications', 'deliveries'],
      }),
  })

  function openCreate() {
    setEditing(null)
    setForm(emptyChannelForm())
    setFormError('')
    setFormOpen(true)
  }
  function openEdit(channel: NotificationChannel) {
    setEditing(channel)
    setForm(formFromChannel(channel))
    setFormError('')
    setFormOpen(true)
  }
  function closeForm() {
    setFormOpen(false)
    setEditing(null)
    setForm(emptyChannelForm())
    setFormError('')
  }
  function submit(event: FormEvent) {
    event.preventDefault()
    setFormError('')
    try {
      save.mutate(channelPayload(form))
    } catch {
      setFormError('自定义请求头必须是 JSON 对象')
    }
  }

  return (
    <div className="mx-auto max-w-7xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader
        eyebrow="Notification Routing"
        title="推送渠道"
        description="配置外部通知目标，并为每个接入实例选择需要推送的消息类型。"
        action={
          <AppButton size="sm" onClick={openCreate}>
            <Plus size={16} />
            新增渠道
          </AppButton>
        }
      />
      {(save.error || test.error || remove.error || retry.error) && <Message variant="error" title={errorMessage(save.error || test.error || remove.error || retry.error, '推送操作失败')} />}

      <Panel title="渠道管理" description={`共 ${channels.data?.length ?? 0} 个渠道`} icon={<Bell size={20} />}>
        {channels.isPending ? (
          <p className="py-10 text-center text-sm text-neutral-500">正在加载渠道…</p>
        ) : channels.error ? (
          <ErrorState message={errorMessage(channels.error)} onRetry={() => void channels.refetch()} />
        ) : !channels.data?.length ? (
          <EmptyState title="还没有推送渠道" description="支持 Webhook、聊天机器人、邮件和移动推送等渠道，新增后即可绑定到接入实例。" />
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {channels.data.map((channel) => (
              <article key={channel.id} className="rounded-xl border border-neutral-200 p-5 dark:border-neutral-800">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h3 className="font-semibold">{channel.name}</h3>
                    <p className="mt-1 text-xs text-neutral-500">
                      {channelLabels[channel.type]} · {channel.binding_count} 个实例{channel.config.use_proxy ? ' · 使用系统代理' : ''}
                    </p>
                  </div>
                  <Badge variant={channel.enabled ? 'success' : 'outline'}>{channel.enabled ? '已启用' : '已停用'}</Badge>
                </div>
                <div className="mt-4 flex items-center gap-2 text-xs text-neutral-500">
                  {channel.last_test_status === 'succeeded' ? (
                    <>
                      <Check size={15} className="text-emerald-600" />
                      最近测试成功
                    </>
                  ) : channel.last_test_status === 'failed' ? (
                    <>
                      <AlertCircle size={15} className="text-red-500" />
                      最近测试失败
                    </>
                  ) : (
                    '尚未测试'
                  )}
                </div>
                <div className="mt-5 flex flex-wrap gap-2">
                  <AppButton size="sm" variant="outline" disabled={test.isPending} onClick={() => test.mutate(channel.id)}>
                    <Refresh size={14} />
                    测试
                  </AppButton>
                  <AppButton size="sm" variant="outline" onClick={() => openEdit(channel)}>
                    <Settings size={14} />
                    编辑
                  </AppButton>
                  <AppButton
                    size="sm"
                    variant="ghost"
                    disabled={remove.isPending || channel.binding_count > 0}
                    onClick={() => {
                      if (window.confirm(`删除渠道“${channel.name}”？`)) remove.mutate(channel.id)
                    }}
                  >
                    <Trash size={14} />
                    删除
                  </AppButton>
                </div>
                {channel.last_test_error && <p className="mt-3 break-words text-xs text-red-600">{channel.last_test_error}</p>}
              </article>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="实例通知设置" description="每个实例最多绑定一个渠道；所有消息类型默认不推送。" icon={<Settings size={20} />}>
        {integrations.isPending ? (
          <p className="py-10 text-center text-sm text-neutral-500">正在加载接入实例…</p>
        ) : integrations.error ? (
          <ErrorState message={errorMessage(integrations.error)} onRetry={() => void integrations.refetch()} />
        ) : !integrations.data?.length ? (
          <EmptyState title="还没有接入实例" />
        ) : (
          <div className="divide-y divide-neutral-200 dark:divide-neutral-800">
            {integrations.data.map((integration) => (
              <div key={integration.id} className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 className="font-semibold">{integration.name}</h3>
                  <p className="mt-1 text-sm text-neutral-500">
                    {integration.app_code} · {integration.notification_channel_id ? `渠道 #${integration.notification_channel_id}` : '未绑定渠道'}
                  </p>
                </div>
                <AppButton size="sm" variant="outline" onClick={() => setSelectedIntegration(integration)}>
                  配置通知
                </AppButton>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="最近推送记录" description="显示最近 50 条发送任务">
        {deliveries.isPending ? (
          <p className="py-8 text-center text-sm text-neutral-500">正在加载记录…</p>
        ) : deliveries.error ? (
          <ErrorState message={errorMessage(deliveries.error)} onRetry={() => void deliveries.refetch()} />
        ) : !deliveries.data?.length ? (
          <EmptyState title="还没有推送记录" />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] text-left text-sm">
              <thead>
                <tr className="border-b border-neutral-200 text-xs text-neutral-500 dark:border-neutral-800">
                  <th className="px-2 py-3">时间</th>
                  <th className="px-2 py-3">状态</th>
                  <th className="px-2 py-3">实例</th>
                  <th className="px-2 py-3">渠道</th>
                  <th className="px-2 py-3">消息类型</th>
                  <th className="px-2 py-3">尝试</th>
                  <th className="px-2 py-3">操作</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.data.map((item) => (
                  <tr key={item.id} className="border-b border-neutral-100 dark:border-neutral-900">
                    <td className="whitespace-nowrap px-2 py-3 text-xs text-neutral-500">{new Date(item.created_at).toLocaleString()}</td>
                    <td className="px-2 py-3">
                      <Badge variant={item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'error' : 'outline'} size="sm">
                        {item.status}
                      </Badge>
                    </td>
                    <td className="px-2 py-3">{item.integration_name}</td>
                    <td className="px-2 py-3">{item.channel_name}</td>
                    <td className="px-2 py-3 font-mono text-xs">{item.event_type}</td>
                    <td className="px-2 py-3">{item.attempt_count}</td>
                    <td className="px-2 py-3">
                      {item.status === 'failed' && (
                        <AppButton size="sm" variant="outline" disabled={retry.isPending} onClick={() => retry.mutate(item.id)}>
                          重试
                        </AppButton>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Panel>

      {formOpen && <ChannelFormDrawer editing={editing} form={form} setForm={setForm} formError={formError} saving={save.isPending} onClose={closeForm} onSubmit={submit} />}

      {selectedIntegration && (
        <IntegrationNotificationDrawer
          integration={selectedIntegration}
          channels={channels.data ?? []}
          onClose={() => {
            setSelectedIntegration(null)
            setSearchParams({}, { replace: true })
          }}
          onSaved={() => {
            setSelectedIntegration(null)
            setSearchParams({}, { replace: true })
            void client.invalidateQueries({
              queryKey: ['webhooks', 'integrations'],
            })
          }}
        />
      )}
    </div>
  )
}

function ChannelFormDrawer({
  editing,
  form,
  setForm,
  formError,
  saving,
  onClose,
  onSubmit,
}: {
  editing: NotificationChannel | null
  form: ChannelFormState
  setForm: Dispatch<SetStateAction<ChannelFormState>>
  formError: string
  saving: boolean
  onClose: () => void
  onSubmit: (event: FormEvent) => void
}) {
  function update<K extends keyof ChannelFormState>(key: K, value: ChannelFormState[K]) {
    setForm((previous) => ({ ...previous, [key]: value }))
  }

  const keepsExistingCredential = Boolean(editing?.has_credentials)
  const keepsExistingAppriseCredential = Boolean(editing?.has_credentials && (editing.config.mode === 'stateless' ? 'stateless' : 'stateful') === form.appriseMode)
  const wxPusherHasTarget = Boolean(form.wxPusherUids.trim() || form.wxPusherTopicIds.trim())

  const secretLabel = (label: string) => `${label}${keepsExistingCredential ? '（留空保持原值）' : ''}`

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={onClose}>
      <aside className="h-full w-full max-w-xl overflow-y-auto bg-white p-5 shadow-2xl dark:bg-neutral-950 sm:p-6" onClick={(event) => event.stopPropagation()}>
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-emerald-700">Notification Channel</p>
            <h2 className="mt-1 text-xl font-bold">{editing ? '编辑推送渠道' : '新增推送渠道'}</h2>
          </div>
          <AppButton size="sm" variant="outline" onClick={onClose}>
            关闭
          </AppButton>
        </div>

        <form className="mt-6 space-y-5" onSubmit={onSubmit}>
          <label className="block space-y-2 text-sm font-medium">
            <span>渠道名称</span>
            <Input value={form.name} onChange={(event) => update('name', event.target.value)} required maxLength={100} />
          </label>
          <label className="block space-y-2 text-sm font-medium">
            <span>渠道类型</span>
            <select className={inputClass} value={form.type} disabled={Boolean(editing)} onChange={(event) => update('type', event.target.value as NotificationChannelType)}>
              {Object.entries(channelLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>

          {form.type === 'webhook' && (
            <>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('目标 URL')}</span>
                <Input type="url" value={form.url} onChange={(event) => update('url', event.target.value)} required={!keepsExistingCredential} placeholder="https://example.com/notify" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>自定义请求头 JSON（可选）</span>
                <textarea
                  className="min-h-28 w-full rounded-lg border border-neutral-300 bg-white p-3 font-mono text-xs dark:border-neutral-700 dark:bg-neutral-900"
                  value={form.headers}
                  onChange={(event) => update('headers', event.target.value)}
                  placeholder={'{"Authorization":"Bearer ..."}'}
                />
              </label>
            </>
          )}

          {form.type === 'telegram' && (
            <>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('Bot Token')}</span>
                <Input type="password" value={form.botToken} onChange={(event) => update('botToken', event.target.value)} required={!keepsExistingCredential} />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>Chat ID</span>
                <Input value={form.chatId} onChange={(event) => update('chatId', event.target.value)} required />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>Message Thread ID（可选）</span>
                <Input type="number" value={form.threadId} onChange={(event) => update('threadId', event.target.value)} />
              </label>
              <SwitchRow title="静默发送" checked={form.silent} onChange={(checked) => update('silent', checked)} />
            </>
          )}

          {form.type === 'apprise' && (
            <>
              <label className="block space-y-2 text-sm font-medium">
                <span>Apprise Base URL</span>
                <Input type="url" value={form.baseUrl} onChange={(event) => update('baseUrl', event.target.value)} required placeholder="https://apprise.example.com" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>模式</span>
                <select className={inputClass} value={form.appriseMode} onChange={(event) => update('appriseMode', event.target.value as 'stateful' | 'stateless')}>
                  <option value="stateful">Stateful（推荐）</option>
                  <option value="stateless">Stateless</option>
                </select>
              </label>
              {form.appriseMode === 'stateful' ? (
                <label className="block space-y-2 text-sm font-medium">
                  <span>
                    配置 Key
                    {keepsExistingAppriseCredential && '（留空保持原值）'}
                  </span>
                  <Input type="password" value={form.appriseKey} onChange={(event) => update('appriseKey', event.target.value)} required={!keepsExistingAppriseCredential} />
                </label>
              ) : (
                <label className="block space-y-2 text-sm font-medium">
                  <span>
                    Apprise URLs
                    {keepsExistingAppriseCredential && '（留空保持原值）'}
                  </span>
                  <Input type="password" value={form.appriseUrls} onChange={(event) => update('appriseUrls', event.target.value)} required={!keepsExistingAppriseCredential} />
                </label>
              )}
              <label className="block space-y-2 text-sm font-medium">
                <span>Tag（可选）</span>
                <Input value={form.tag} onChange={(event) => update('tag', event.target.value)} />
              </label>
            </>
          )}

          {form.type === 'email' && (
            <>
              <div className="grid gap-4 sm:grid-cols-[1fr_8rem]">
                <label className="block space-y-2 text-sm font-medium">
                  <span>SMTP 主机</span>
                  <Input value={form.smtpHost} onChange={(event) => update('smtpHost', event.target.value)} required placeholder="smtp.example.com" />
                </label>
                <label className="block space-y-2 text-sm font-medium">
                  <span>端口</span>
                  <Input type="number" min="1" max="65535" value={form.smtpPort} onChange={(event) => update('smtpPort', event.target.value)} required />
                </label>
              </div>
              <label className="block space-y-2 text-sm font-medium">
                <span>连接加密</span>
                <select className={inputClass} value={form.smtpEncryption} onChange={(event) => update('smtpEncryption', event.target.value as 'starttls' | 'tls' | 'none')}>
                  <option value="starttls">STARTTLS（常用端口 587）</option>
                  <option value="tls">TLS（常用端口 465）</option>
                  <option value="none">不加密（仅允许私有网络）</option>
                </select>
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>发件人</span>
                <Input type="email" value={form.smtpFrom} onChange={(event) => update('smtpFrom', event.target.value)} required placeholder="notice@example.com" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>收件人</span>
                <Input value={form.smtpTo} onChange={(event) => update('smtpTo', event.target.value)} required placeholder="a@example.com, b@example.com" />
                <span className="block text-xs font-normal text-neutral-500">多个地址使用逗号或分号分隔。</span>
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>SMTP 用户名（可选，与密码同时填写）</span>
                <Input value={form.smtpUsername} onChange={(event) => update('smtpUsername', event.target.value)} autoComplete="off" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('SMTP 密码（可选，与用户名同时填写）')}</span>
                <Input type="password" value={form.smtpPassword} onChange={(event) => update('smtpPassword', event.target.value)} autoComplete="new-password" />
              </label>
            </>
          )}

          {form.type === 'serverchan' && (
            <label className="block space-y-2 text-sm font-medium">
              <span>{secretLabel('SendKey')}</span>
              <Input
                type="password"
                value={form.serverChanSendKey}
                onChange={(event) => update('serverChanSendKey', event.target.value)}
                required={!keepsExistingCredential}
                placeholder="SCT... 或 sctp..."
              />
            </label>
          )}

          {form.type === 'bark' && (
            <>
              <label className="block space-y-2 text-sm font-medium">
                <span>Bark 服务地址</span>
                <Input type="url" value={form.barkBaseUrl} onChange={(event) => update('barkBaseUrl', event.target.value)} required placeholder="https://api.day.app" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('Device Key')}</span>
                <Input type="password" value={form.barkDeviceKey} onChange={(event) => update('barkDeviceKey', event.target.value)} required={!keepsExistingCredential} />
              </label>
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block space-y-2 text-sm font-medium">
                  <span>分组（可选）</span>
                  <Input value={form.barkGroup} onChange={(event) => update('barkGroup', event.target.value)} />
                </label>
                <label className="block space-y-2 text-sm font-medium">
                  <span>声音（可选）</span>
                  <Input value={form.barkSound} onChange={(event) => update('barkSound', event.target.value)} />
                </label>
              </div>
            </>
          )}

          {(form.type === 'dingtalk' || form.type === 'feishu') && (
            <>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel(`${form.type === 'dingtalk' ? '钉钉' : '飞书'}机器人 Webhook URL`)}</span>
                <Input type="password" value={form.robotWebhookUrl} onChange={(event) => update('robotWebhookUrl', event.target.value)} required={!keepsExistingCredential} autoComplete="off" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('签名密钥（可选）')}</span>
                <Input type="password" value={form.robotSigningSecret} onChange={(event) => update('robotSigningSecret', event.target.value)} autoComplete="new-password" />
              </label>
            </>
          )}

          {form.type === 'whatsapp' && (
            <>
              <Message variant="warning" title="当前发送普通文本消息" description="需要满足 WhatsApp Cloud API 的会话窗口要求；超出窗口时应使用已审核的消息模板。" />
              <label className="block space-y-2 text-sm font-medium">
                <span>Graph API 版本</span>
                <Input value={form.whatsappVersion} onChange={(event) => update('whatsappVersion', event.target.value)} required pattern="v[0-9]+\\.[0-9]+" placeholder="v25.0" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('Access Token')}</span>
                <Input type="password" value={form.whatsappToken} onChange={(event) => update('whatsappToken', event.target.value)} required={!keepsExistingCredential} />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('Phone Number ID')}</span>
                <Input
                  type="password"
                  inputMode="numeric"
                  value={form.whatsappPhoneNumberId}
                  onChange={(event) => update('whatsappPhoneNumberId', event.target.value)}
                  required={!keepsExistingCredential}
                />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('收件号码')}</span>
                <Input
                  type="password"
                  inputMode="tel"
                  value={form.whatsappRecipient}
                  onChange={(event) => update('whatsappRecipient', event.target.value)}
                  required={!keepsExistingCredential}
                  placeholder="国家码 + 手机号"
                />
              </label>
            </>
          )}

          {form.type === 'wxpusher' && (
            <>
              <label className="block space-y-2 text-sm font-medium">
                <span>{secretLabel('AppToken')}</span>
                <Input type="password" value={form.wxPusherAppToken} onChange={(event) => update('wxPusherAppToken', event.target.value)} required={!keepsExistingCredential} />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>UIDs（可选）</span>
                <Input value={form.wxPusherUids} onChange={(event) => update('wxPusherUids', event.target.value)} placeholder="UID_xxx, UID_yyy" />
              </label>
              <label className="block space-y-2 text-sm font-medium">
                <span>Topic IDs（可选）</span>
                <Input value={form.wxPusherTopicIds} onChange={(event) => update('wxPusherTopicIds', event.target.value)} placeholder="123, 456" />
                <span className="block text-xs font-normal text-neutral-500">UIDs 和 Topic IDs 至少填写一项。</span>
              </label>
            </>
          )}

          <SwitchRow title="使用系统 HTTP 代理" description="仅此渠道的测试和消息推送经过系统设置中的代理。" checked={form.useProxy} onChange={(checked) => update('useProxy', checked)} />
          <SwitchRow title="启用渠道" description="建议先保存并测试成功后再启用。" checked={form.enabled} onChange={(checked) => update('enabled', checked)} />

          {form.type === 'wxpusher' && !wxPusherHasTarget && <Message variant="error" title="UIDs 和 Topic IDs 至少填写一项" />}
          {formError && <Message variant="error" title={formError} />}
          <AppButton type="submit" disabled={saving || !form.name.trim() || (form.type === 'wxpusher' && !wxPusherHasTarget)}>
            {saving ? '正在保存…' : '保存渠道'}
          </AppButton>
        </form>
      </aside>
    </div>
  )
}

function SwitchRow({ title, description, checked, onChange }: { title: string; description?: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
      <div>
        <p className="text-sm font-medium">{title}</p>
        {description && <p className="mt-1 text-xs text-neutral-500">{description}</p>}
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

function IntegrationNotificationDrawer({ integration, channels, onClose, onSaved }: { integration: Integration; channels: NotificationChannel[]; onClose: () => void; onSaved: () => void }) {
  const settings = useQuery({
    queryKey: ['notifications', 'integration', integration.id],
    queryFn: () => api.integrationNotificationSettings(integration.id),
  })
  const [channelId, setChannelId] = useState('')
  const [enabledTypes, setEnabledTypes] = useState<string[]>([])
  useEffect(() => {
    if (settings.data) {
      setChannelId(settings.data.channel_id ? String(settings.data.channel_id) : '')
      setEnabledTypes(settings.data.event_types.filter((item) => item.notify).map((item) => item.code))
    }
  }, [settings.data])
  const save = useMutation({
    mutationFn: () => api.updateIntegrationNotificationSettings(integration.id, channelId ? Number(channelId) : null, enabledTypes),
    onSuccess: onSaved,
  })
  function toggle(code: string, checked: boolean) {
    setEnabledTypes((current) => (checked ? [...new Set([...current, code])] : current.filter((item) => item !== code)))
  }
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={onClose}>
      <aside className="h-full w-full max-w-xl overflow-y-auto bg-white p-6 shadow-2xl dark:bg-neutral-950" onClick={(event) => event.stopPropagation()}>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-emerald-700">{integration.app_code}</p>
            <h2 className="mt-1 text-xl font-bold">{integration.name} · 通知设置</h2>
          </div>
          <AppButton size="sm" variant="outline" onClick={onClose}>
            关闭
          </AppButton>
        </div>
        {settings.isPending ? (
          <p className="mt-8 text-sm text-neutral-500">正在加载设置…</p>
        ) : settings.error ? (
          <div className="mt-8">
            <ErrorState message={errorMessage(settings.error)} onRetry={() => void settings.refetch()} />
          </div>
        ) : (
          <div className="mt-7 space-y-6">
            <label className="block space-y-2 text-sm font-medium">
              <span>推送渠道</span>
              <select className={inputClass} value={channelId} onChange={(event) => setChannelId(event.target.value)}>
                <option value="">不绑定渠道</option>
                {channels.map((channel) => (
                  <option key={channel.id} value={channel.id}>
                    {channel.name} · {channelLabels[channel.type]}
                    {channel.enabled ? '' : '（已停用）'}
                  </option>
                ))}
              </select>
            </label>
            {channelId && !channels.find((item) => item.id === Number(channelId))?.enabled && (
              <Message variant="warning" title="所选渠道当前已停用" description="消息类型设置会保留，但在渠道启用前不会发送。" />
            )}
            <section>
              <h3 className="font-semibold">推送消息类型</h3>
              <p className="mt-1 text-sm text-neutral-500">未开启的类型只保存在消息流中，不发送外部通知。</p>
              <div className="mt-4 space-y-3">
                {settings.data?.event_types.map((eventType) => (
                  <div key={eventType.code} className="flex items-center justify-between gap-4 rounded-xl border border-neutral-200 p-4 dark:border-neutral-800">
                    <div>
                      <p className="font-medium">
                        {eventType.name}
                        {eventType.is_default && (
                          <Badge className="ml-2" variant="outline" size="sm">
                            默认类型
                          </Badge>
                        )}
                      </p>
                      <p className="mt-1 font-mono text-xs text-neutral-500">
                        {eventType.code}
                        {eventType.is_default ? ' · 未知消息类型将匹配此规则' : ''}
                      </p>
                    </div>
                    <Switch checked={enabledTypes.includes(eventType.code)} onCheckedChange={(checked) => toggle(eventType.code, checked)} />
                  </div>
                ))}
              </div>
            </section>
            {save.error && <Message variant="error" title={errorMessage(save.error, '保存通知设置失败')} />}
            <AppButton disabled={save.isPending} onClick={() => save.mutate()}>
              {save.isPending ? '正在保存…' : '保存通知设置'}
            </AppButton>
          </div>
        )}
      </aside>
    </div>
  )
}
