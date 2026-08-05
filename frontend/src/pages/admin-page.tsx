import { Badge } from '@appica/ui-react/badge'
import { Switch } from '@appica/ui-react/switch'
import { Spinner } from '@appica/ui-react/spinner'
import { Refresh, Settings, ShieldCheck, User, Users } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo } from 'react'
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
  const users = useQuery({ queryKey: ['admin', 'users'], queryFn: api.users })
  const settings = useQuery({ queryKey: ['admin', 'settings'], queryFn: api.systemSettings })
  const updateSettings = useMutation({
    mutationFn: api.updateSystemSettings,
    onSuccess: (_, variables) => queryClient.setQueryData(['admin', 'settings'], variables),
  })
  const updateRole = useMutation({
    mutationFn: ({ id, isAdmin }: { id: number; isAdmin: boolean }) => api.setUserRole(id, isAdmin),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })

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
      <PageHeader eyebrow="Administration" title="系统管理" description="集中管理开放注册策略与用户权限。" />
      <Panel title="系统设置" description="设置会立即应用到公开注册页面" icon={<Settings size={20} />}>
        {settings.isPending ? <div className="grid min-h-28 place-items-center"><Spinner className="size-7" /></div> : settings.error ? <ErrorState message={errorMessage(settings.error)} onRetry={() => void settings.refetch()} /> : (
          <div className="flex items-center justify-between gap-5 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
            <div><h3 className="font-semibold">允许用户注册</h3><p className="mt-1 text-sm text-neutral-500">开启后，访客可以自行创建账户并登录。</p></div>
            <Switch size="lg" aria-label="允许用户注册" checked={settings.data?.allow_register ?? false} disabled={updateSettings.isPending} onCheckedChange={(checked) => updateSettings.mutate({ allow_register: checked })} />
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
