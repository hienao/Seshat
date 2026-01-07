<script setup lang="ts">
const { user, changePassword, logout } = useAuth()

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const loading = ref(false)
const error = ref('')
const success = ref('')

const handleChangePassword = async () => {
  error.value = ''
  success.value = ''
  
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    error.value = '两次输入的新密码不一致'
    return
  }
  
  if (passwordForm.newPassword.length < 6) {
    error.value = '新密码长度至少 6 位'
    return
  }
  
  loading.value = true
  
  try {
    await changePassword(passwordForm.oldPassword, passwordForm.newPassword)
    success.value = '密码修改成功，请重新登录'
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
    
    // 3 秒后退出登录
    setTimeout(() => {
      logout()
    }, 3000)
  } catch (e: any) {
    error.value = e.message || '修改密码失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
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
            <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 dark:bg-gray-700">
              <UIcon name="i-heroicons-user-circle" class="w-5 h-5 text-gray-500 dark:text-gray-400" />
              <span class="text-gray-700 dark:text-gray-300 font-medium">
                {{ user?.username }}
              </span>
            </div>
            <UButton color="error" variant="soft" @click="logout">
              <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-4 h-4 mr-1" />
              退出
            </UButton>
          </div>
        </div>
      </div>
    </header>

    <main class="max-w-3xl mx-auto py-10 px-4 sm:px-6 lg:px-8">
      <div class="flex items-center gap-3 mb-8">
        <div class="w-12 h-12 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
          <UIcon name="i-heroicons-cog-6-tooth" class="w-6 h-6 text-primary-500" />
        </div>
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">个人中心</h1>
          <p class="text-gray-500 dark:text-gray-400">管理您的账户设置</p>
        </div>
      </div>
      
      <div class="space-y-6">
        <!-- 用户信息 -->
        <UCard class="shadow-lg">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-user" class="w-5 h-5 text-primary-500" />
              <h2 class="text-lg font-semibold">账户信息</h2>
            </div>
          </template>
          
          <div class="divide-y divide-gray-200 dark:divide-gray-700">
            <div class="flex justify-between py-4 first:pt-0 last:pb-0">
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-at-symbol" class="w-5 h-5 text-gray-400" />
                <span class="text-gray-600 dark:text-gray-400">用户名</span>
              </div>
              <span class="font-medium text-gray-900 dark:text-white">{{ user?.username }}</span>
            </div>
            <div class="flex justify-between py-4 first:pt-0 last:pb-0">
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-identification" class="w-5 h-5 text-gray-400" />
                <span class="text-gray-600 dark:text-gray-400">用户 ID</span>
              </div>
              <span class="font-medium font-mono text-gray-900 dark:text-white">{{ user?.id }}</span>
            </div>
            <div class="flex justify-between py-4 first:pt-0 last:pb-0">
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-calendar" class="w-5 h-5 text-gray-400" />
                <span class="text-gray-600 dark:text-gray-400">注册时间</span>
              </div>
              <span class="font-medium text-gray-900 dark:text-white">{{ user?.created_at }}</span>
            </div>
          </div>
        </UCard>

        <!-- 修改密码 -->
        <UCard class="shadow-lg">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-key" class="w-5 h-5 text-primary-500" />
              <h2 class="text-lg font-semibold">修改密码</h2>
            </div>
          </template>
          
          <form @submit.prevent="handleChangePassword" class="space-y-5">
            <UAlert 
              v-if="error" 
              color="error" 
              :title="error" 
              icon="i-heroicons-exclamation-circle"
            />
            <UAlert 
              v-if="success" 
              color="success" 
              :title="success" 
              icon="i-heroicons-check-circle"
            />
            
            <UFormField label="当前密码" name="oldPassword" size="lg">
              <UInput
                v-model="passwordForm.oldPassword"
                type="password"
                placeholder="请输入当前密码"
                icon="i-heroicons-lock-closed"
                size="lg"
                class="w-full"
                required
              />
            </UFormField>

            <UFormField label="新密码" name="newPassword" size="lg">
              <UInput
                v-model="passwordForm.newPassword"
                type="password"
                placeholder="请输入新密码（至少 6 位）"
                icon="i-heroicons-lock-closed"
                size="lg"
                class="w-full"
                required
                minlength="6"
              />
            </UFormField>

            <UFormField label="确认新密码" name="confirmPassword" size="lg">
              <UInput
                v-model="passwordForm.confirmPassword"
                type="password"
                placeholder="请再次输入新密码"
                icon="i-heroicons-lock-closed"
                size="lg"
                class="w-full"
                required
              />
            </UFormField>

            <div class="pt-2">
              <UButton
                type="submit"
                color="primary"
                size="lg"
                :loading="loading"
              >
                <UIcon name="i-heroicons-check" class="w-5 h-5 mr-1" />
                修改密码
              </UButton>
            </div>
          </form>
        </UCard>
      </div>
    </main>
  </div>
</template>
