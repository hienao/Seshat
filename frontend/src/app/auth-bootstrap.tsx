import { Spinner } from '@appica/ui-react/spinner'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { api } from '@/api/services'
import { ApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { AdminSetupDialog } from '@/components/auth/admin-setup-dialog'

export function AuthBootstrap({ children }: { children: React.ReactNode }) {
  const publicEventView = window.location.pathname.startsWith('/public/events/')
  const accessToken = useAuthStore((state) => state.accessToken)
  const setUser = useAuthStore((state) => state.setUser)
  const clearSession = useAuthStore((state) => state.clearSession)
  const profile = useQuery({
    queryKey: ['profile'],
    queryFn: api.profile,
    enabled: Boolean(accessToken) && !publicEventView,
  })

  useEffect(() => {
    if (profile.data) setUser(profile.data)
    if (profile.error instanceof ApiError && profile.error.status === 401) clearSession()
  }, [profile.data, profile.error, setUser, clearSession])

  if (!publicEventView && accessToken && profile.isPending) {
    return (
      <div className="grid min-h-screen place-items-center" aria-label="正在加载账户信息">
        <Spinner className="size-8" />
      </div>
    )
  }

  return <>{children}{!publicEventView && <AdminSetupDialog />}</>
}
