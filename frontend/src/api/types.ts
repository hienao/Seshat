export interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

export interface User {
  id: number
  username: string
  is_admin: boolean
  requires_admin_setup: boolean
  created_at: string
}

export interface TokenResponse {
  token: string
  expires_at: number
}

export interface SystemSettings {
  allow_register: boolean
  api_log_retention_days: number
  allow_private_notification_targets: boolean
  http_proxy_configured: boolean
  http_proxy_display?: string
}

export type UpdateSystemSettingsInput = Partial<Pick<SystemSettings, 'allow_register' | 'api_log_retention_days' | 'allow_private_notification_targets'>> & {
  http_proxy_url?: string
  clear_http_proxy?: boolean
}

export interface RegistrationStatus {
  allowed: boolean
}

export interface AppEventType {
  code: string
  name: string
  render_mode: 'raw' | 'custom'
}

export interface AppDefinition {
  code: string
  name: string
  description: string
  auth_mode: 'secret_header' | 'github_signature' | 'endpoint_url'
  default_event_type: string
  event_types: AppEventType[]
}

export interface Integration {
  id: number
  owner_id: number
  app_code: string
  name: string
  endpoint_key: string
  webhook_path: string
  config: Record<string, unknown>
  enabled: boolean
  notification_channel_id?: number
  created_at: string
  updated_at: string
}

export interface CreatedIntegration {
  id: number
  owner_id: number
  app_code: string
  name: string
  endpoint_key: string
  webhook_path: string
  enabled: boolean
  secret: string
}

export interface IntegrationSecret {
  secret: string
}

export interface EventPresentation {
  schema_version: number
  title: string
  summary?: string
  severity: string
  tags?: string[]
  facts?: Array<Record<string, string>>
  links?: Array<Record<string, string>>
  data?: Record<string, unknown>
}

export interface WebhookEvent {
  id: number
  integration_id: number
  app_code: string
  source_event_type: string
  display_event_type: string
  external_event_id: string
  status: string
  is_fallback: boolean
  title: string
  summary: string
  severity: string
  occurred_at?: string
  presentation_version: number
  presentation: EventPresentation
  raw_body?: string
  content_type: string
  received_at: string
}

export interface EventListResponse {
  items: WebhookEvent[]
  total: number
}

export interface ApiRequestLog {
  id: number
  request_id: string
  occurred_at: string
  method: string
  route: string
  status_code: number
  latency_ms: number
  request_bytes: number
  response_bytes: number
  request_headers?: string
  request_body?: string
  client_ip: string
  user_id?: number
  username?: string
  app_code?: string
  integration_id?: number
  event_id?: number
  error_code?: string
  error_message?: string
  user_agent?: string
}

export interface ApiLogListResponse {
  items: ApiRequestLog[]
  total: number
  next_cursor?: number
  has_more: boolean
}

export interface ApiLogSummary {
  total: number
  success_count: number
  client_errors: number
  server_errors: number
  average_ms: number
  dropped: number
}

export type ApplicationLogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

export interface ApplicationLog {
  id: number
  occurred_at: string
  level: ApplicationLogLevel
  source?: string
  message: string
  fields?: string
  request_id?: string
  user_id?: number
  app_code?: string
  integration_id?: number
  event_id?: number
  created_at: string
}

export interface ApplicationLogListResponse {
  items: ApplicationLog[]
  total: number
  next_cursor?: number
  has_more: boolean
}

export interface ApplicationLogSummary {
  total: number
  debug_count: number
  info_count: number
  warn_count: number
  error_count: number
  dropped: number
}

export type NotificationChannelType = 'webhook' | 'telegram' | 'apprise' | 'email' | 'serverchan' | 'bark' | 'dingtalk' | 'feishu' | 'whatsapp' | 'wxpusher'

export interface NotificationChannel {
  id: number
  name: string
  type: NotificationChannelType
  enabled: boolean
  config: Record<string, unknown>
  has_credentials: boolean
  binding_count: number
  last_test_status?: string
  last_test_at?: string
  last_test_error?: string
  created_at: string
  updated_at: string
}

export interface NotificationChannelInput {
  name: string
  type: NotificationChannelType
  enabled: boolean
  config: Record<string, unknown>
  credentials?: Record<string, unknown>
}

export interface NotificationEventTypeSetting {
  code: string
  name: string
  is_default: boolean
  notify: boolean
}

export interface IntegrationNotificationSettings {
  integration_id: number
  app_code: string
  channel_id?: number
  channel_enabled: boolean
  event_types: NotificationEventTypeSetting[]
}

export interface NotificationDelivery {
  id: number
  event_id: number
  integration_id: number
  integration_name: string
  channel_id: number
  channel_name: string
  channel_type: NotificationChannelType
  event_type: string
  status: 'pending' | 'sending' | 'retrying' | 'succeeded' | 'failed'
  attempt_count: number
  next_attempt_at?: string
  last_status_code?: number
  last_error?: string
  sent_at?: string
  created_at: string
  updated_at: string
}

export interface EventNotificationStatus {
  state: 'not_configured' | 'channel_disabled' | 'not_matched' | 'not_created' | NotificationDelivery['status']
  reason?: string
  channel_id?: number
  delivery?: NotificationDelivery
}
