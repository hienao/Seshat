const ACCESS_TOKEN_STORAGE_KEY = 'seshat_access_token'
let cachedAccessToken: string | null = null

export function readAccessToken(): string | null {
  try {
    cachedAccessToken = localStorage.getItem(ACCESS_TOKEN_STORAGE_KEY)
  } catch {
    // Fall back to memory when browser storage is unavailable.
  }
  return cachedAccessToken
}

export function writeAccessToken(token: string): void {
  cachedAccessToken = token
  try {
    localStorage.setItem(ACCESS_TOKEN_STORAGE_KEY, token)
  } catch {
    // The in-memory auth state still works when storage is unavailable.
  }
}

export function removeAccessToken(): void {
  cachedAccessToken = null
  try {
    localStorage.removeItem(ACCESS_TOKEN_STORAGE_KEY)
  } catch {
    // Ignore storage failures while clearing the local session.
  }
}
