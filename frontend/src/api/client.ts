import type { ApiResponse } from './types'

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code = -1,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

type RequestOptions = Omit<RequestInit, 'body'> & { body?: unknown }

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const hasBody = options.body !== undefined
  if (hasBody && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')

  const response = await fetch(path, {
    ...options,
    headers,
    credentials: 'include',
    body: hasBody ? JSON.stringify(options.body) : undefined,
  })

  let payload: ApiResponse<T> | undefined
  try {
    payload = (await response.json()) as ApiResponse<T>
  } catch {
    throw new ApiError(response.ok ? '服务器返回了无效响应' : `请求失败 (${response.status})`, response.status)
  }

  if (!response.ok || payload.code !== 0) {
    throw new ApiError(payload.message || `请求失败 (${response.status})`, response.status, payload.code)
  }

  return payload.data as T
}
