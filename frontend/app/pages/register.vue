<script setup lang="ts">
const { register, login, checkRegistrationAllowed } = useAuth()

const form = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

const loading = ref(false)
const error = ref('')
const registrationAllowed = ref(true)
const checkingStatus = ref(true)

// 检查是否允许注册
onMounted(async () => {
  try {
    registrationAllowed.value = await checkRegistrationAllowed()
  } catch (e) {
    registrationAllowed.value = false
  } finally {
    checkingStatus.value = false
  }
})

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
    navigateTo('/')
  } catch (e: any) {
    error.value = e.message || '注册失败'
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
            <UIcon name="i-heroicons-user-plus" class="w-10 h-10 text-primary-500" />
          </div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">创建账户</h1>
          <p class="mt-2 text-gray-600 dark:text-gray-400">注册新账户以开始使用</p>
        </div>
      </template>

      <!-- 加载状态 -->
      <div v-if="checkingStatus" class="flex justify-center py-8">
        <UIcon name="i-heroicons-arrow-path" class="w-8 h-8 text-primary-500 animate-spin" />
      </div>

      <!-- 不允许注册提示 -->
      <div v-else-if="!registrationAllowed" class="text-center py-8">
        <UIcon name="i-heroicons-lock-closed" class="w-16 h-16 mx-auto text-gray-400 mb-4" />
        <h2 class="text-lg font-medium text-gray-900 dark:text-white mb-2">注册功能已关闭</h2>
        <p class="text-gray-600 dark:text-gray-400 mb-6">系统当前不允许新用户注册，请联系管理员。</p>
        <NuxtLink to="/login">
          <UButton color="primary" size="lg">
            <UIcon name="i-heroicons-arrow-left" class="w-4 h-4 mr-2" />
            返回登录
          </UButton>
        </NuxtLink>
      </div>

      <!-- 注册表单 -->
      <form v-else @submit.prevent="handleSubmit" class="space-y-5">
        <UAlert v-if="error" color="error" :title="error" icon="i-heroicons-exclamation-circle" />
        
        <UFormField label="用户名" name="username" size="lg">
          <UInput
            v-model="form.username"
            placeholder="请输入用户名（3-50 个字符）"
            icon="i-heroicons-user"
            size="lg"
            class="w-full"
            required
            minlength="3"
            maxlength="50"
          />
        </UFormField>

        <UFormField label="密码" name="password" size="lg">
          <UInput
            v-model="form.password"
            type="password"
            placeholder="请输入密码（至少 6 位）"
            icon="i-heroicons-lock-closed"
            size="lg"
            class="w-full"
            required
            minlength="6"
          />
        </UFormField>

        <UFormField label="确认密码" name="confirmPassword" size="lg">
          <UInput
            v-model="form.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
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
          <UIcon name="i-heroicons-user-plus" class="w-5 h-5 mr-2" />
          注册
        </UButton>
      </form>

      <template #footer>
        <div class="text-center py-2">
          <p class="text-gray-600 dark:text-gray-400">
            已有账户？
            <NuxtLink to="/login" class="text-primary-500 hover:text-primary-600 font-medium hover:underline transition-colors">
              立即登录
            </NuxtLink>
          </p>
        </div>
      </template>
    </UCard>
  </div>
</template>
