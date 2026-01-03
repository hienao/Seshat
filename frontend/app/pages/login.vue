<script setup lang="ts">
const { login } = useAuth()
const router = useRouter()

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
    router.push('/')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <UCard class="w-full max-w-md">
      <template #header>
        <div class="text-center">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">登录</h1>
          <p class="mt-2 text-gray-600 dark:text-gray-400">登录您的账户</p>
        </div>
      </template>

      <form @submit.prevent="handleSubmit" class="space-y-6">
        <UAlert v-if="error" color="error" :title="error" />
        
        <UFormField label="用户名" name="username">
          <UInput
            v-model="form.username"
            placeholder="请输入用户名"
            size="lg"
            required
          />
        </UFormField>

        <UFormField label="密码" name="password">
          <UInput
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            size="lg"
            required
          />
        </UFormField>

        <UButton
          type="submit"
          color="primary"
          size="lg"
          block
          :loading="loading"
        >
          登录
        </UButton>
      </form>

      <template #footer>
        <p class="text-center text-gray-600 dark:text-gray-400">
          还没有账户？
          <NuxtLink to="/register" class="text-primary hover:underline">
            立即注册
          </NuxtLink>
        </p>
      </template>
    </UCard>
  </div>
</template>
