import { Input } from '@appica/ui-react/input'
import { Spinner } from '@appica/ui-react/spinner'
import { ArrowLeft, Lock, User, UserPlus } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { FormField } from '@/components/common/form-field'
import { Message } from '@/components/common/feedback'
import { Panel } from '@/components/common/panel'
import { useAuth } from '@/features/auth/use-auth'
import { errorMessage } from '@/lib/error-message'

export function RegisterPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [validation, setValidation] = useState('')
  const status = useQuery({ queryKey: ['registration-status'], queryFn: api.registrationStatus })
  const { register } = useAuth()
  const navigate = useNavigate()
  const { t } = useTranslation()

  async function submit(event: FormEvent) {
    event.preventDefault()
    setValidation('')
    if (password !== confirmPassword) return setValidation(t('auth.register.mismatch'))
    if (password.length < 6) return setValidation(t('auth.register.tooShort'))
    await register.mutateAsync({ username: username.trim(), password })
    navigate('/', { replace: true })
  }

  return (
    <div className="grid min-h-[calc(100vh-8rem)] place-items-center px-4 py-12">
      <Panel className="rise-in w-full max-w-md">
        {status.isPending ? <div className="grid min-h-64 place-items-center"><Spinner className="size-8" /></div> : !status.data?.allowed ? (
          <div className="py-7 text-center"><span className="mx-auto grid size-16 place-items-center rounded-2xl bg-neutral-100 text-neutral-500 dark:bg-neutral-800"><Lock size={30} /></span><h1 className="mt-5 text-xl font-bold">{t('auth.register.closed')}</h1><p className="mt-2 text-sm text-neutral-500">{t('auth.register.closedDescription')}</p><AppButton className="mt-6" render={<Link to="/login" />}><ArrowLeft size={17} />{t('auth.register.backToLogin')}</AppButton></div>
        ) : (
          <><div className="mb-7 text-center"><span className="mx-auto grid size-14 place-items-center rounded-2xl bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300"><UserPlus size={28} /></span><h1 className="mt-4 text-2xl font-bold tracking-tight">{t('auth.register.title')}</h1><p className="mt-1 text-sm text-neutral-500">{t('auth.register.description')}</p></div>
          <form className="space-y-5" onSubmit={(event) => void submit(event)}>
            {(validation || register.error) && <Message variant="error" title={validation || errorMessage(register.error, t('auth.register.failed'))} />}
            <FormField label={t('auth.register.username')} description={t('auth.register.usernameHint')}><Input value={username} onChange={(event) => setUsername(event.target.value)} startSlot={<User size={17} />} autoComplete="username" required minLength={3} maxLength={50} /></FormField>
            <FormField label={t('auth.register.password')} description={t('auth.register.passwordHint')}><Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" required minLength={6} /></FormField>
            <FormField label={t('auth.register.confirmPassword')}><Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" required /></FormField>
            <AppButton className="w-full justify-center" type="submit" size="lg" disabled={register.isPending}>{register.isPending ? t('auth.register.submitting') : <><UserPlus size={18} />{t('auth.register.submit')}</>}</AppButton>
          </form><p className="mt-6 text-center text-sm text-neutral-500">{t('auth.register.hasAccount')} <Link className="font-semibold text-emerald-700 hover:underline dark:text-emerald-400" to="/login">{t('auth.register.login')}</Link></p></>
        )}
      </Panel>
    </div>
  )
}
