import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { AlertCircle, Bell, Check, Plus, Refresh, Settings, Trash } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState, type Dispatch, type FormEvent, type SetStateAction } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import type { Integration, NotificationChannel, NotificationChannelInput, NotificationChannelType, NotificationEventTypeSetting } from '@/api/types'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState, Message } from '@/components/common/feedback'
import { notificationChannelAdapter, notificationChannelAdapters, notificationChannelLabel } from '@/components/notification-channels/registry'
import { channelInputClass, notificationChannelBindingName, type ChannelCommonValues, type ChannelFields, type ChannelFieldValue } from '@/components/notification-channels/types'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'
import { formatDateTime } from '@/i18n/format'

type ChannelFormState = ChannelCommonValues & { fields: ChannelFields }

function emptyChannelForm(type: NotificationChannelType = 'webhook'): ChannelFormState {
  return {
    name: '',
    type,
    enabled: false,
    useProxy: false,
    fields: notificationChannelAdapter(type).defaultFields(),
  }
}

function formFromChannel(channel: NotificationChannel): ChannelFormState {
  return {
    name: channel.name,
    type: channel.type,
    enabled: channel.enabled,
    useProxy: Boolean(channel.config.use_proxy),
    fields: notificationChannelAdapter(channel.type).fieldsFromChannel(channel),
  }
}

function channelPayload(form: ChannelFormState): NotificationChannelInput {
  return notificationChannelAdapter(form.type).toPayload(form, form.fields)
}

export function NotificationChannelsPage() {
  const { t } = useTranslation()
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
    const validationError = notificationChannelAdapter(form.type).validate?.(form.fields)
    if (validationError) {
      setFormError(validationError)
      return
    }
    try {
      save.mutate(channelPayload(form))
    } catch {
      setFormError(t('notifications.validationFailed'))
    }
  }

  return (
    <div className="mx-auto max-w-7xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader
        eyebrow={t('notifications.eyebrow')}
        title={t('notifications.title')}
        description={t('notifications.description')}
        action={
          <AppButton size="sm" onClick={openCreate}>
            <Plus size={16} />
            {t('notifications.create')}
          </AppButton>
        }
      />
      {(save.error || test.error || remove.error || retry.error) && <Message variant="error" title={errorMessage(save.error || test.error || remove.error || retry.error, t('notifications.operationFailed'))} />}

      <Panel title={t('notifications.channels')} description={t('notifications.channelsCount', { count: channels.data?.length ?? 0 })} icon={<Bell size={20} />}>
        {channels.isPending ? (
          <p className="py-10 text-center text-sm text-neutral-500">{t('notifications.loadingChannels')}</p>
        ) : channels.error ? (
          <ErrorState message={errorMessage(channels.error)} onRetry={() => void channels.refetch()} />
        ) : !channels.data?.length ? (
          <EmptyState title={t('notifications.noChannels')} description={t('notifications.noChannelsDescription')} />
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {channels.data.map((channel) => (
              <article key={channel.id} className="rounded-xl border border-neutral-200 p-5 dark:border-neutral-800">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h3 className="font-semibold">{channel.name}</h3>
                    <p className="mt-1 text-xs text-neutral-500">
                      {notificationChannelLabel(channel.type)} · {t('notifications.bindingsCount', { count: channel.binding_count })}{channel.config.use_proxy ? ` · ${t('notifications.usesProxy')}` : ''}
                    </p>
                  </div>
                  <Badge variant={channel.enabled ? 'success' : 'outline'}>{t(channel.enabled ? 'common.states.enabled' : 'common.states.disabled')}</Badge>
                </div>
                <div className="mt-4 flex items-center gap-2 text-xs text-neutral-500">
                  {channel.last_test_status === 'succeeded' ? (
                    <>
                      <Check size={15} className="text-emerald-600" />
                      {t('notifications.testSucceeded')}
                    </>
                  ) : channel.last_test_status === 'failed' ? (
                    <>
                      <AlertCircle size={15} className="text-red-500" />
                      {t('notifications.testFailedStatus')}
                    </>
                  ) : (
                    t('notifications.neverTested')
                  )}
                </div>
                <div className="mt-5 flex flex-wrap gap-2">
                  <AppButton size="sm" variant="outline" disabled={test.isPending} onClick={() => test.mutate(channel.id)}>
                    <Refresh size={14} />
                    {t('common.actions.test')}
                  </AppButton>
                  <AppButton size="sm" variant="outline" onClick={() => openEdit(channel)}>
                    <Settings size={14} />
                    {t('common.actions.edit')}
                  </AppButton>
                  <AppButton
                    size="sm"
                    variant="ghost"
                    disabled={remove.isPending || channel.binding_count > 0}
                    onClick={() => {
                      if (window.confirm(t('notifications.deletePrompt', { name: channel.name }))) remove.mutate(channel.id)
                    }}
                  >
                    <Trash size={14} />
                    {t('common.actions.delete')}
                  </AppButton>
                </div>
                {channel.last_test_error && <p className="mt-3 break-words text-xs text-red-600">{channel.last_test_error}</p>}
              </article>
            ))}
          </div>
        )}
      </Panel>

      <Panel title={t('notifications.appBindings')} description={t('notifications.appBindingsDescription')} icon={<Settings size={20} />}>
        {integrations.isPending ? (
          <p className="py-10 text-center text-sm text-neutral-500">{t('integrations.loading')}</p>
        ) : integrations.error ? (
          <ErrorState message={errorMessage(integrations.error)} onRetry={() => void integrations.refetch()} />
        ) : !integrations.data?.length ? (
          <EmptyState title={t('integrations.empty')} />
        ) : (
          <div className="divide-y divide-neutral-200 dark:divide-neutral-800">
            {integrations.data.map((integration) => {
              return <div key={integration.id} className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 className="font-semibold">{integration.name}</h3>
                  <p className="mt-1 text-sm text-neutral-500">
                    {integration.app_code} · {notificationChannelBindingName(integration.notification_channel_id, channels.data ?? [], channels.isPending)}
                  </p>
                </div>
                <AppButton size="sm" variant="outline" onClick={() => setSelectedIntegration(integration)}>
                  {t('notifications.configure')}
                </AppButton>
              </div>
            })}
          </div>
        )}
      </Panel>

      <Panel title={t('notifications.deliveryRecords')} description={t('notifications.deliveryDescription')}>
        {deliveries.isPending ? (
          <p className="py-8 text-center text-sm text-neutral-500">{t('notifications.loadingDeliveries')}</p>
        ) : deliveries.error ? (
          <ErrorState message={errorMessage(deliveries.error)} onRetry={() => void deliveries.refetch()} />
        ) : !deliveries.data?.length ? (
          <EmptyState title={t('notifications.noDeliveries')} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] text-left text-sm">
              <thead>
                <tr className="border-b border-neutral-200 text-xs text-neutral-500 dark:border-neutral-800">
                  <th className="px-2 py-3">{t('notifications.time')}</th>
                  <th className="px-2 py-3">{t('common.fields.status')}</th>
                  <th className="px-2 py-3">{t('notifications.instance')}</th>
                  <th className="px-2 py-3">{t('notifications.channel')}</th>
                  <th className="px-2 py-3">{t('events.messageType')}</th>
                  <th className="px-2 py-3">{t('notifications.attempts')}</th>
                  <th className="px-2 py-3">{t('notifications.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.data.map((item) => (
                  <tr key={item.id} className="border-b border-neutral-100 dark:border-neutral-900">
                    <td className="whitespace-nowrap px-2 py-3 text-xs text-neutral-500">{formatDateTime(item.created_at)}</td>
                    <td className="px-2 py-3">
                      <Badge variant={item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'error' : 'outline'} size="sm">
                        {t(`events.notificationStates.${item.status}`, { defaultValue: item.status })}
                      </Badge>
                      {item.next_attempt_at && item.status !== 'succeeded' && <p className="mt-1 whitespace-nowrap text-xs text-neutral-500">{t('notifications.deferredUntil', { time: formatDateTime(item.next_attempt_at) })}</p>}
                      {item.provider_error_code && <p className="mt-1 whitespace-nowrap text-xs text-neutral-500">{t('notifications.providerCode', { code: item.provider_error_code })}</p>}
                    </td>
                    <td className="px-2 py-3">{item.integration_name}</td>
                    <td className="px-2 py-3">{item.channel_name}</td>
                    <td className="px-2 py-3 font-mono text-xs">{item.event_type}</td>
                    <td className="px-2 py-3">{item.attempt_count}</td>
                    <td className="px-2 py-3">
                      {item.status === 'failed' && (
                        <AppButton size="sm" variant="outline" disabled={retry.isPending} onClick={() => retry.mutate(item.id)}>
                          {t('common.actions.retry')}
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
  const { t } = useTranslation()
  function update<K extends keyof ChannelCommonValues>(key: K, value: ChannelCommonValues[K]) {
    setForm((previous) => ({ ...previous, [key]: value }))
  }

  function updateField(key: string, value: ChannelFieldValue) {
    setForm((previous) => ({ ...previous, fields: { ...previous.fields, [key]: value } }))
  }

  const adapter = notificationChannelAdapter(form.type)
  const AdapterForm = adapter.Form
  const validationError = adapter.validate?.(form.fields) ?? ''

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={onClose}>
      <aside className="h-full w-full max-w-xl overflow-y-auto bg-white p-5 shadow-2xl dark:bg-neutral-950 sm:p-6" onClick={(event) => event.stopPropagation()}>
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-emerald-700">Notification Channel</p>
            <h2 className="mt-1 text-xl font-bold">{t(editing ? 'notifications.edit' : 'notifications.createTitle')}</h2>
          </div>
          <AppButton size="sm" variant="outline" onClick={onClose}>
            {t('common.actions.close')}
          </AppButton>
        </div>

        <form className="mt-6 space-y-5" onSubmit={onSubmit}>
          <label className="block space-y-2 text-sm font-medium">
            <span>{t('notifications.channelName')}</span>
            <Input value={form.name} onChange={(event) => update('name', event.target.value)} required maxLength={100} />
          </label>
          <label className="block space-y-2 text-sm font-medium">
            <span>{t('notifications.channelType')}</span>
            <select
              className={channelInputClass}
              value={form.type}
              disabled={Boolean(editing)}
              onChange={(event) => {
                const type = event.target.value as NotificationChannelType
                setForm((previous) => ({ ...previous, type, fields: notificationChannelAdapter(type).defaultFields() }))
              }}
            >
              {notificationChannelAdapters().map((item) => (
                <option key={item.type} value={item.type}>
                  {notificationChannelLabel(item.type)}
                </option>
              ))}
            </select>
          </label>

          <AdapterForm fields={form.fields} update={updateField} />

          <SwitchRow title={t('notifications.proxy')} description={t('notifications.proxyDescription')} checked={form.useProxy} onChange={(checked) => update('useProxy', checked)} />
          <SwitchRow title={t('notifications.enabled')} description={t('notifications.enabledDescription')} checked={form.enabled} onChange={(checked) => update('enabled', checked)} />

          {validationError && <Message variant="error" title={validationError} />}
          {formError && <Message variant="error" title={formError} />}
          <AppButton type="submit" disabled={saving || !form.name.trim() || Boolean(validationError)}>
            {t(saving ? 'common.states.saving' : 'common.actions.save')}
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
  const { t } = useTranslation()
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
  const eventTypeGroups = groupNotificationEventTypes(settings.data?.event_types ?? [])
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={onClose}>
      <aside className="h-full w-full max-w-xl overflow-y-auto bg-white p-6 shadow-2xl dark:bg-neutral-950" onClick={(event) => event.stopPropagation()}>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-emerald-700">{integration.app_code}</p>
            <h2 className="mt-1 text-xl font-bold">{integration.name} · {t('integrations.notificationSettings')}</h2>
          </div>
          <AppButton size="sm" variant="outline" onClick={onClose}>
            {t('common.actions.close')}
          </AppButton>
        </div>
        {settings.isPending ? (
          <p className="mt-8 text-sm text-neutral-500">{t('notifications.loadingSettings')}</p>
        ) : settings.error ? (
          <div className="mt-8">
            <ErrorState message={errorMessage(settings.error)} onRetry={() => void settings.refetch()} />
          </div>
        ) : (
          <div className="mt-7 space-y-6">
            <label className="block space-y-2 text-sm font-medium">
              <span>{t('notifications.boundChannel')}</span>
              <select className={channelInputClass} value={channelId} onChange={(event) => setChannelId(event.target.value)}>
                <option value="">{t('notifications.noBinding')}</option>
                {channels.map((channel) => (
                  <option key={channel.id} value={channel.id}>
                    {channel.name} · {notificationChannelLabel(channel.type)}
                    {channel.enabled ? '' : ` (${t('common.states.disabled')})`}
                  </option>
                ))}
              </select>
            </label>
            {channelId && !channels.find((item) => item.id === Number(channelId))?.enabled && (
              <Message variant="warning" title={t('notifications.selectedDisabled')} description={t('notifications.selectedDisabledDescription')} />
            )}
            <section>
              <h3 className="font-semibold">{t('notifications.enabledTypes')}</h3>
              <p className="mt-1 text-sm text-neutral-500">{t('notifications.enabledTypesDescription')}</p>
              <div className="mt-4 space-y-5">
                {eventTypeGroups.map((group) => (
                  <div key={group.name}>
                    <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-neutral-500">{t(group.name)}</h4>
                    <div className="space-y-2">
                      {group.items.map((eventType) => (
                        <div key={eventType.code} className="flex items-center justify-between gap-4 rounded-xl border border-neutral-200 p-4 dark:border-neutral-800">
                          <div>
                            <p className="font-medium">
                              {t(`events.types.${eventType.code}`, { defaultValue: eventType.name })}
                              {eventType.is_default && <Badge className="ml-2" variant="outline" size="sm">{t('notifications.defaultType')}</Badge>}
                            </p>
                            <p className="mt-1 font-mono text-xs text-neutral-500">
                              {eventType.code}
                              {eventType.is_default ? ` · ${t('notifications.defaultTypeDescription')}` : ''}
                            </p>
                          </div>
                          <Switch checked={enabledTypes.includes(eventType.code)} onCheckedChange={(checked) => toggle(eventType.code, checked)} />
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </section>
            {save.error && <Message variant="error" title={errorMessage(save.error, t('notifications.saveFailed'))} />}
            <AppButton disabled={save.isPending} onClick={() => save.mutate()}>
              {t(save.isPending ? 'common.states.saving' : 'notifications.saveBinding')}
            </AppButton>
          </div>
        )}
      </aside>
    </div>
  )
}

function groupNotificationEventTypes(items: NotificationEventTypeSetting[]) {
  const groups = [
    { name: 'notifications.groups.media', matches: (code: string) => code.startsWith('media_'), items: [] as NotificationEventTypeSetting[] },
    { name: 'notifications.groups.playback', matches: (code: string) => code.startsWith('playback_'), items: [] as NotificationEventTypeSetting[] },
    { name: 'notifications.groups.auth', matches: (code: string) => code.startsWith('authentication_') || code.startsWith('user_') || code === 'session_started', items: [] as NotificationEventTypeSetting[] },
    { name: 'notifications.groups.system', matches: (code: string) => code.startsWith('plugin_') || ['server_restart_required', 'task_completed', 'subtitle_download_failed'].includes(code), items: [] as NotificationEventTypeSetting[] },
    { name: 'notifications.groups.other', matches: () => true, items: [] as NotificationEventTypeSetting[] },
  ]
  for (const item of items) (groups.find((group) => group.matches(item.code)) ?? groups.at(-1))?.items.push(item)
  return groups.filter((group) => group.items.length > 0)
}
