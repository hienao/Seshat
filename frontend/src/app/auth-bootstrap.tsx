import { Spinner } from '@appica/ui-react/spinner'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { api } from '@/api/services'
import { ApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

export function AuthBootstrap({ children }: { children: React.ReactNode }) {
  const setUser = useAuthStore((state) => state.setUser)
  const clearUser = useAuthStore((state) => state.clearUser)
  const profile = useQuery({ queryKey: ['profile'], queryFn: api.profile })

  useEffect(() => {
    if (profile.data) setUser(profile.data)
    if (profile.error instanceof ApiError && profile.error.status === 401) clearUser()
  }, [profile.data, profile.error, setUser, clearUser])

  if (profile.isPending) {
    return (
      <div className="grid min-h-screen place-items-center" aria-label="正在加载账户信息">
        <Spinner className="size-8" />
      </div>
    )
  }

  return children
}
