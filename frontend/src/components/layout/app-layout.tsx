import { Outlet, useLocation } from 'react-router-dom'
import { Spinner } from '@appica/ui-react/spinner'
import { Suspense, useEffect, useState } from 'react'
import { useAuthStore } from '@/stores/auth'
import { AppHeader } from './app-header'
import { AppSidebar } from './app-sidebar'

const loadingFallback = <div className="grid min-h-[60vh] place-items-center"><Spinner className="size-8" /></div>

export function AppLayout() {
  const user = useAuthStore((state) => state.user)
  const location = useLocation()
  const [navigationOpen, setNavigationOpen] = useState(false)

  useEffect(() => setNavigationOpen(false), [location.pathname])

  useEffect(() => {
    if (!navigationOpen) return
    const previousOverflow = document.body.style.overflow
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setNavigationOpen(false)
    }
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', closeOnEscape)
    return () => {
      document.body.style.overflow = previousOverflow
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [navigationOpen])

  if (!user) {
    return (
      <div className="flex min-h-screen flex-col">
        <a href="#main-content" className="sr-only z-50 rounded-lg bg-emerald-700 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4">跳到主要内容</a>
        <AppHeader />
        <main id="main-content" className="relative flex-1">
          <Suspense fallback={loadingFallback}><Outlet /></Suspense>
        </main>
        <footer className="border-t border-neutral-200/70 px-4 py-6 text-center text-sm text-neutral-500 dark:border-neutral-800">Seshat · Go 与 React 的轻量全栈起点</footer>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen">
      <a href="#main-content" className="sr-only z-50 rounded-lg bg-emerald-700 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4">跳到主要内容</a>
      <AppSidebar className="sticky top-0 hidden h-screen shrink-0 lg:flex" />

      <div className="flex min-w-0 flex-1 flex-col">
        <AppHeader navigationOpen={navigationOpen} onOpenNavigation={() => setNavigationOpen(true)} />
        <main id="main-content" className="relative flex-1">
          <Suspense fallback={loadingFallback}><Outlet /></Suspense>
        </main>
      </div>

      {navigationOpen && (
        <div className="fixed inset-0 z-50 lg:hidden">
          <button type="button" className="absolute inset-0 bg-neutral-950/45 backdrop-blur-[2px]" aria-label="关闭导航菜单" onClick={() => setNavigationOpen(false)} />
          <AppSidebar
            className="relative flex h-full max-w-[88vw] shadow-2xl"
            mobile
            onClose={() => setNavigationOpen(false)}
            onNavigate={() => setNavigationOpen(false)}
          />
        </div>
      )}
    </div>
  )
}
