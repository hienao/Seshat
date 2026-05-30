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

const USER_KEY = 'auth_user'

export const useAuth = () => {
    const config = useRuntimeConfig()

    const user = useState<User | null>('auth_user', () => {
        if (import.meta.client) {
            const userStr = localStorage.getItem(USER_KEY)
            return userStr ? JSON.parse(userStr) : null
        }
        return null
    })

    const isAuthenticated = computed(() => !!user.value)
    const isAdmin = computed(() => user.value?.is_admin ?? false)

    const setAuth = (userValue: User) => {
        user.value = userValue
        if (import.meta.client) {
            localStorage.setItem(USER_KEY, JSON.stringify(userValue))
        }
    }

    const clearAuth = () => {
        user.value = null
        if (import.meta.client) {
            localStorage.removeItem(USER_KEY)
        }
    }

    const apiFetch = async <T>(url: string, options: UseFetchOptions<ApiResponse<T>> = {}) => {
        return await useFetch<ApiResponse<T>>(url, {
            baseURL: config.public.apiBase,
            credentials: 'include',
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

        const profileData = await $fetch<ApiResponse<User>>(`${config.public.apiBase}/user/profile`, {
            credentials: 'include'
        })

        if (!profileData || profileData.code !== 0) {
            throw new Error('获取用户信息失败')
        }

        setAuth(profileData.data!)
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
        } catch (_e) {
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
        const data = await $fetch<ApiResponse<User>>(`${config.public.apiBase}/user/profile`, {
            credentials: 'include'
        })

        if (!data || data.code !== 0) {
            throw new Error(data?.message || '获取用户信息失败')
        }

        setAuth(data.data!)
        return data.data!
    }

    const checkRegistrationAllowed = async () => {
        const data = await $fetch<ApiResponse<{ allowed: boolean }>>(`${config.public.apiBase}/settings/registration-status`, {
            credentials: 'include'
        })
        return data?.data?.allowed ?? false
    }

    const getSystemSettings = async () => {
        const data = await $fetch<ApiResponse<SystemSettings>>(`${config.public.apiBase}/settings/system`, {
            credentials: 'include'
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '获取系统设置失败')
        }
        return data.data!
    }

    const updateSystemSettings = async (settings: Partial<SystemSettings>) => {
        const data = await $fetch<ApiResponse<void>>(`${config.public.apiBase}/settings/system`, {
            method: 'PUT',
            credentials: 'include',
            body: settings
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '更新系统设置失败')
        }
    }

    const listUsers = async () => {
        const data = await $fetch<ApiResponse<User[]>>(`${config.public.apiBase}/admin/users`, {
            credentials: 'include'
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '获取用户列表失败')
        }
        return data.data!
    }

    const setUserRole = async (userId: number, isAdmin: boolean) => {
        const data = await $fetch<ApiResponse<void>>(`${config.public.apiBase}/admin/users/${userId}/role`, {
            method: 'PUT',
            credentials: 'include',
            body: { is_admin: isAdmin }
        })
        if (!data || data.code !== 0) {
            throw new Error(data?.message || '设置用户角色失败')
        }
    }

    return {
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
