<script setup lang="ts">
const { user, changePassword, logout } = useAuth()
const router = useRouter()

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
            <span class="text-gray-700 dark:text-gray-300">
              {{ user?.username }}
            </span>
            <UButton color="error" variant="soft" @click="logout">
              退出
            </UButton>
          </div>
        </div>
      </div>
    </header>

    <main class="max-w-3xl mx-auto py-10 px-4 sm:px-6 lg:px-8">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-8">个人中心</h1>
      
      <div class="space-y-8">
        <!-- 用户信息 -->
        <UCard>
          <template #header>
            <h2 class="text-lg font-semibold">账户信息</h2>
          </template>
          
          <div class="space-y-4">
            <div class="flex justify-between">
              <span class="text-gray-600 dark:text-gray-400">用户名</span>
              <span class="font-medium">{{ user?.username }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-600 dark:text-gray-400">用户 ID</span>
              <span class="font-medium">{{ user?.id }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-600 dark:text-gray-400">注册时间</span>
              <span class="font-medium">{{ user?.created_at }}</span>
            </div>
          </div>
        </UCard>

        <!-- 修改密码 -->
        <UCard>
          <template #header>
            <h2 class="text-lg font-semibold">修改密码</h2>
          </template>
          
          <form @submit.prevent="handleChangePassword" class="space-y-6">
            <UAlert v-if="error" color="error" :title="error" />
            <UAlert v-if="success" color="success" :title="success" />
            
            <UFormField label="当前密码" name="oldPassword">
              <UInput
                v-model="passwordForm.oldPassword"
                type="password"
                placeholder="请输入当前密码"
                required
              />
            </UFormField>

            <UFormField label="新密码" name="newPassword">
              <UInput
                v-model="passwordForm.newPassword"
                type="password"
                placeholder="请输入新密码（至少 6 位）"
                required
                minlength="6"
              />
            </UFormField>

            <UFormField label="确认新密码" name="confirmPassword">
              <UInput
                v-model="passwordForm.confirmPassword"
                type="password"
                placeholder="请再次输入新密码"
                required
              />
            </UFormField>

            <UButton
              type="submit"
              color="primary"
              :loading="loading"
            >
              修改密码
            </UButton>
          </form>
        </UCard>
      </div>
    </main>
  </div>
</template>
