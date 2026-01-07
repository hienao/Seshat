<script setup lang="ts">
const { login } = useAuth()

const form = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const error = ref('')

const handleSubmit = async () => {
  error.value = ''
  loading.value = true
  
  try {
    await login(form.username, form.password)
    navigateTo('/')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex-1 flex items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100 dark:from-gray-900 dark:to-gray-800 p-4">
    <UCard class="w-full max-w-md shadow-xl">
      <template #header>
        <div class="text-center py-2">
          <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
            <UIcon name="i-heroicons-user-circle" class="w-10 h-10 text-primary-500" />
          </div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">欢迎回来</h1>
          <p class="mt-2 text-gray-600 dark:text-gray-400">登录您的账户以继续</p>
        </div>
      </template>

      <form @submit.prevent="handleSubmit" class="space-y-5">
        <UAlert v-if="error" color="error" :title="error" icon="i-heroicons-exclamation-circle" />
        
        <UFormField label="用户名" name="username" size="lg">
          <UInput
            v-model="form.username"
            placeholder="请输入用户名"
            icon="i-heroicons-user"
            size="lg"
            class="w-full"
            required
          />
        </UFormField>

        <UFormField label="密码" name="password" size="lg">
          <UInput
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            icon="i-heroicons-lock-closed"
            size="lg"
            class="w-full"
            required
          />
        </UFormField>

        <UButton
          type="submit"
          color="primary"
          size="xl"
          block
          :loading="loading"
          class="mt-6"
        >
          <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-5 h-5 mr-2" />
          登录
        </UButton>
      </form>

      <template #footer>
        <div class="text-center py-2">
          <p class="text-gray-600 dark:text-gray-400">
            还没有账户？
            <NuxtLink to="/register" class="text-primary-500 hover:text-primary-600 font-medium hover:underline transition-colors">
              立即注册
            </NuxtLink>
          </p>
        </div>
      </template>
    </UCard>
  </div>
</template>
