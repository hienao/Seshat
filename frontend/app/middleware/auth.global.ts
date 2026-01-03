export default defineNuxtRouteMiddleware((to) => {
    // SSG 预渲染时跳过
    if (import.meta.server) {
        return
    }

    const { isAuthenticated } = useAuth()

    // 需要认证的页面
    const protectedRoutes = ['/profile']

    if (protectedRoutes.includes(to.path) && !isAuthenticated.value) {
        return navigateTo('/login')
    }

    // 已登录用户不能访问登录/注册页
    const guestRoutes = ['/login', '/register']

    if (guestRoutes.includes(to.path) && isAuthenticated.value) {
        return navigateTo('/')
    }
})
