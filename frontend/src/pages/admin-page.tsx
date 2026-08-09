import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { Spinner } from '@appica/ui-react/spinner'
import { Eye, EyeOff, Refresh, Settings, ShieldCheck, User, Users } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import type { User as UserModel } from '@/api/types'
import { AppButton } from '@/components/common/app-button'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { ErrorState, Message } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { DataTable } from '@/components/data-table/data-table'
import { errorMessage } from '@/lib/error-message'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/i18n/format'

export function AdminPage() {
  const { t } = useTranslation()
  const currentUser = useAuthStore((state) => state.user)
  const queryClient = useQueryClient()
  const [retentionDays, setRetentionDays] = useState('7')
  const [httpProxyURL, setHTTPProxyURL] = useState('')
  const [tmdbAPIKey, setTMDBAPIKey] = useState('')
  const [tmdbAPIKeyVisible, setTMDBAPIKeyVisible] = useState(false)
  const [connectionTestMessage, setConnectionTestMessage] = useState('')
  const [publicBaseURL, setPublicBaseURL] = useState('')
  const users = useQuery({ queryKey: ['admin', 'users'], queryFn: api.users })
  const settings = useQuery({ queryKey: ['admin', 'settings'], queryFn: api.systemSettings })
  const updateSettings = useMutation({
    mutationFn: api.updateSystemSettings,
    onSuccess: (_, variables) => {
      if (variables.clear_tmdb_api_key) setTMDBAPIKeyVisible(false)
      void queryClient.invalidateQueries({ queryKey: ['admin', 'settings'] })
    },
  })
  const updateRole = useMutation({
    mutationFn: ({ id, isAdmin }: { id: number; isAdmin: boolean }) => api.setUserRole(id, isAdmin),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
  const testTMDBConnection = useMutation({
    mutationFn: api.testTMDBConnection,
    onMutate: () => setConnectionTestMessage(''),
    onSuccess: () => setConnectionTestMessage(t('common.feedback.connectionSuccess')),
  })
  const testHTTPProxy = useMutation({
    mutationFn: api.testHTTPProxy,
    onMutate: () => setConnectionTestMessage(''),
    onSuccess: () => setConnectionTestMessage(t('common.feedback.connectionSuccess')),
  })

  useEffect(() => {
    if (settings.data) {
      setRetentionDays(String(settings.data.api_log_retention_days))
      setHTTPProxyURL(settings.data.http_proxy_url || '')
      setTMDBAPIKey(settings.data.tmdb_api_key || '')
      setPublicBaseURL(settings.data.public_base_url || window.location.origin)
    }
  }, [settings.data])

  const parsedRetentionDays = Number(retentionDays)
  const retentionDaysValid = Number.isInteger(parsedRetentionDays) && parsedRetentionDays >= 1 && parsedRetentionDays <= 30
  const publicBaseURLValid = validPublicBaseURL(publicBaseURL)
  const httpProxyURLValid = validHTTPProxyURL(httpProxyURL)

  const columns = useMemo<ColumnDef<UserModel, unknown>[]>(() => [
    {
      accessorKey: 'username',
      header: t('admin.user'),
      cell: ({ row }) => <div className="flex items-center gap-3"><span className="grid size-9 place-items-center rounded-full bg-emerald-50 text-emerald-700 dark:bg-emerald-950"><User size={17} /></span><div><p className="font-semibold">{row.original.username}</p><p className="text-xs text-neutral-500">ID: {row.original.id}</p></div></div>,
    },
    {
      accessorKey: 'is_admin',
      header: t('admin.role'),
      cell: ({ row }) => <div className="flex flex-wrap gap-2"><Badge variant={row.original.is_admin ? 'primary' : 'soft'} size="sm">{t(row.original.is_admin ? 'admin.administrator' : 'admin.regularUser')}</Badge>{row.original.id === currentUser?.id && <Badge variant="outline" size="sm">{t('admin.currentUser')}</Badge>}</div>,
    },
    { accessorKey: 'created_at', header: t('profile.registeredAt'), cell: ({ getValue }) => <span className="whitespace-nowrap text-sm text-neutral-500">{getValue() ? formatDateTime(String(getValue())) : '-'}</span> },
    {
      id: 'actions', header: t('admin.actions'),
      cell: ({ row }) => row.original.id === currentUser?.id ? <span className="text-xs text-neutral-400">{t('admin.cannotModifySelf')}</span> : (
        <ConfirmDialog
          title={t(row.original.is_admin ? 'admin.revokeTitle' : 'admin.grantTitle')}
          description={t(row.original.is_admin ? 'admin.revokeDescription' : 'admin.grantDescription', { username: row.original.username })}
          confirmLabel={t(row.original.is_admin ? 'admin.revoke' : 'admin.grant')}
          destructive={row.original.is_admin}
          busy={updateRole.isPending}
          onConfirm={() => updateRole.mutate({ id: row.original.id, isAdmin: !row.original.is_admin })}
          trigger={<AppButton size="sm" variant={row.original.is_admin ? 'destructive' : 'outline'}>{t(row.original.is_admin ? 'admin.revokeAdmin' : 'admin.makeAdmin')}</AppButton>}
        />
      ),
    },
  ], [currentUser?.id, updateRole.isPending, t])

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow={t('admin.eyebrow')} title={t('admin.title')} description={t('admin.description')} />
      <Panel title={t('admin.settings')} description={t('admin.settingsImmediate')} icon={<Settings size={20} />}>
        {settings.isPending ? <div className="grid min-h-28 place-items-center"><Spinner className="size-7" /></div> : settings.error ? <ErrorState message={errorMessage(settings.error)} onRetry={() => void settings.refetch()} /> : (
          <div className="space-y-3">
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="min-w-0 flex-1 space-y-2 text-sm font-medium">
                <span>{t('admin.publicBaseUrl')}</span>
                <Input type="url" value={publicBaseURL} disabled={updateSettings.isPending} onChange={(event) => setPublicBaseURL(event.target.value)} placeholder="https://seshat.example.com" />
                <span className={`block text-xs ${publicBaseURL && !publicBaseURLValid ? 'text-red-600' : 'text-neutral-500'}`}>{t('admin.publicBaseUrlHelp')}</span>
              </label>
              <AppButton disabled={updateSettings.isPending || !publicBaseURLValid || publicBaseURL.trim().replace(/\/$/, '') === settings.data?.public_base_url} onClick={() => updateSettings.mutate({ public_base_url: publicBaseURL.trim().replace(/\/$/, '') })}>{t(updateSettings.isPending ? 'common.states.saving' : 'admin.savePublicUrl')}</AppButton>
            </div>
            <div className="flex items-center justify-between gap-5 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
              <div><h3 className="font-semibold">{t('admin.allowRegistration')}</h3><p className="mt-1 text-sm text-neutral-500">{t('admin.allowRegistrationDescription')}</p></div>
              <Switch size="lg" aria-label={t('admin.allowRegistration')} checked={settings.data?.allow_register ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ allow_register: checked })} />
            </div>
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="min-w-0 flex-1 space-y-2 text-sm font-medium">
                <span>{t('admin.httpProxy')}</span>
                <Input type="url" autoComplete="off" value={httpProxyURL} disabled={updateSettings.isPending || testHTTPProxy.isPending || testTMDBConnection.isPending} onChange={(event) => setHTTPProxyURL(event.target.value)} placeholder="http://proxy.example.com:7890" />
                <span className={`block text-xs ${httpProxyURL && !httpProxyURLValid ? 'text-red-600' : 'text-neutral-500'}`}>{t(httpProxyURL && !httpProxyURLValid ? 'admin.httpProxyInvalid' : 'admin.httpProxyHelp')}</span>
              </label>
              <div className="flex flex-wrap gap-2">
                {settings.data?.http_proxy_configured && <ConfirmDialog title={t('admin.clearProxyTitle')} description={t('admin.clearProxyDescription')} confirmLabel={t('admin.clearProxy')} destructive busy={updateSettings.isPending} onConfirm={() => updateSettings.mutate({ clear_http_proxy: true })} trigger={<AppButton variant="outline" disabled={updateSettings.isPending}>{t('admin.clearProxy')}</AppButton>} />}
                <AppButton variant="outline" disabled={testHTTPProxy.isPending || !httpProxyURLValid} onClick={() => testHTTPProxy.mutate({ http_proxy_url: httpProxyURL.trim() })}>{t(testHTTPProxy.isPending ? 'common.states.testing' : 'admin.testProxy')}</AppButton>
                <AppButton disabled={updateSettings.isPending || !httpProxyURLValid} onClick={() => updateSettings.mutate({ http_proxy_url: httpProxyURL.trim() })}>{t(updateSettings.isPending ? 'common.states.saving' : 'admin.saveProxy')}</AppButton>
              </div>
            </div>
            <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                <div className="min-w-0 flex-1 space-y-4">
                <label className="block space-y-2 text-sm font-medium">
                  <span>{t('admin.tmdbApiKey')}</span>
                  <Input
                    type={tmdbAPIKeyVisible ? 'text' : 'password'}
                    autoComplete="off"
                    value={tmdbAPIKey}
                    disabled={updateSettings.isPending || testTMDBConnection.isPending}
                    onChange={(event) => setTMDBAPIKey(event.target.value)}
                    placeholder={t('admin.tmdbPlaceholder')}
                    endSlot={tmdbAPIKey && <button type="button" disabled={updateSettings.isPending} className="rounded p-1 text-neutral-500 hover:text-neutral-900 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:text-white" aria-label={t(tmdbAPIKeyVisible ? 'admin.hideTmdbKey' : 'admin.showTmdbKey')} title={t(tmdbAPIKeyVisible ? 'admin.hideTmdbKey' : 'admin.showTmdbKey')} onClick={() => setTMDBAPIKeyVisible((visible) => !visible)}>{tmdbAPIKeyVisible ? <EyeOff size={17} /> : <Eye size={17} />}</button>}
                  />
                  <span className="block text-xs text-neutral-500">{t('admin.tmdbHelp')}</span>
                </label>
                <div className="flex items-center justify-between gap-4 rounded-lg border border-neutral-200 p-3 dark:border-neutral-800">
                  <div><p className="text-sm font-medium">{t('admin.useProxy')}</p><p className="mt-1 text-xs text-neutral-500">{t(settings.data?.http_proxy_configured ? 'admin.proxyConfiguredHelp' : 'admin.proxyMissingHelp')}</p></div>
                  <Switch aria-label={t('admin.useProxy')} checked={settings.data?.tmdb_use_proxy ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ tmdb_use_proxy: checked })} />
                </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  {settings.data?.tmdb_configured && <ConfirmDialog title={t('admin.clearKeyTitle')} description={t('admin.clearKeyDescription')} confirmLabel={t('admin.clearKey')} destructive busy={updateSettings.isPending} onConfirm={() => updateSettings.mutate({ clear_tmdb_api_key: true })} trigger={<AppButton variant="outline" disabled={updateSettings.isPending}>{t('admin.clearKey')}</AppButton>} />}
                  <AppButton variant="outline" disabled={testTMDBConnection.isPending || !tmdbAPIKey.trim() || Boolean(settings.data?.tmdb_use_proxy && !httpProxyURLValid)} onClick={() => testTMDBConnection.mutate({ tmdb_api_key: tmdbAPIKey.trim(), http_proxy_url: httpProxyURL.trim(), use_proxy: settings.data?.tmdb_use_proxy ?? false })}>{t(testTMDBConnection.isPending ? 'common.states.testing' : 'admin.testTmdb')}</AppButton>
                  <AppButton disabled={updateSettings.isPending || !tmdbAPIKey.trim()} onClick={() => updateSettings.mutate({ tmdb_api_key: tmdbAPIKey.trim() })}>{t(updateSettings.isPending ? 'common.states.saving' : 'admin.saveKey')}</AppButton>
                </div>
              </div>
              <p className="mt-3 border-t border-neutral-200 pt-3 text-xs text-neutral-400 dark:border-neutral-800">{t('admin.tmdbNoticeLabel')}: {t('admin.tmdbNotice')}</p>
            </div>
            {connectionTestMessage && <Message variant="success" title={connectionTestMessage} />}
            {testHTTPProxy.error && <Message variant="error" title={errorMessage(testHTTPProxy.error, t('admin.proxyTestFailed'))} />}
            {testTMDBConnection.error && <Message variant="error" title={errorMessage(testTMDBConnection.error, t('integrations.mediaSettings.testFailed'))} />}
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="max-w-xs flex-1 space-y-2 text-sm font-medium">
                <span>{t('admin.retention')}</span>
                <Input type="number" min={1} max={30} step={1} value={retentionDays} disabled={updateSettings.isPending} onChange={(event) => setRetentionDays(event.target.value)} />
                <span className={`block text-xs ${retentionDaysValid ? 'text-neutral-500' : 'text-red-600'}`}>{t(retentionDaysValid ? 'admin.retentionHelp' : 'admin.retentionInvalid')}</span>
              </label>
              <AppButton disabled={updateSettings.isPending || !retentionDaysValid || parsedRetentionDays === settings.data?.api_log_retention_days} onClick={() => updateSettings.mutate({ api_log_retention_days: parsedRetentionDays })}>{t(updateSettings.isPending ? 'common.states.saving' : 'admin.saveRetention')}</AppButton>
            </div>
          </div>
        )}
        {updateSettings.error && <div className="mt-4"><Message variant="error" title={errorMessage(updateSettings.error, t('admin.settingsFailed'))} /></div>}
      </Panel>
      <Panel title={t('admin.users')} description={t('admin.usersCount', { count: users.data?.length ?? 0 })} icon={<Users size={20} />} action={<AppButton size="sm" variant="ghost" disabled={users.isFetching} onClick={() => void users.refetch()}><Refresh size={16} />{t('common.actions.refresh')}</AppButton>}>
        {updateRole.error && <div className="mb-4"><Message variant="error" title={errorMessage(updateRole.error, t('admin.roleFailed'))} /></div>}
        {users.isPending ? <div className="grid min-h-56 place-items-center"><Spinner className="size-8" /></div> : users.error ? <ErrorState message={errorMessage(users.error)} onRetry={() => void users.refetch()} /> : <DataTable data={users.data ?? []} columns={columns} emptyText={t('admin.noUsers')} />}
      </Panel>
      <div className="flex items-center gap-2 text-xs text-neutral-500"><ShieldCheck size={15} />{t('admin.selfProtection')}</div>
    </div>
  )
}

function validPublicBaseURL(value: string) {
  try {
    const parsed = new URL(value.trim())
    return (parsed.protocol === 'http:' || parsed.protocol === 'https:')
      && Boolean(parsed.hostname)
      && !parsed.username
      && !parsed.password
      && (parsed.pathname === '/' || parsed.pathname === '')
      && !parsed.search
      && !parsed.hash
  } catch {
    return false
  }
}

function validHTTPProxyURL(value: string) {
  try {
    const parsed = new URL(value.trim())
    return (parsed.protocol === 'http:' || parsed.protocol === 'https:')
      && Boolean(parsed.hostname)
      && (parsed.pathname === '/' || parsed.pathname === '')
      && !parsed.search
      && !parsed.hash
  } catch {
    return false
  }
}
