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
