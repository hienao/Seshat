import { Button } from '@appica/ui-react/button'
import { Logout, Menu, User } from '@appica/icons-react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '@/features/auth/use-auth'
import { getCurrentPageTitle } from './app-navigation'
import { ThemeToggle } from './theme-toggle'

export function AppHeader({ navigationOpen = false, onOpenNavigation }: {
  navigationOpen?: boolean
  onOpenNavigation?: () => void
}) {
  const { user, logout } = useAuth()
  const location = useLocation()

  if (!user) {
    return (
      <header className="sticky top-0 z-40 border-b border-neutral-200/70 bg-white/82 backdrop-blur-xl dark:border-neutral-800 dark:bg-neutral-950/82">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <Link to="/" className="group flex items-center gap-2.5">
            <span className="grid size-9 place-items-center rounded-xl bg-emerald-700 font-mono text-sm font-bold text-white shadow-lg shadow-emerald-900/20 transition group-hover:-rotate-3">S</span>
            <span className="font-bold tracking-tight text-neutral-950 dark:text-white">Seshat</span>
          </Link>
          <div className="flex items-center gap-1.5">
            <ThemeToggle />
            <div className="flex gap-1 sm:gap-2">
              <Button render={<Link to="/login" />} variant="ghost">登录</Button>
              <Button render={<Link to="/register" />}>注册</Button>
            </div>
          </div>
        </div>
      </header>
    )
  }

  return (
    <header className="sticky top-0 z-30 border-b border-neutral-200/70 bg-white/82 backdrop-blur-xl dark:border-neutral-800 dark:bg-neutral-950/82">
      <div className="flex h-16 items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-2.5">
          <Button className="lg:hidden" variant="ghost" size="icon-md" aria-label="打开导航菜单" aria-expanded={navigationOpen} onClick={onOpenNavigation}>
            <Menu size={20} />
          </Button>
          <p className="truncate text-sm font-semibold text-neutral-900 dark:text-white sm:text-base">{getCurrentPageTitle(location.pathname)}</p>
        </div>
        <div className="flex items-center gap-1.5">
          <ThemeToggle />
          <span className="hidden items-center gap-2 rounded-lg bg-neutral-100 px-3 py-2 text-sm dark:bg-neutral-800 md:flex"><User size={16} />{user.username}</span>
          <Button className="hidden gap-2 md:flex" variant="ghost" onClick={() => logout.mutate()} disabled={logout.isPending}><Logout size={17} />退出</Button>
        </div>
      </div>
    </header>
  )
}
