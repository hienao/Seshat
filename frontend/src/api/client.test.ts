import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiRequest } from './client'
import { writeAccessToken } from '@/lib/auth-token'

describe('apiRequest', () => {
  afterEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it('adds the Bearer token, omits credentials, and unwraps successful responses', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: 0, message: 'ok', data: { id: 1 } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    writeAccessToken('test-token')

    await expect(apiRequest<{ id: number }>('/api/user/profile')).resolves.toEqual({ id: 1 })
    expect(fetchMock).toHaveBeenCalledWith('/api/user/profile', expect.objectContaining({ credentials: 'omit' }))
    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(new Headers(request.headers).get('Authorization')).toBe('Bearer test-token')
  })

  it('uses the backend error message for rejected responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: -1, message: '认证令牌无效' }), { status: 403 })))

    await expect(apiRequest('/api/auth/login')).rejects.toEqual(expect.objectContaining<ApiError>({
      name: 'ApiError', message: '认证令牌无效', status: 403, code: -1,
    }))
  })

  it('serializes JSON request bodies', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: 0, message: 'ok' }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await apiRequest('/api/auth/login', { method: 'POST', body: { username: 'admin' } })

    expect(fetchMock).toHaveBeenCalledWith('/api/auth/login', expect.objectContaining({ body: '{"username":"admin"}' }))
  })
})
