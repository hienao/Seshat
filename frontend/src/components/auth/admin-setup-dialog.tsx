import { Input } from '@appica/ui-react/input'
import { Lock, ShieldCheck, User } from '@appica/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation()
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
    if (username.trim().length < 3) return setValidation(t('auth.setup.usernameTooShort'))
    if (password.length < 6) return setValidation(t('auth.setup.passwordTooShort'))
    if (password !== confirmPassword) return setValidation(t('auth.setup.mismatch'))
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
          <div><p className="text-xs font-semibold uppercase tracking-widest text-amber-700">{t('auth.setup.eyebrow')}</p><h2 id="admin-setup-title" className="mt-1 text-xl font-bold">{t('auth.setup.title')}</h2><p className="mt-1 text-sm text-neutral-500">{t('auth.setup.description')}</p></div>
        </div>
        <form className="space-y-5" onSubmit={(event) => void submit(event)}>
          {(validation || setup.error) && <Message variant="error" title={validation || errorMessage(setup.error, t('auth.setup.failed'))} />}
          <FormField label={t('auth.setup.username')} description={t('auth.setup.usernameHint')}><Input value={username} onChange={(event) => setUsername(event.target.value)} startSlot={<User size={17} />} autoComplete="username" minLength={3} maxLength={50} required /></FormField>
          <FormField label={t('auth.setup.password')} description={t('auth.setup.passwordHint')}><Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" minLength={6} required /></FormField>
          <FormField label={t('auth.setup.confirmPassword')}><Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" minLength={6} required /></FormField>
          <AppButton className="w-full justify-center" type="submit" size="lg" disabled={setup.isPending}>{setup.isPending ? t('auth.setup.submitting') : t('auth.setup.submit')}</AppButton>
        </form>
      </div>
    </div>
  )
}
