// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  modules: ['@nuxt/ui'],

  css: ['~/assets/css/main.css'],

  // 启用预渲染（SSG 使用 `npm run generate`，SSR 使用 `npm run build`）
  ssr: true,

  // 运行时配置（自动从环境变量 NUXT_PUBLIC_API_BASE 读取）
  runtimeConfig: {
    public: {
      apiBase: '/api'
    }
  },

  // 路由配置
  routeRules: {
    '/api/**': { proxy: 'http://localhost:8080/api/**' }
  }
})
