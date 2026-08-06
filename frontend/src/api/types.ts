export interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

export interface User {
  id: number
  username: string
  is_admin: boolean
  created_at: string
}

export interface TokenResponse {
  token: string
  expires_at: number
}

export interface SystemSettings {
  allow_register: boolean
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
