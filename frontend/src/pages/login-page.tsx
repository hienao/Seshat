import { AlertCircle, Lock, Login, User } from '@appica/icons-react'
import { Input } from '@appica/ui-react/input'
import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AppButton } from '@/components/common/app-button'
import { FormField } from '@/components/common/form-field'
import { Message } from '@/components/common/feedback'
import { Panel } from '@/components/common/panel'
import { useAuth } from '@/features/auth/use-auth'
import { errorMessage } from '@/lib/error-message'

export function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()

  async function submit(event: FormEvent) {
    event.preventDefault()
    const profile = await login.mutateAsync({ username: username.trim(), password })
    const requested = (location.state as { from?: string } | null)?.from
    navigate(profile.requires_admin_setup ? '/' : requested || (profile.is_admin ? '/admin' : '/'), { replace: true })
  }

  return (
    <div className="grid min-h-[calc(100vh-8rem)] place-items-center px-4 py-12">
      <Panel className="rise-in w-full max-w-md">
        <div className="mb-7 text-center"><span className="mx-auto grid size-14 place-items-center rounded-2xl bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300"><Login size={28} /></span><h1 className="mt-4 text-2xl font-bold tracking-tight">{t('auth.login.title')}</h1><p className="mt-1 text-sm text-neutral-500">{t('auth.login.description')}</p></div>
        <form className="space-y-5" onSubmit={(event) => void submit(event)}>
          {login.error && <Message variant="error" title={errorMessage(login.error, t('auth.login.failed'))} />}
          <FormField label={t('auth.login.username')}><Input value={username} onChange={(event) => setUsername(event.target.value)} startSlot={<User size={17} />} placeholder={t('auth.login.usernamePlaceholder')} autoComplete="username" required minLength={3} /></FormField>
          <FormField label={t('auth.login.password')}><Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} startSlot={<Lock size={17} />} placeholder={t('auth.login.passwordPlaceholder')} autoComplete="current-password" required /></FormField>
          <AppButton className="w-full justify-center" type="submit" size="lg" disabled={login.isPending}>{login.isPending ? t('auth.login.submitting') : <><Login size={18} />{t('auth.login.submit')}</>}</AppButton>
        </form>
        <p className="mt-5 rounded-lg bg-amber-50 px-3 py-2 text-center text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">{t('auth.login.bootstrapHint')}</p>
        <p className="mt-6 text-center text-sm text-neutral-500">{t('auth.login.noAccount')} <Link className="font-semibold text-emerald-700 hover:underline dark:text-emerald-400" to="/register">{t('auth.login.register')}</Link></p>
        <p className="sr-only"><AlertCircle />{t('auth.login.failed')}</p>
      </Panel>
    </div>
  )
}
