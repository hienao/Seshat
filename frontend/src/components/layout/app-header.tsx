import { Button } from '@appica/ui-react/button'
import { Home, Logout, Menu, Settings, ShieldCheck, User, X } from '@appica/icons-react'
import { useState } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { useAuth } from '@/features/auth/use-auth'
import { ThemeToggle } from './theme-toggle'

const navClass = ({ isActive }: { isActive: boolean }) =>
  `flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition ${isActive ? 'bg-emerald-50 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300' : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-950 dark:text-neutral-300 dark:hover:bg-neutral-800'}`

export function AppHeader() {
  const [open, setOpen] = useState(false)
  const { user, isAdmin, logout } = useAuth()
  const links = user ? [
    { to: '/', label: '首页', icon: Home },
    { to: '/profile', label: '个人中心', icon: Settings },
    ...(isAdmin ? [{ to: '/admin', label: '系统管理', icon: ShieldCheck }] : []),
  ] : []

  return (
    <header className="sticky top-0 z-40 border-b border-neutral-200/70 bg-white/82 backdrop-blur-xl dark:border-neutral-800 dark:bg-neutral-950/82">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
        <Link to="/" className="group flex items-center gap-2.5" onClick={() => setOpen(false)}>
          <span className="grid size-9 place-items-center rounded-xl bg-emerald-700 font-mono text-sm font-bold text-white shadow-lg shadow-emerald-900/20 transition group-hover:-rotate-3">BG</span>
          <span className="font-bold tracking-tight text-neutral-950 dark:text-white">BaseGoApp</span>
        </Link>
        <nav className="hidden items-center gap-1 md:flex">
          {links.map(({ to, label, icon: Icon }) => <NavLink key={to} to={to} className={navClass}><Icon size={17} />{label}</NavLink>)}
        </nav>
        <div className="flex items-center gap-1.5">
          <ThemeToggle />
          {user ? (
            <>
              <span className="hidden items-center gap-2 rounded-lg bg-neutral-100 px-3 py-2 text-sm dark:bg-neutral-800 sm:flex"><User size={16} />{user.username}</span>
              <Button className="hidden gap-2 sm:flex" variant="ghost" onClick={() => logout.mutate()} disabled={logout.isPending}><Logout size={17} />退出</Button>
            </>
          ) : (
            <div className="hidden gap-2 sm:flex"><Button render={<Link to="/login" />} variant="ghost">登录</Button><Button render={<Link to="/register" />}>注册</Button></div>
          )}
          <Button className="md:hidden" variant="ghost" size="icon-md" aria-label="打开导航" onClick={() => setOpen((value) => !value)}>{open ? <X size={20} /> : <Menu size={20} />}</Button>
        </div>
      </div>
      {open && (
        <div className="border-t border-neutral-200 px-4 py-3 dark:border-neutral-800 md:hidden">
          <nav className="space-y-1">
            {links.map(({ to, label, icon: Icon }) => <NavLink key={to} to={to} className={navClass} onClick={() => setOpen(false)}><Icon size={17} />{label}</NavLink>)}
            {user ? <Button className="mt-2 w-full justify-start gap-2" variant="ghost" onClick={() => logout.mutate()}><Logout size={17} />退出登录</Button> : <><Button className="w-full justify-start" render={<Link to="/login" />} variant="ghost">登录</Button><Button className="mt-1 w-full" render={<Link to="/register" />}>注册</Button></>}
          </nav>
        </div>
      )}
    </header>
  )
}
