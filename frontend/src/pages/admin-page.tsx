import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { Spinner } from '@appica/ui-react/spinner'
import { Eye, EyeOff, Refresh, Settings, ShieldCheck, User, Users } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useEffect, useMemo, useState } from 'react'
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

export function AdminPage() {
  const currentUser = useAuthStore((state) => state.user)
  const queryClient = useQueryClient()
  const [retentionDays, setRetentionDays] = useState('7')
  const [httpProxyURL, setHTTPProxyURL] = useState('')
  const [tmdbToken, setTMDBToken] = useState('')
  const [tmdbTokenVisible, setTMDBTokenVisible] = useState(false)
  const [connectionTestMessage, setConnectionTestMessage] = useState('')
  const [publicBaseURL, setPublicBaseURL] = useState('')
  const users = useQuery({ queryKey: ['admin', 'users'], queryFn: api.users })
  const settings = useQuery({ queryKey: ['admin', 'settings'], queryFn: api.systemSettings })
  const updateSettings = useMutation({
    mutationFn: api.updateSystemSettings,
    onSuccess: (_, variables) => {
      if (variables.clear_tmdb_token) setTMDBTokenVisible(false)
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
    onSuccess: (result) => setConnectionTestMessage(result.message),
  })
  const testHTTPProxy = useMutation({
    mutationFn: api.testHTTPProxy,
    onMutate: () => setConnectionTestMessage(''),
    onSuccess: (result) => setConnectionTestMessage(result.message),
  })

  useEffect(() => {
    if (settings.data) {
      setRetentionDays(String(settings.data.api_log_retention_days))
      setHTTPProxyURL(settings.data.http_proxy_url || '')
      setTMDBToken(settings.data.tmdb_read_access_token || '')
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
      header: '用户',
      cell: ({ row }) => <div className="flex items-center gap-3"><span className="grid size-9 place-items-center rounded-full bg-emerald-50 text-emerald-700 dark:bg-emerald-950"><User size={17} /></span><div><p className="font-semibold">{row.original.username}</p><p className="text-xs text-neutral-500">ID: {row.original.id}</p></div></div>,
    },
    {
      accessorKey: 'is_admin',
      header: '角色',
      cell: ({ row }) => <div className="flex flex-wrap gap-2"><Badge variant={row.original.is_admin ? 'primary' : 'soft'} size="sm">{row.original.is_admin ? '管理员' : '普通用户'}</Badge>{row.original.id === currentUser?.id && <Badge variant="outline" size="sm">当前用户</Badge>}</div>,
    },
    { accessorKey: 'created_at', header: '注册时间', cell: ({ getValue }) => <span className="whitespace-nowrap text-sm text-neutral-500">{String(getValue() || '-')}</span> },
    {
      id: 'actions', header: '操作',
      cell: ({ row }) => row.original.id === currentUser?.id ? <span className="text-xs text-neutral-400">不可修改自己</span> : (
        <ConfirmDialog
          title={row.original.is_admin ? '撤销管理员权限？' : '授予管理员权限？'}
          description={`${row.original.username} ${row.original.is_admin ? '将无法继续访问系统管理功能。' : '将可以管理用户和系统设置。'}`}
          confirmLabel={row.original.is_admin ? '撤销权限' : '授予权限'}
          destructive={row.original.is_admin}
          busy={updateRole.isPending}
          onConfirm={() => updateRole.mutate({ id: row.original.id, isAdmin: !row.original.is_admin })}
          trigger={<AppButton size="sm" variant={row.original.is_admin ? 'destructive' : 'outline'}>{row.original.is_admin ? '撤销管理员' : '设为管理员'}</AppButton>}
        />
      ),
    },
  ], [currentUser?.id, updateRole.isPending])

  return (
    <div className="mx-auto max-w-6xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Administration" title="系统管理" description="集中管理系统策略、日志保留周期与用户权限。" />
      <Panel title="系统设置" description="设置保存后立即生效" icon={<Settings size={20} />}>
        {settings.isPending ? <div className="grid min-h-28 place-items-center"><Spinner className="size-7" /></div> : settings.error ? <ErrorState message={errorMessage(settings.error)} onRetry={() => void settings.refetch()} /> : (
          <div className="space-y-3">
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="min-w-0 flex-1 space-y-2 text-sm font-medium">
                <span>对外访问地址</span>
                <Input type="url" value={publicBaseURL} disabled={updateSettings.isPending} onChange={(event) => setPublicBaseURL(event.target.value)} placeholder="https://seshat.example.com" />
                <span className={`block text-xs ${publicBaseURL && !publicBaseURLValid ? 'text-red-600' : 'text-neutral-500'}`}>推送消息使用该地址生成无需登录的详情链接，请填写用户可访问的 Seshat 完整地址。</span>
              </label>
              <AppButton disabled={updateSettings.isPending || !publicBaseURLValid || publicBaseURL.trim().replace(/\/$/, '') === settings.data?.public_base_url} onClick={() => updateSettings.mutate({ public_base_url: publicBaseURL.trim().replace(/\/$/, '') })}>{updateSettings.isPending ? '正在保存…' : '保存访问地址'}</AppButton>
            </div>
            <div className="flex items-center justify-between gap-5 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
              <div><h3 className="font-semibold">允许用户注册</h3><p className="mt-1 text-sm text-neutral-500">开启后，访客可以自行创建账户并登录。</p></div>
              <Switch size="lg" aria-label="允许用户注册" checked={settings.data?.allow_register ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ allow_register: checked })} />
            </div>
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="min-w-0 flex-1 space-y-2 text-sm font-medium">
                <span>HTTP 代理</span>
                <Input type="url" autoComplete="off" value={httpProxyURL} disabled={updateSettings.isPending || testHTTPProxy.isPending || testTMDBConnection.isPending} onChange={(event) => setHTTPProxyURL(event.target.value)} placeholder="http://proxy.example.com:7890" />
                <span className={`block text-xs ${httpProxyURL && !httpProxyURLValid ? 'text-red-600' : 'text-neutral-500'}`}>{httpProxyURL && !httpProxyURLValid ? '请输入有效的 http:// 或 https:// 代理地址。' : '支持 HTTP/HTTPS 代理，可在保存前独立测试代理连通性。'}</span>
              </label>
              <div className="flex flex-wrap gap-2">
                {settings.data?.http_proxy_configured && <ConfirmDialog title="清空 HTTP 代理？" description="启用了系统代理的推送渠道和 TMDB API 请求将无法访问，直到重新配置代理或关闭对应代理开关。" confirmLabel="清空代理" destructive busy={updateSettings.isPending} onConfirm={() => updateSettings.mutate({ clear_http_proxy: true })} trigger={<AppButton variant="outline" disabled={updateSettings.isPending}>清空代理</AppButton>} />}
                <AppButton variant="outline" disabled={testHTTPProxy.isPending || !httpProxyURLValid} onClick={() => testHTTPProxy.mutate({ http_proxy_url: httpProxyURL.trim() })}>{testHTTPProxy.isPending ? '正在测试…' : '测试代理'}</AppButton>
                <AppButton disabled={updateSettings.isPending || !httpProxyURLValid} onClick={() => updateSettings.mutate({ http_proxy_url: httpProxyURL.trim() })}>{updateSettings.isPending ? '正在保存…' : '保存代理'}</AppButton>
              </div>
            </div>
            <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                <div className="min-w-0 flex-1 space-y-4">
                <label className="block space-y-2 text-sm font-medium">
                  <span>TMDB API Read Access Token</span>
                  <Input
                    type={tmdbTokenVisible ? 'text' : 'password'}
                    autoComplete="off"
                    value={tmdbToken}
                    disabled={updateSettings.isPending || testTMDBConnection.isPending}
                    onChange={(event) => setTMDBToken(event.target.value)}
                    placeholder="用于按 Provider ID 获取海报和补充简介"
                    endSlot={tmdbToken && <button type="button" disabled={updateSettings.isPending} className="rounded p-1 text-neutral-500 hover:text-neutral-900 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:text-white" aria-label={tmdbTokenVisible ? '隐藏 TMDB Token' : '显示 TMDB Token'} title={tmdbTokenVisible ? '隐藏 Token' : '显示 Token'} onClick={() => setTMDBTokenVisible((visible) => !visible)}>{tmdbTokenVisible ? <EyeOff size={17} /> : <Eye size={17} />}</button>}
                  />
                  <span className="block text-xs text-neutral-500">可选。配置后会补充 Jellyfin/Emby 媒体海报与简介；默认隐藏，可点击输入框右侧图标查看。</span>
                </label>
                <div className="flex items-center justify-between gap-4 rounded-lg border border-neutral-200 p-3 dark:border-neutral-800">
                  <div><p className="text-sm font-medium">通过系统 HTTP 代理请求 TMDB</p><p className="mt-1 text-xs text-neutral-500">{settings.data?.http_proxy_configured ? '仅 TMDB API 元数据请求使用上方代理。' : '开启前请先配置上方 HTTP 代理。'}</p></div>
                  <Switch aria-label="TMDB 使用系统 HTTP 代理" checked={settings.data?.tmdb_use_proxy ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ tmdb_use_proxy: checked })} />
                </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  {settings.data?.tmdb_configured && <ConfirmDialog title="清空 TMDB Token？" description="清空后不再获取新的外部媒体资料，已缓存内容会保留至到期。" confirmLabel="清空 Token" destructive busy={updateSettings.isPending} onConfirm={() => updateSettings.mutate({ clear_tmdb_token: true })} trigger={<AppButton variant="outline" disabled={updateSettings.isPending}>清空 Token</AppButton>} />}
                  <AppButton variant="outline" disabled={testTMDBConnection.isPending || !tmdbToken.trim() || Boolean(settings.data?.tmdb_use_proxy && !httpProxyURLValid)} onClick={() => testTMDBConnection.mutate({ tmdb_read_access_token: tmdbToken.trim(), http_proxy_url: httpProxyURL.trim(), use_proxy: settings.data?.tmdb_use_proxy ?? false })}>{testTMDBConnection.isPending ? '正在测试…' : '测试连接'}</AppButton>
                  <AppButton disabled={updateSettings.isPending || !tmdbToken.trim()} onClick={() => updateSettings.mutate({ tmdb_read_access_token: tmdbToken.trim() })}>{updateSettings.isPending ? '正在保存…' : '保存 Token'}</AppButton>
                </div>
              </div>
              <p className="mt-3 border-t border-neutral-200 pt-3 text-xs text-neutral-400 dark:border-neutral-800">TMDB API 使用声明：This product uses the TMDB API but is not endorsed or certified by TMDB.</p>
            </div>
            {connectionTestMessage && <Message variant="success" title={connectionTestMessage} />}
            {testHTTPProxy.error && <Message variant="error" title={errorMessage(testHTTPProxy.error, '代理测试失败')} />}
            {testTMDBConnection.error && <Message variant="error" title={errorMessage(testTMDBConnection.error, '连接测试失败')} />}
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="max-w-xs flex-1 space-y-2 text-sm font-medium">
                <span>日志保留天数</span>
                <Input type="number" min={1} max={30} step={1} value={retentionDays} disabled={updateSettings.isPending} onChange={(event) => setRetentionDays(event.target.value)} />
                <span className={`block text-xs ${retentionDaysValid ? 'text-neutral-500' : 'text-red-600'}`}>{retentionDaysValid ? '接口日志、业务日志和媒体资料缓存统一保留，默认 7 天，可设置 1–30 天。' : '请输入 1–30 之间的整数。'}</span>
              </label>
              <AppButton disabled={updateSettings.isPending || !retentionDaysValid || parsedRetentionDays === settings.data?.api_log_retention_days} onClick={() => updateSettings.mutate({ api_log_retention_days: parsedRetentionDays })}>{updateSettings.isPending ? '正在保存…' : '保存保留周期'}</AppButton>
            </div>
          </div>
        )}
        {updateSettings.error && <div className="mt-4"><Message variant="error" title={errorMessage(updateSettings.error, '保存设置失败')} /></div>}
      </Panel>
      <Panel title="用户管理" description={`共 ${users.data?.length ?? 0} 个账户`} icon={<Users size={20} />} action={<AppButton size="sm" variant="ghost" disabled={users.isFetching} onClick={() => void users.refetch()}><Refresh size={16} />刷新</AppButton>}>
        {updateRole.error && <div className="mb-4"><Message variant="error" title={errorMessage(updateRole.error, '设置用户角色失败')} /></div>}
        {users.isPending ? <div className="grid min-h-56 place-items-center"><Spinner className="size-8" /></div> : users.error ? <ErrorState message={errorMessage(users.error)} onRetry={() => void users.refetch()} /> : <DataTable data={users.data ?? []} columns={columns} emptyText="还没有用户" />}
      </Panel>
      <div className="flex items-center gap-2 text-xs text-neutral-500"><ShieldCheck size={15} />为避免误锁定，当前登录用户不能撤销自己的管理员权限。</div>
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
