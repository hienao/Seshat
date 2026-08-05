import { AlertCircle, Lock, Login, User } from '@appica/icons-react'
import { Input } from '@appica/ui-react/input'
import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
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

  async function submit(event: FormEvent) {
    event.preventDefault()
    const profile = await login.mutateAsync({ username: username.trim(), password })
    const requested = (location.state as { from?: string } | null)?.from
    navigate(requested || (profile.is_admin ? '/admin' : '/'), { replace: true })
  }

  return (
    <div className="grid min-h-[calc(100vh-8rem)] place-items-center px-4 py-12">
      <Panel className="rise-in w-full max-w-md">
        <div className="mb-7 text-center"><span className="mx-auto grid size-14 place-items-center rounded-2xl bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300"><Login size={28} /></span><h1 className="mt-4 text-2xl font-bold tracking-tight">欢迎回来</h1><p className="mt-1 text-sm text-neutral-500">登录账户以继续使用 BaseGoApp</p></div>
        <form className="space-y-5" onSubmit={(event) => void submit(event)}>
          {login.error && <Message variant="error" title={errorMessage(login.error, '登录失败')} />}
          <FormField label="用户名"><Input value={username} onChange={(event) => setUsername(event.target.value)} startSlot={<User size={17} />} placeholder="请输入用户名" autoComplete="username" required minLength={3} /></FormField>
          <FormField label="密码"><Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} startSlot={<Lock size={17} />} placeholder="请输入密码" autoComplete="current-password" required /></FormField>
          <AppButton className="w-full justify-center" type="submit" size="lg" disabled={login.isPending}>{login.isPending ? '正在登录…' : <><Login size={18} />登录</>}</AppButton>
        </form>
        <p className="mt-6 text-center text-sm text-neutral-500">还没有账户？ <Link className="font-semibold text-emerald-700 hover:underline dark:text-emerald-400" to="/register">立即注册</Link></p>
        <p className="sr-only"><AlertCircle />登录失败时页面会显示错误原因</p>
      </Panel>
    </div>
  )
}
