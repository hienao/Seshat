import { Button } from '@appica/ui-react/button'
import { Logout, X } from '@appica/icons-react'
import { Link, NavLink } from 'react-router-dom'
import { useAuth } from '@/features/auth/use-auth'
import { getNavigationSections, profileItem, type AppNavigationItem } from './app-navigation'

const navigationClass = ({ isActive }: { isActive: boolean }) =>
  `group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors ${
    isActive
      ? 'bg-emerald-50 text-emerald-900 dark:bg-emerald-950/70 dark:text-emerald-200'
      : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-950 dark:text-neutral-400 dark:hover:bg-neutral-900 dark:hover:text-white'
  }`

function NavigationLink({ item, onNavigate }: { item: AppNavigationItem; onNavigate?: () => void }) {
  const Icon = item.icon
  return (
    <NavLink to={item.to} end={item.end} className={navigationClass} onClick={onNavigate}>
      <Icon className="shrink-0" size={18} />
      <span>{item.label}</span>
    </NavLink>
  )
}

export function AppSidebar({ className = '', mobile = false, onClose, onNavigate }: {
  className?: string
  mobile?: boolean
  onClose?: () => void
  onNavigate?: () => void
}) {
  const { user, isAdmin, logout } = useAuth()
  const sections = getNavigationSections(isAdmin)

  return (
    <aside
      className={`w-60 flex-col border-r border-neutral-200/80 bg-white/88 backdrop-blur-xl dark:border-neutral-800 dark:bg-neutral-950/88 ${className}`}
      aria-label="应用导航"
      aria-modal={mobile || undefined}
      role={mobile ? 'dialog' : undefined}
    >
      <div className="flex h-16 shrink-0 items-center justify-between px-4">
        <Link to="/" className="group flex min-w-0 items-center gap-2.5" onClick={onNavigate}>
          <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-emerald-700 font-mono text-sm font-bold text-white shadow-lg shadow-emerald-900/20 transition-transform group-hover:-rotate-3">S</span>
          <span className="truncate font-bold tracking-tight text-neutral-950 dark:text-white">Seshat</span>
        </Link>
        {mobile && (
          <Button variant="ghost" size="icon-md" aria-label="关闭导航菜单" autoFocus onClick={onClose}>
            <X size={20} />
          </Button>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <div className="space-y-6">
          {sections.map((section) => (
            <section key={section.label} aria-labelledby={`navigation-${section.label}`}>
              <p id={`navigation-${section.label}`} className="mb-2 px-3 text-[11px] font-bold uppercase tracking-[0.16em] text-neutral-400 dark:text-neutral-600">
                {section.label}
              </p>
              <div className="space-y-1">
                {section.items.map((item) => <NavigationLink key={item.to} item={item} onNavigate={onNavigate} />)}
              </div>
            </section>
          ))}
        </div>
      </nav>

      <div className="shrink-0 border-t border-neutral-200/80 p-3 dark:border-neutral-800">
        <NavigationLink item={profileItem} onNavigate={onNavigate} />
        <div className="mt-2 flex items-center gap-3 rounded-xl bg-neutral-50 px-3 py-3 dark:bg-neutral-900/70">
          <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-emerald-100 text-sm font-bold text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300">
            {user?.username.slice(0, 1).toUpperCase()}
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold text-neutral-900 dark:text-white">{user?.username}</p>
            <p className="text-xs text-neutral-500">{isAdmin ? '管理员' : '普通用户'}</p>
          </div>
        </div>
        {mobile && (
          <Button className="mt-2 w-full justify-start gap-2" variant="ghost" onClick={() => logout.mutate()} disabled={logout.isPending}>
            <Logout size={17} />退出登录
          </Button>
        )}
      </div>
    </aside>
  )
}
