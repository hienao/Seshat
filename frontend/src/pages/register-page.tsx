import { Input } from '@appica/ui-react/input'
import { Spinner } from '@appica/ui-react/spinner'
import { ArrowLeft, Lock, User, UserPlus } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
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

  async function submit(event: FormEvent) {
    event.preventDefault()
    setValidation('')
    if (password !== confirmPassword) return setValidation('两次输入的密码不一致')
    if (password.length < 6) return setValidation('密码长度至少 6 位')
    await register.mutateAsync({ username: username.trim(), password })
    navigate('/', { replace: true })
  }

  return (
    <div className="grid min-h-[calc(100vh-8rem)] place-items-center px-4 py-12">
      <Panel className="rise-in w-full max-w-md">
        {status.isPending ? <div className="grid min-h-64 place-items-center"><Spinner className="size-8" /></div> : !status.data?.allowed ? (
          <div className="py-7 text-center"><span className="mx-auto grid size-16 place-items-center rounded-2xl bg-neutral-100 text-neutral-500 dark:bg-neutral-800"><Lock size={30} /></span><h1 className="mt-5 text-xl font-bold">注册功能已关闭</h1><p className="mt-2 text-sm text-neutral-500">系统当前不允许新用户注册，请联系管理员。</p><AppButton className="mt-6" render={<Link to="/login" />}><ArrowLeft size={17} />返回登录</AppButton></div>
        ) : (
          <><div className="mb-7 text-center"><span className="mx-auto grid size-14 place-items-center rounded-2xl bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300"><UserPlus size={28} /></span><h1 className="mt-4 text-2xl font-bold tracking-tight">创建账户</h1><p className="mt-1 text-sm text-neutral-500">注册后将自动登录</p></div>
          <form className="space-y-5" onSubmit={(event) => void submit(event)}>
            {(validation || register.error) && <Message variant="error" title={validation || errorMessage(register.error, '注册失败')} />}
            <FormField label="用户名" description="3 至 50 个字符"><Input value={username} onChange={(event) => setUsername(event.target.value)} startSlot={<User size={17} />} autoComplete="username" required minLength={3} maxLength={50} /></FormField>
            <FormField label="密码" description="至少 6 位，建议混合字母、数字和符号"><Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" required minLength={6} /></FormField>
            <FormField label="确认密码"><Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} startSlot={<Lock size={17} />} autoComplete="new-password" required /></FormField>
            <AppButton className="w-full justify-center" type="submit" size="lg" disabled={register.isPending}>{register.isPending ? '正在创建…' : <><UserPlus size={18} />注册</>}</AppButton>
          </form><p className="mt-6 text-center text-sm text-neutral-500">已有账户？ <Link className="font-semibold text-emerald-700 hover:underline dark:text-emerald-400" to="/login">立即登录</Link></p></>
        )}
      </Panel>
    </div>
  )
}
