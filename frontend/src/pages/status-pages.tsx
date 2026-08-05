import { Home, Lock } from '@appica/icons-react'
import { Link } from 'react-router-dom'
import { AppButton } from '@/components/common/app-button'

function StatusPage({ code, title, description }: { code: string; title: string; description: string }) {
  return (
    <div className="grid min-h-[calc(100vh-8rem)] place-items-center px-4 text-center">
      <div><p className="font-mono text-sm font-bold tracking-[0.3em] text-emerald-700 dark:text-emerald-400">{code}</p><span className="mx-auto mt-5 grid size-16 place-items-center rounded-2xl bg-neutral-100 text-neutral-500 dark:bg-neutral-800"><Lock size={28} /></span><h1 className="mt-5 text-3xl font-bold tracking-tight">{title}</h1><p className="mx-auto mt-2 max-w-md text-neutral-500">{description}</p><AppButton className="mt-7" render={<Link to="/" />}><Home size={17} />返回首页</AppButton></div>
    </div>
  )
}

export function ForbiddenPage() { return <StatusPage code="403" title="权限不足" description="当前账户没有访问此页面的权限。如需操作，请联系系统管理员。" /> }
export function NotFoundPage() { return <StatusPage code="404" title="页面不存在" description="请求的地址不存在或已经被移动。" /> }
