import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Switch } from '@appica/ui-react/switch'
import { Spinner } from '@appica/ui-react/spinner'
import { Refresh, Settings, ShieldCheck, User, Users } from '@appica/icons-react'
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
  const [retentionDays, setRetentionDays] = useState('30')
  const [httpProxyURL, setHTTPProxyURL] = useState('')
  const users = useQuery({ queryKey: ['admin', 'users'], queryFn: api.users })
  const settings = useQuery({ queryKey: ['admin', 'settings'], queryFn: api.systemSettings })
  const updateSettings = useMutation({
    mutationFn: api.updateSystemSettings,
    onSuccess: (_, variables) => {
      if (variables.http_proxy_url !== undefined || variables.clear_http_proxy) setHTTPProxyURL('')
      void queryClient.invalidateQueries({ queryKey: ['admin', 'settings'] })
    },
  })
  const updateRole = useMutation({
    mutationFn: ({ id, isAdmin }: { id: number; isAdmin: boolean }) => api.setUserRole(id, isAdmin),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })

  useEffect(() => {
    if (settings.data) setRetentionDays(String(settings.data.api_log_retention_days))
  }, [settings.data])

  const parsedRetentionDays = Number(retentionDays)
  const retentionDaysValid = Number.isInteger(parsedRetentionDays) && parsedRetentionDays >= 1 && parsedRetentionDays <= 3650

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
            <div className="flex items-center justify-between gap-5 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
              <div><h3 className="font-semibold">允许用户注册</h3><p className="mt-1 text-sm text-neutral-500">开启后，访客可以自行创建账户并登录。</p></div>
              <Switch size="lg" aria-label="允许用户注册" checked={settings.data?.allow_register ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ allow_register: checked })} />
            </div>
            <div className="flex items-center justify-between gap-5 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
              <div><h3 className="font-semibold">允许推送到私有网络</h3><p className="mt-1 text-sm text-neutral-500">默认仅允许公网 HTTPS。开启后可访问内网地址和 HTTP，请仅在可信部署环境中使用。</p></div>
              <Switch size="lg" aria-label="允许推送到私有网络" checked={settings.data?.allow_private_notification_targets ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ allow_private_notification_targets: checked })} />
            </div>
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="min-w-0 flex-1 space-y-2 text-sm font-medium">
                <span>HTTP 代理</span>
                <Input type="password" autoComplete="off" value={httpProxyURL} disabled={updateSettings.isPending} onChange={(event) => setHTTPProxyURL(event.target.value)} placeholder={settings.data?.http_proxy_configured ? `已配置：${settings.data.http_proxy_display || '地址已隐藏'}（留空保持）` : 'http://user:password@proxy.example.com:7890'} />
                <span className="block text-xs text-neutral-500">支持 HTTP/HTTPS 代理，可包含用户名和密码。地址不会通过设置接口返回。</span>
              </label>
              <div className="flex flex-wrap gap-2">
                {settings.data?.http_proxy_configured && <ConfirmDialog title="清空 HTTP 代理？" description="启用了“使用系统代理”的推送渠道将无法发送，直到重新配置代理或关闭渠道代理开关。" confirmLabel="清空代理" destructive busy={updateSettings.isPending} onConfirm={() => updateSettings.mutate({ clear_http_proxy: true })} trigger={<AppButton variant="outline" disabled={updateSettings.isPending}>清空代理</AppButton>} />}
                <AppButton disabled={updateSettings.isPending || !httpProxyURL.trim()} onClick={() => updateSettings.mutate({ http_proxy_url: httpProxyURL.trim() })}>{updateSettings.isPending ? '正在保存…' : '保存代理'}</AppButton>
              </div>
            </div>
            <div className="flex flex-col gap-4 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900 sm:flex-row sm:items-end sm:justify-between">
              <label className="max-w-xs flex-1 space-y-2 text-sm font-medium">
                <span>日志保留天数</span>
                <Input type="number" min={1} max={3650} step={1} value={retentionDays} disabled={updateSettings.isPending} onChange={(event) => setRetentionDays(event.target.value)} />
                <span className={`block text-xs ${retentionDaysValid ? 'text-neutral-500' : 'text-red-600'}`}>{retentionDaysValid ? '接口日志和业务日志统一保留，默认 30 天，可设置 1–3650 天。' : '请输入 1–3650 之间的整数。'}</span>
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
