<script setup lang="ts">
const { register, login } = useAuth()
const router = useRouter()

const form = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

const loading = ref(false)
const error = ref('')

const handleSubmit = async () => {
  error.value = ''
  
  if (form.password !== form.confirmPassword) {
    error.value = '两次输入的密码不一致'
    return
  }
  
  if (form.password.length < 6) {
    error.value = '密码长度至少 6 位'
    return
  }
  
  loading.value = true
  
  try {
    await register(form.username, form.password)
    // 注册成功后自动登录
    await login(form.username, form.password)
    router.push('/')
  } catch (e: any) {
    error.value = e.message || '注册失败'
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
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">注册</h1>
          <p class="mt-2 text-gray-600 dark:text-gray-400">创建新账户</p>
        </div>
      </template>

      <form @submit.prevent="handleSubmit" class="space-y-6">
        <UAlert v-if="error" color="error" :title="error" />
        
        <UFormField label="用户名" name="username">
          <UInput
            v-model="form.username"
            placeholder="请输入用户名（3-50 个字符）"
            size="lg"
            required
            minlength="3"
            maxlength="50"
          />
        </UFormField>

        <UFormField label="密码" name="password">
          <UInput
            v-model="form.password"
            type="password"
            placeholder="请输入密码（至少 6 位）"
            size="lg"
            required
            minlength="6"
          />
        </UFormField>

        <UFormField label="确认密码" name="confirmPassword">
          <UInput
            v-model="form.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
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
          注册
        </UButton>
      </form>

      <template #footer>
        <p class="text-center text-gray-600 dark:text-gray-400">
          已有账户？
          <NuxtLink to="/login" class="text-primary hover:underline">
            立即登录
          </NuxtLink>
        </p>
      </template>
    </UCard>
  </div>
</template>
