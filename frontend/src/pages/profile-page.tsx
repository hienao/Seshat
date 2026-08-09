import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { Calendar, Check, Key, Lock, User } from '@appica/icons-react'
import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { FormField } from '@/components/common/form-field'
import { Message } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { useAuth } from '@/features/auth/use-auth'
import { errorMessage } from '@/lib/error-message'
import { formatDateTime } from '@/i18n/format'

export function ProfilePage() {
  const { user, logout } = useAuth()
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [validation, setValidation] = useState('')
  const [success, setSuccess] = useState(false)
  const changePassword = useMutation({ mutationFn: () => api.changePassword(oldPassword, newPassword) })
  const { t } = useTranslation()

  async function submit(event: FormEvent) {
    event.preventDefault(); setValidation(''); setSuccess(false)
    if (newPassword !== confirmPassword) return setValidation(t('profile.mismatch'))
    if (newPassword.length < 6) return setValidation(t('profile.tooShort'))
    await changePassword.mutateAsync()
    setOldPassword(''); setNewPassword(''); setConfirmPassword(''); setSuccess(true)
  }

  return (
    <div className="mx-auto max-w-4xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow={t('profile.eyebrow')} title={t('profile.title')} description={t('profile.description')} />
      <Panel title={t('profile.account')} description={t('profile.accountDescription')} icon={<User size={20} />}>
        <dl className="grid gap-4 sm:grid-cols-3">
          <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900"><dt className="text-xs text-neutral-500">{t('common.fields.username')}</dt><dd className="mt-2 font-semibold">{user?.username}</dd></div>
          <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900"><dt className="text-xs text-neutral-500">{t('profile.userId')}</dt><dd className="mt-2 font-mono font-semibold">#{user?.id}</dd></div>
          <div className="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900"><dt className="flex items-center gap-1 text-xs text-neutral-500"><Calendar size={14} />{t('profile.registeredAt')}</dt><dd className="mt-2 text-sm font-semibold">{user?.created_at ? formatDateTime(user.created_at) : '-'}</dd></div>
        </dl>
        <div className="mt-4"><Badge variant={user?.is_admin ? 'primary' : 'soft'}>{t(user?.is_admin ? 'profile.administrator' : 'profile.regularUser')}</Badge></div>
      </Panel>
      <Panel title={t('profile.changePassword')} description={t('profile.changePasswordDescription')} icon={<Key size={20} />}>
        <form className="max-w-xl space-y-5" onSubmit={(event) => void submit(event)}>
          {(validation || changePassword.error) && <Message variant="error" title={validation || errorMessage(changePassword.error, t('profile.failed'))} />}
          {success && <Message variant="success" title={t('profile.success')} description={t('profile.successDescription')} />}
          <FormField label={t('profile.currentPassword')}><Input type="password" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="current-password" required /></FormField>
          <FormField label={t('profile.newPassword')} description={t('profile.passwordHint')}><Input type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" minLength={6} required /></FormField>
          <FormField label={t('profile.confirmPassword')}><Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" required /></FormField>
          <div className="flex flex-wrap gap-3"><AppButton type="submit" disabled={changePassword.isPending}><Check size={17} />{changePassword.isPending ? t('profile.submitting') : t('profile.submit')}</AppButton>{success && <AppButton type="button" variant="outline" onClick={() => logout.mutate()}>{t('profile.relogin')}</AppButton>}</div>
        </form>
      </Panel>
    </div>
  )
}
