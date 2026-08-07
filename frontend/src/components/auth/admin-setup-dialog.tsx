import { Input } from '@appica/ui-react/input'
import { Lock, ShieldCheck, User } from '@appica/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { Message } from '@/components/common/feedback'
import { FormField } from '@/components/common/form-field'
import { errorMessage } from '@/lib/error-message'
import { useAuthStore } from '@/stores/auth'

export function AdminSetupDialog() {
  const user = useAuthStore((state) => state.user)
  const setUser = useAuthStore((state) => state.setUser)
  const setAccessToken = useAuthStore((state) => state.setAccessToken)
  const queryClient = useQueryClient()
  const [username, setUsername] = useState(user?.username ?? 'admin')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [validation, setValidation] = useState('')
  const setup = useMutation({
    mutationFn: async () => {
      const session = await api.setupAdmin(username.trim(), password)
      setAccessToken(session.token)
      return api.profile()
    },
    onSuccess: (profile) => {
      setUser(profile)
      queryClient.setQueryData(['profile'], profile)
      void queryClient.invalidateQueries({ predicate: (query) => query.queryKey[0] !== 'profile' })
    },
  })

  if (!user?.requires_admin_setup) return null

  async function submit(event: FormEvent) {
    event.preventDefault()
    setValidation('')
    if (username.trim().length < 3) return setValidation('管理员用户名至少 3 位')
    if (password.length < 6) return setValidation('管理员密码至少 6 位')
    if (password !== confirmPassword) return setValidation('两次输入的密码不一致')
    try {
      await setup.mutateAsync()
    } catch {
      // 错误由 mutation 状态展示在弹窗内。
    }
  }

  return (
    <div className="fixed inset-0 z-[100] grid place-items-center bg-black/60 p-4 backdrop-blur-sm">
      <div role="dialog" aria-modal="true" aria-labelledby="admin-setup-title" className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-2xl dark:bg-neutral-950 sm:p-8">
        <div className="mb-6 flex gap-4">
          <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300"><ShieldCheck size={24} /></span>
          <div><p className="text-xs font-semibold uppercase tracking-widest text-amber-700">First-time setup</p><h2 id="admin-setup-title" className="mt-1 text-xl font-bold">创建正式管理员</h2><p className="mt-1 text-sm text-neutral-500">当前使用的是一次性 admin/admin 凭据，完成设置后将立即失效。</p></div>
        </div>
        <form className="space-y-5" onSubmit={(event) => void submit(event)}>
          {(validation || setup.error) && <Message variant="error" title={validation || errorMessage(setup.error, '管理员设置失败')} />}
          <FormField label="管理员用户名" description="3–50 个字符"><Input value={username} onChange={(event) => setUsername(event.target.value)} startSlot={<User size={17} />} autoComplete="username" minLength={3} maxLength={50} required /></FormField>
          <FormField label="管理员密码" description="至少 6 位"><Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" minLength={6} required /></FormField>
          <FormField label="确认管理员密码"><Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" minLength={6} required /></FormField>
          <AppButton className="w-full justify-center" type="submit" size="lg" disabled={setup.isPending}>{setup.isPending ? '正在创建…' : '创建管理员并继续'}</AppButton>
        </form>
      </div>
    </div>
  )
}
