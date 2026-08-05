import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Calendar, Check, Key, Lock, User } from '@appica/icons-react'
import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { FormField } from '@/components/common/form-field'
import { Message } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { useAuth } from '@/features/auth/use-auth'
import { errorMessage } from '@/lib/error-message'

export function ProfilePage() {
  const { user, logout } = useAuth()
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [validation, setValidation] = useState('')
  const [success, setSuccess] = useState(false)
  const changePassword = useMutation({ mutationFn: () => api.changePassword(oldPassword, newPassword) })

  async function submit(event: FormEvent) {
    event.preventDefault(); setValidation(''); setSuccess(false)
    if (newPassword !== confirmPassword) return setValidation('两次输入的新密码不一致')
    if (newPassword.length < 6) return setValidation('新密码长度至少 6 位')
    await changePassword.mutateAsync()
    setOldPassword(''); setNewPassword(''); setConfirmPassword(''); setSuccess(true)
  }

  return (
    <div className="mx-auto max-w-4xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Account" title="个人中心" description="查看账户信息并维护登录凭据。" />
      <Panel title="账户信息" description="这些信息来自当前服务端会话" icon={<User size={20} />}>
        <dl className="grid gap-4 sm:grid-cols-3">
          <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900"><dt className="text-xs text-neutral-500">用户名</dt><dd className="mt-2 font-semibold">{user?.username}</dd></div>
          <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900"><dt className="text-xs text-neutral-500">用户 ID</dt><dd className="mt-2 font-mono font-semibold">#{user?.id}</dd></div>
          <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900"><dt className="flex items-center gap-1 text-xs text-neutral-500"><Calendar size={14} />注册时间</dt><dd className="mt-2 text-sm font-semibold">{user?.created_at || '-'}</dd></div>
        </dl>
        <div className="mt-4"><Badge variant={user?.is_admin ? 'primary' : 'soft'}>{user?.is_admin ? '系统管理员' : '普通用户'}</Badge></div>
      </Panel>
      <Panel title="修改密码" description="修改成功后需要使用新密码重新登录" icon={<Key size={20} />}>
        <form className="max-w-xl space-y-5" onSubmit={(event) => void submit(event)}>
          {(validation || changePassword.error) && <Message variant="error" title={validation || errorMessage(changePassword.error, '修改密码失败')} />}
          {success && <Message variant="success" title="密码修改成功" description="请退出后使用新密码重新登录。" />}
          <FormField label="当前密码"><Input type="password" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="current-password" required /></FormField>
          <FormField label="新密码" description="至少 6 位"><Input type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" minLength={6} required /></FormField>
          <FormField label="确认新密码"><Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" required /></FormField>
          <div className="flex flex-wrap gap-3"><AppButton type="submit" disabled={changePassword.isPending}><Check size={17} />{changePassword.isPending ? '正在修改…' : '修改密码'}</AppButton>{success && <AppButton type="button" variant="outline" onClick={() => logout.mutate()}>立即重新登录</AppButton>}</div>
        </form>
      </Panel>
    </div>
  )
}
