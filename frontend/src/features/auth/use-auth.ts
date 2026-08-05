import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '@/api/services'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.user)
  const setUser = useAuthStore((state) => state.setUser)
  const clearUser = useAuthStore((state) => state.clearUser)

  const login = useMutation({
    mutationFn: async (values: { username: string; password: string }) => {
      await api.login(values.username, values.password)
      return api.profile()
    },
    onSuccess: (profile) => {
      setUser(profile)
      queryClient.setQueryData(['profile'], profile)
    },
  })

  const register = useMutation({
    mutationFn: async (values: { username: string; password: string }) => {
      await api.register(values.username, values.password)
      await api.login(values.username, values.password)
      return api.profile()
    },
    onSuccess: (profile) => {
      setUser(profile)
      queryClient.setQueryData(['profile'], profile)
    },
  })

  const logout = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      clearUser()
      queryClient.clear()
      navigate('/login', { replace: true })
    },
  })

  return { user, isAdmin: Boolean(user?.is_admin), login, register, logout }
}
