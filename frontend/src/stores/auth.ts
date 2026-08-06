import { create } from 'zustand'
import type { User } from '@/api/types'
import {
  readAccessToken,
  removeAccessToken,
  writeAccessToken,
} from '@/lib/auth-token'

interface AuthState {
  user: User | null
  accessToken: string | null
  setUser: (user: User) => void
  setAccessToken: (token: string) => void
  clearSession: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: readAccessToken(),
  setUser: (user) => set({ user }),
  setAccessToken: (token) => {
    writeAccessToken(token)
    set({ accessToken: token })
  },
  clearSession: () => {
    removeAccessToken()
    set({ user: null, accessToken: null })
  },
}))
