<script setup lang="ts">
const { isAuthenticated, user, logout, isAdmin } = useAuth()
</script>

<template>
  <div class="flex-1 bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
    <!-- 导航栏 -->
    <header class="bg-white/80 dark:bg-gray-800/80 backdrop-blur-md shadow-sm sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex items-center">
            <NuxtLink to="/" class="text-xl font-bold text-primary-500 hover:text-primary-600 transition-colors">
              BaseGoApp
            </NuxtLink>
          </div>
          
          <div class="flex items-center gap-4">
            <UColorModeButton />
            
            <template v-if="isAuthenticated">
              <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 dark:bg-gray-700">
                <UIcon name="i-heroicons-user-circle" class="w-5 h-5 text-gray-500 dark:text-gray-400" />
                <span class="text-gray-700 dark:text-gray-300 font-medium">
                  {{ user?.username }}
                </span>
              </div>
              <NuxtLink to="/profile">
                <UButton color="neutral" variant="ghost">
                  <UIcon name="i-heroicons-cog-6-tooth" class="w-4 h-4 mr-1" />
                  个人中心
                </UButton>
              </NuxtLink>
              <NuxtLink v-if="isAdmin" to="/admin">
                <UButton color="primary" variant="ghost">
                  <UIcon name="i-heroicons-shield-check" class="w-4 h-4 mr-1" />
                  系统管理
                </UButton>
              </NuxtLink>
              <UButton color="error" variant="soft" @click="logout">
                <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-4 h-4 mr-1" />
                退出
              </UButton>
            </template>
            
            <template v-else>
              <NuxtLink to="/login">
                <UButton color="primary" variant="soft">
                  登录
                </UButton>
              </NuxtLink>
              <NuxtLink to="/register">
                <UButton color="primary">
                  注册
                </UButton>
              </NuxtLink>
            </template>
          </div>
        </div>
      </div>
    </header>

    <!-- 主内容区 -->
    <main class="max-w-7xl mx-auto py-16 px-4 sm:px-6 lg:px-8">
      <!-- Hero 区域 -->
      <div class="text-center mb-16">
        <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 text-sm font-medium mb-6">
          <UIcon name="i-heroicons-sparkles" class="w-4 h-4" />
          全栈模板工程
        </div>
        <h1 class="text-5xl font-bold text-gray-900 dark:text-white mb-6 leading-tight">
          欢迎使用 <span class="text-primary-500">BaseGoApp</span>
        </h1>
        <p class="text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto leading-relaxed">
          这是一个基于 Nuxt 3 + Go Gin 的现代化全栈模板工程，
          <br>助您快速启动新项目
        </p>
        
        <div class="flex justify-center gap-4 mt-8">
          <NuxtLink v-if="!isAuthenticated" to="/register">
            <UButton color="primary" size="xl">
              <UIcon name="i-heroicons-rocket-launch" class="w-5 h-5 mr-2" />
              立即开始
            </UButton>
          </NuxtLink>
          <UButton 
            color="neutral" 
            variant="outline" 
            size="xl"
            as="a"
            href="/swagger/index.html"
            target="_blank"
          >
            <UIcon name="i-heroicons-document-text" class="w-5 h-5 mr-2" />
            API 文档
          </UButton>
        </div>
      </div>
      
      <!-- 特性卡片 -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
        <UCard class="shadow-lg hover:shadow-xl transition-shadow duration-300 group">
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-xl bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                <UIcon name="i-heroicons-code-bracket" class="w-6 h-6 text-blue-500" />
              </div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">前端技术栈</h3>
            </div>
          </template>
          <ul class="space-y-3">
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              Nuxt 3 (SSG/SSR)
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              @nuxt/ui 组件库
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              Vue 3 Composition API
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              TypeScript 类型安全
            </li>
          </ul>
        </UCard>
        
        <UCard class="shadow-lg hover:shadow-xl transition-shadow duration-300 group">
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-xl bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                <UIcon name="i-heroicons-server" class="w-6 h-6 text-emerald-500" />
              </div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">后端技术栈</h3>
            </div>
          </template>
          <ul class="space-y-3">
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              Go + Gin 高性能框架
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              GORM ORM 框架
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              PostgreSQL 数据库
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              Swagger API 文档
            </li>
          </ul>
        </UCard>
        
        <UCard class="shadow-lg hover:shadow-xl transition-shadow duration-300 group">
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-xl bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                <UIcon name="i-heroicons-puzzle-piece" class="w-6 h-6 text-purple-500" />
              </div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">功能特性</h3>
            </div>
          </template>
          <ul class="space-y-3">
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              JWT 安全认证
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              用户注册 / 登录
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              密码修改功能
            </li>
            <li class="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-500 flex-shrink-0" />
              Docker 一键部署
            </li>
          </ul>
        </UCard>
      </div>
    </main>
  </div>
</template>
