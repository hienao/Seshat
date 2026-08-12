import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '@/api/services'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.user)
  const setUser = useAuthStore((state) => state.setUser)
  const setAccessToken = useAuthStore((state) => state.setAccessToken)
  const clearSession = useAuthStore((state) => state.clearSession)

  const login = useMutation({
    mutationFn: async (values: { username: string; password: string }) => {
      const session = await api.login(values.username, values.password)
      setAccessToken(session.token)
      try {
        return await api.profile()
      } catch (error) {
        clearSession()
        throw error
      }
    },
    onSuccess: (profile) => {
      setUser(profile)
      queryClient.setQueryData(['profile'], profile)
    },
  })

  const register = useMutation({
    mutationFn: async (values: { username: string; password: string }) => {
      await api.register(values.username, values.password)
      const session = await api.login(values.username, values.password)
      setAccessToken(session.token)
      try {
        return await api.profile()
      } catch (error) {
        clearSession()
        throw error
      }
    },
    onSuccess: (profile) => {
      setUser(profile)
      queryClient.setQueryData(['profile'], profile)
    },
  })

  const logout = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      clearSession()
      queryClient.clear()
      navigate('/login', { replace: true })
    },
  })

  return { user, isAdmin: Boolean(user?.is_admin), login, register, logout }
}
