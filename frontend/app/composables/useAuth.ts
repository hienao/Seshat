import type { UseFetchOptions } from 'nuxt/app'

export interface User {
    id: number
    username: string
    created_at: string
}

export interface TokenResponse {
    token: string
    expires_at: number
}

export interface ApiResponse<T = any> {
    code: number
    message: string
    data?: T
}

const TOKEN_KEY = 'auth_token'
const USER_KEY = 'auth_user'

export const useAuth = () => {
    const config = useRuntimeConfig()
    const router = useRouter()

    const token = useState<string | null>('auth_token', () => {
        if (import.meta.client) {
            return localStorage.getItem(TOKEN_KEY)
        }
        return null
    })

    const user = useState<User | null>('auth_user', () => {
        if (import.meta.client) {
            const userStr = localStorage.getItem(USER_KEY)
            return userStr ? JSON.parse(userStr) : null
        }
        return null
    })

    const isAuthenticated = computed(() => !!token.value)

    const setAuth = (tokenValue: string, userValue: User) => {
        token.value = tokenValue
        user.value = userValue
        if (import.meta.client) {
            localStorage.setItem(TOKEN_KEY, tokenValue)
            localStorage.setItem(USER_KEY, JSON.stringify(userValue))
        }
    }

    const clearAuth = () => {
        token.value = null
        user.value = null
        if (import.meta.client) {
            localStorage.removeItem(TOKEN_KEY)
            localStorage.removeItem(USER_KEY)
        }
    }

    const apiFetch = async <T>(url: string, options: UseFetchOptions<ApiResponse<T>> = {}) => {
        const headers: Record<string, string> = {}
        if (token.value) {
            headers['Authorization'] = `Bearer ${token.value}`
        }

        return await useFetch<ApiResponse<T>>(url, {
            baseURL: config.public.apiBase,
            headers,
            ...options
        })
    }

    const login = async (username: string, password: string) => {
        const { data, error } = await apiFetch<TokenResponse>('/auth/login', {
            method: 'POST',
            body: { username, password }
        })

        if (error.value || !data.value || data.value.code !== 0) {
            throw new Error(data.value?.message || '登录失败')
        }

        // 获取用户信息
        const tokenValue = data.value.data!.token
        token.value = tokenValue
        if (import.meta.client) {
            localStorage.setItem(TOKEN_KEY, tokenValue)
        }

        const { data: profileData, error: profileError } = await apiFetch<User>('/user/profile')

        if (profileError.value || !profileData.value || profileData.value.code !== 0) {
            throw new Error('获取用户信息失败')
        }

        setAuth(tokenValue, profileData.value.data!)
        return profileData.value.data!
    }

    const register = async (username: string, password: string) => {
        const { data, error } = await apiFetch<User>('/auth/register', {
            method: 'POST',
            body: { username, password }
        })

        if (error.value || !data.value || data.value.code !== 0) {
            throw new Error(data.value?.message || '注册失败')
        }

        return data.value.data!
    }

    const logout = async () => {
        try {
            await apiFetch('/auth/logout', { method: 'POST' })
        } catch (e) {
            // 忽略错误
        }
        clearAuth()
        router.push('/login')
    }

    const changePassword = async (oldPassword: string, newPassword: string) => {
        const { data, error } = await apiFetch('/user/password', {
            method: 'PUT',
            body: { old_password: oldPassword, new_password: newPassword }
        })

        if (error.value || !data.value || data.value.code !== 0) {
            throw new Error(data.value?.message || '修改密码失败')
        }
    }

    const fetchProfile = async () => {
        const { data, error } = await apiFetch<User>('/user/profile')

        if (error.value || !data.value || data.value.code !== 0) {
            throw new Error(data.value?.message || '获取用户信息失败')
        }

        user.value = data.value.data!
        if (import.meta.client) {
            localStorage.setItem(USER_KEY, JSON.stringify(data.value.data!))
        }
        return data.value.data!
    }

    return {
        token,
        user,
        isAuthenticated,
        login,
        register,
        logout,
        changePassword,
        fetchProfile,
        apiFetch
    }
}
