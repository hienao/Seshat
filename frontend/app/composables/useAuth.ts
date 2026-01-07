import type { UseFetchOptions } from 'nuxt/app'

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

export interface ApiResponse<T = any> {
    code: number
    message: string
    data?: T
}

const TOKEN_KEY = 'auth_token'
const USER_KEY = 'auth_user'

export const useAuth = () => {
    const config = useRuntimeConfig()

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
    const isAdmin = computed(() => user.value?.is_admin ?? false)

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

        // 使用 $fetch 避免缓存问题，确保获取当前登录用户的信息
        const profileData = await $fetch<ApiResponse<User>>(`${config.public.apiBase}/user/profile`, {
            headers: {
                'Authorization': `Bearer ${tokenValue}`
            }
        })

        if (!profileData || profileData.code !== 0) {
            throw new Error('获取用户信息失败')
        }

        setAuth(tokenValue, profileData.data!)
        return profileData.data!
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
        navigateTo('/login')
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
        // 使用 $fetch 避免缓存问题
        const data = await $fetch<ApiResponse<User>>(`${config.public.apiBase}/user/profile`, {
            headers: {
                'Authorization': `Bearer ${token.value}`
            }
        })

        if (!data || data.code !== 0) {
            throw new Error(data?.message || '获取用户信息失败')
        }

        user.value = data.data!
        if (import.meta.client) {
            localStorage.setItem(USER_KEY, JSON.stringify(data.data!))
        }
        return data.data!
    }

    // 检查是否允许注册
    const checkRegistrationAllowed = async () => {
        const data = await $fetch<ApiResponse<{ allowed: boolean }>>(`${config.public.apiBase}/settings/registration-status`)
        return data?.data?.allowed ?? false
    }

    // 获取系统设置（管理员）
    const getSystemSettings = async () => {
        const data = await $fetch<ApiResponse<SystemSettings>>(`${config.public.apiBase}/settings/system`, {
            headers: {
                'Authorization': `Bearer ${token.value}`
            }
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '获取系统设置失败')
        }
        return data.data!
    }

    // 更新系统设置（管理员）
    const updateSystemSettings = async (settings: Partial<SystemSettings>) => {
        const data = await $fetch<ApiResponse<void>>(`${config.public.apiBase}/settings/system`, {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${token.value}`
            },
            body: settings
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '更新系统设置失败')
        }
    }

    // 获取用户列表（管理员）
    const listUsers = async () => {
        const data = await $fetch<ApiResponse<User[]>>(`${config.public.apiBase}/admin/users`, {
            headers: {
                'Authorization': `Bearer ${token.value}`
            }
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '获取用户列表失败')
        }
        return data.data!
    }

    // 设置用户角色（管理员）
    const setUserRole = async (userId: number, isAdmin: boolean) => {
        const data = await $fetch<ApiResponse<void>>(`${config.public.apiBase}/admin/users/${userId}/role`, {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${token.value}`
            },
            body: { is_admin: isAdmin }
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '设置用户角色失败')
        }
    }

    return {
        token,
        user,
        isAuthenticated,
        isAdmin,
        login,
        register,
        logout,
        changePassword,
        fetchProfile,
        apiFetch,
        checkRegistrationAllowed,
        getSystemSettings,
        updateSystemSettings,
        listUsers,
        setUserRole
    }
}
