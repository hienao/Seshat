<script setup lang="ts">
const { isAuthenticated, user, logout } = useAuth()
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 导航栏 -->
    <header class="bg-white dark:bg-gray-800 shadow">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex items-center">
            <NuxtLink to="/" class="text-xl font-bold text-primary">
              BaseGoApp
            </NuxtLink>
          </div>
          
          <div class="flex items-center gap-4">
            <UColorModeButton />
            
            <template v-if="isAuthenticated">
              <span class="text-gray-700 dark:text-gray-300">
                {{ user?.username }}
              </span>
              <NuxtLink to="/profile">
                <UButton color="neutral" variant="ghost">
                  个人中心
                </UButton>
              </NuxtLink>
              <UButton color="error" variant="soft" @click="logout">
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
    <main class="max-w-7xl mx-auto py-10 px-4 sm:px-6 lg:px-8">
      <div class="text-center">
        <h1 class="text-4xl font-bold text-gray-900 dark:text-white mb-4">
          欢迎使用 BaseGoApp
        </h1>
        <p class="text-lg text-gray-600 dark:text-gray-400 mb-8">
          这是一个基于 Nuxt 3 + Go Gin 的全栈模板工程
        </p>
        
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mt-10">
          <UCard>
            <template #header>
              <h3 class="text-lg font-semibold">前端技术栈</h3>
            </template>
            <ul class="text-left space-y-2 text-gray-600 dark:text-gray-400">
              <li>• Nuxt 3 (SSG)</li>
              <li>• @nuxt/ui</li>
              <li>• Vue 3</li>
              <li>• TypeScript</li>
            </ul>
          </UCard>
          
          <UCard>
            <template #header>
              <h3 class="text-lg font-semibold">后端技术栈</h3>
            </template>
            <ul class="text-left space-y-2 text-gray-600 dark:text-gray-400">
              <li>• Go + Gin</li>
              <li>• GORM</li>
              <li>• PostgreSQL</li>
              <li>• Swagger 文档</li>
            </ul>
          </UCard>
          
          <UCard>
            <template #header>
              <h3 class="text-lg font-semibold">功能特性</h3>
            </template>
            <ul class="text-left space-y-2 text-gray-600 dark:text-gray-400">
              <li>• JWT 认证</li>
              <li>• 用户注册/登录</li>
              <li>• 修改密码</li>
              <li>• Docker 部署</li>
            </ul>
          </UCard>
        </div>
      </div>
    </main>
  </div>
</template>
