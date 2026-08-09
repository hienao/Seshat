import { apiRequest } from './client'
import type { ApiLogListResponse, ApiLogSummary, ApiRequestLog, AppDefinition, ApplicationLog, ApplicationLogListResponse, ApplicationLogSummary, CreatedIntegration, DisplayWebhookEvent, EventListResponse, EventNotificationStatus, Integration, IntegrationNotificationSettings, IntegrationSecret, NotificationChannel, NotificationChannelInput, NotificationDelivery, RegistrationStatus, SystemSettings, TokenResponse, UpdateSystemSettingsInput, User, WebhookEvent } from './types'
import { readAccessToken } from '@/lib/auth-token'

export const api = {
  login: (username: string, password: string) =>
    apiRequest<TokenResponse>('/api/auth/login', { method: 'POST', body: { username, password } }),
  setupAdmin: (username: string, password: string) =>
    apiRequest<TokenResponse>('/api/auth/setup-admin', { method: 'POST', body: { username, password } }),
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
  updateSystemSettings: (settings: UpdateSystemSettingsInput) =>
    apiRequest<void>('/api/settings/system', { method: 'PUT', body: settings }),
  users: () => apiRequest<User[]>('/api/admin/users'),
  setUserRole: (userId: number, isAdmin: boolean) =>
    apiRequest<void>(`/api/admin/users/${userId}/role`, {
      method: 'PUT',
      body: { is_admin: isAdmin },
    }),
  webhookApps: () => apiRequest<AppDefinition[]>('/api/webhooks/apps'),
  integrations: () => apiRequest<Integration[]>('/api/webhooks/integrations'),
  createIntegration: (appCode: string, name: string) =>
    apiRequest<CreatedIntegration>('/api/webhooks/integrations', { method: 'POST', body: { app_code: appCode, name } }),
  integrationSecret: (id: number) => apiRequest<IntegrationSecret>(`/api/webhooks/integrations/${id}/secret`),
  rotateIntegrationSecret: (id: number) =>
    apiRequest<CreatedIntegration>(`/api/webhooks/integrations/${id}/rotate-secret`, { method: 'POST' }),
  integrationNotificationSettings: (id: number) => apiRequest<IntegrationNotificationSettings>(`/api/webhooks/integrations/${id}/notification-settings`),
  updateIntegrationNotificationSettings: (id: number, channelId: number | null, enabledEventTypes: string[]) => apiRequest<IntegrationNotificationSettings>(`/api/webhooks/integrations/${id}/notification-settings`, { method: 'PUT', body: { channel_id: channelId, enabled_event_types: enabledEventTypes } }),
  notificationChannels: () => apiRequest<NotificationChannel[]>('/api/notification-channels'),
  createNotificationChannel: (body: NotificationChannelInput) => apiRequest<NotificationChannel>('/api/notification-channels', { method: 'POST', body }),
  updateNotificationChannel: (id: number, body: NotificationChannelInput) => apiRequest<NotificationChannel>(`/api/notification-channels/${id}`, { method: 'PUT', body }),
  deleteNotificationChannel: (id: number) => apiRequest<void>(`/api/notification-channels/${id}`, { method: 'DELETE' }),
  testNotificationChannel: (id: number) => apiRequest<NotificationChannel>(`/api/notification-channels/${id}/test`, { method: 'POST' }),
  notificationDeliveries: (status = '') => apiRequest<NotificationDelivery[]>(`/api/notifications/deliveries${status ? `?status=${encodeURIComponent(status)}` : ''}`),
  retryNotificationDelivery: (id: number) => apiRequest<void>(`/api/notifications/deliveries/${id}/retry`, { method: 'POST' }),
  events: (params: { appCode?: string; eventType?: string; offset?: number } = {}) => {
    const query = new URLSearchParams()
    if (params.appCode) query.set('app_code', params.appCode)
    if (params.eventType) query.set('event_type', params.eventType)
    if (params.offset) query.set('offset', String(params.offset))
    return apiRequest<EventListResponse>(`/api/webhooks/events${query.size ? `?${query.toString()}` : ''}`)
  },
  event: (id: number) => apiRequest<WebhookEvent>(`/api/webhooks/events/${id}`),
  publicEvent: (token: string) => apiRequest<DisplayWebhookEvent>(`/api/public/events/${encodeURIComponent(token)}`, { auth: false }),
  eventNotificationStatus: (id: number) => apiRequest<EventNotificationStatus>(`/api/webhooks/events/${id}/notification-status`),
  apiLogs: (params: { startAt?: string; endAt?: string; method?: string; route?: string; statusGroup?: string; requestId?: string; keyword?: string; cursor?: number } = {}) => {
    const query = new URLSearchParams()
    if (params.startAt) query.set('start_at', params.startAt)
    if (params.endAt) query.set('end_at', params.endAt)
    if (params.method) query.set('method', params.method)
    if (params.route) query.set('route', params.route)
    if (params.statusGroup) query.set('status_group', params.statusGroup)
    if (params.requestId) query.set('request_id', params.requestId)
    if (params.keyword) query.set('keyword', params.keyword)
    if (params.cursor) query.set('cursor', String(params.cursor))
    return apiRequest<ApiLogListResponse>(`/api/admin/logs${query.size ? `?${query.toString()}` : ''}`)
  },
  apiLogSummary: (params: { startAt?: string; endAt?: string; method?: string; route?: string; statusGroup?: string; requestId?: string; keyword?: string } = {}) => {
    const query = new URLSearchParams()
    if (params.startAt) query.set('start_at', params.startAt)
    if (params.endAt) query.set('end_at', params.endAt)
    if (params.method) query.set('method', params.method)
    if (params.route) query.set('route', params.route)
    if (params.statusGroup) query.set('status_group', params.statusGroup)
    if (params.requestId) query.set('request_id', params.requestId)
    if (params.keyword) query.set('keyword', params.keyword)
    return apiRequest<ApiLogSummary>(`/api/admin/logs/summary${query.size ? `?${query.toString()}` : ''}`)
  },
  apiLog: (id: number) => apiRequest<ApiRequestLog>(`/api/admin/logs/${id}`),
  clearApiLogs: (body: { start_at?: string; end_at?: string; method?: string; route?: string; status_group?: string; confirmation: string }) =>
    apiRequest<{ deleted_count: number }>('/api/admin/logs/clear', { method: 'POST', body }),
  exportApiLogs: async (params: { format: 'csv' | 'jsonl'; startAt?: string; endAt?: string; method?: string; route?: string; statusGroup?: string; requestId?: string; keyword?: string }) => {
    const query = new URLSearchParams({ format: params.format })
    if (params.startAt) query.set('start_at', params.startAt)
    if (params.endAt) query.set('end_at', params.endAt)
    if (params.method) query.set('method', params.method)
    if (params.route) query.set('route', params.route)
    if (params.statusGroup) query.set('status_group', params.statusGroup)
    if (params.requestId) query.set('request_id', params.requestId)
    if (params.keyword) query.set('keyword', params.keyword)
    const accessToken = readAccessToken()
    const response = await fetch(`/api/admin/logs/export?${query.toString()}`, {
      credentials: 'omit',
      headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : undefined,
    })
    if (!response.ok) throw new Error(`导出失败 (${response.status})`)
    return { blob: await response.blob(), filename: response.headers.get('Content-Disposition')?.match(/filename=([^;]+)/)?.[1] ?? `seshat-api-logs.${params.format}` }
  },
  applicationLogs: (params: { startAt?: string; endAt?: string; level?: string; source?: string; requestId?: string; keyword?: string; cursor?: number } = {}) => {
    const query = new URLSearchParams()
    if (params.startAt) query.set('start_at', params.startAt)
    if (params.endAt) query.set('end_at', params.endAt)
    if (params.level) query.set('level', params.level)
    if (params.source) query.set('source', params.source)
    if (params.requestId) query.set('request_id', params.requestId)
    if (params.keyword) query.set('keyword', params.keyword)
    if (params.cursor) query.set('cursor', String(params.cursor))
    return apiRequest<ApplicationLogListResponse>(`/api/admin/application-logs${query.size ? `?${query.toString()}` : ''}`)
  },
  applicationLogSummary: (params: { startAt?: string; endAt?: string; level?: string; source?: string; requestId?: string; keyword?: string } = {}) => {
    const query = new URLSearchParams()
    if (params.startAt) query.set('start_at', params.startAt)
    if (params.endAt) query.set('end_at', params.endAt)
    if (params.level) query.set('level', params.level)
    if (params.source) query.set('source', params.source)
    if (params.requestId) query.set('request_id', params.requestId)
    if (params.keyword) query.set('keyword', params.keyword)
    return apiRequest<ApplicationLogSummary>(`/api/admin/application-logs/summary${query.size ? `?${query.toString()}` : ''}`)
  },
  applicationLog: (id: number) => apiRequest<ApplicationLog>(`/api/admin/application-logs/${id}`),
  clearApplicationLogs: (body: { start_at?: string; end_at?: string; level?: string; source?: string; request_id?: string; keyword?: string; confirmation: string }) =>
    apiRequest<{ deleted_count: number }>('/api/admin/application-logs/clear', { method: 'POST', body }),
  exportApplicationLogs: async (params: { format: 'csv' | 'jsonl'; startAt?: string; endAt?: string; level?: string; source?: string; requestId?: string; keyword?: string }) => {
    const query = new URLSearchParams({ format: params.format })
    if (params.startAt) query.set('start_at', params.startAt)
    if (params.endAt) query.set('end_at', params.endAt)
    if (params.level) query.set('level', params.level)
    if (params.source) query.set('source', params.source)
    if (params.requestId) query.set('request_id', params.requestId)
    if (params.keyword) query.set('keyword', params.keyword)
    const accessToken = readAccessToken()
    const response = await fetch(`/api/admin/application-logs/export?${query.toString()}`, {
      credentials: 'omit',
      headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : undefined,
    })
    if (!response.ok) throw new Error(`导出失败 (${response.status})`)
    return { blob: await response.blob(), filename: response.headers.get('Content-Disposition')?.match(/filename=([^;]+)/)?.[1] ?? `seshat-application-logs.${params.format}` }
  },
}
