import { ThemeProvider } from '@appica/ui-react/providers/theme-provider'
import { ToastProvider, Toaster } from '@appica/ui-react/toast'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import { AuthBootstrap } from './auth-bootstrap'

export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider defaultTheme="system" enableSystem storageKey="basegoapp-theme">
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <AuthBootstrap>{children}</AuthBootstrap>
          <Toaster position="top-right" />
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>
  )
}
