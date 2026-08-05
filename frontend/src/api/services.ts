import { apiRequest } from './client'
import type { RegistrationStatus, SystemSettings, TokenResponse, User } from './types'

export const api = {
  login: (username: string, password: string) =>
    apiRequest<TokenResponse>('/api/auth/login', { method: 'POST', body: { username, password } }),
  register: (username: string, password: string) =>
    apiRequest<User>('/api/auth/register', { method: 'POST', body: { username, password } }),
  logout: () => apiRequest<void>('/api/auth/logout', { method: 'POST' }),
  profile: () => apiRequest<User>('/api/user/profile'),
  changePassword: (oldPassword: string, newPassword: string) =>
    apiRequest<void>('/api/user/password', {
      method: 'PUT',
      body: { old_password: oldPassword, new_password: newPassword },
    }),
  registrationStatus: () => apiRequest<RegistrationStatus>('/api/settings/registration-status'),
  systemSettings: () => apiRequest<SystemSettings>('/api/settings/system'),
  updateSystemSettings: (settings: SystemSettings) =>
    apiRequest<void>('/api/settings/system', { method: 'PUT', body: settings }),
  users: () => apiRequest<User[]>('/api/admin/users'),
  setUserRole: (userId: number, isAdmin: boolean) =>
    apiRequest<void>(`/api/admin/users/${userId}/role`, {
      method: 'PUT',
      body: { is_admin: isAdmin },
    }),
}
