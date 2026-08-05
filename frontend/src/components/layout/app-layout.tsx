import { Outlet } from 'react-router-dom'
import { Spinner } from '@appica/ui-react/spinner'
import { Suspense } from 'react'
import { AppHeader } from './app-header'

export function AppLayout() {
  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader />
      <main className="relative flex-1">
        <Suspense fallback={<div className="grid min-h-[60vh] place-items-center"><Spinner className="size-8" /></div>}><Outlet /></Suspense>
      </main>
      <footer className="border-t border-neutral-200/70 px-4 py-6 text-center text-sm text-neutral-500 dark:border-neutral-800">Seshat · Go 与 React 的轻量全栈起点</footer>
    </div>
  )
}
