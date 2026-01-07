<script setup lang="ts">
import type { User, SystemSettings } from '~/composables/useAuth'



const { user, isAdmin, listUsers, setUserRole, getSystemSettings, updateSystemSettings } = useAuth()

// 权限检查
const hasAccess = computed(() => isAdmin.value)

// 用户列表
const users = ref<User[]>([])
const loadingUsers = ref(false)
const userError = ref('')

// 系统设置
const settings = ref<SystemSettings>({ allow_register: false })
const loadingSettings = ref(false)
const settingsError = ref('')
const savingSettings = ref(false)

// 客户端就绪状态
const clientReady = ref(false)

// 加载用户列表
const fetchUsers = async () => {
  loadingUsers.value = true
  userError.value = ''
  try {
    users.value = await listUsers()
  } catch (e: any) {
    userError.value = e.message || '获取用户列表失败'
  } finally {
    loadingUsers.value = false
  }
}

// 加载系统设置
const fetchSettings = async () => {
  loadingSettings.value = true
  settingsError.value = ''
  try {
    settings.value = await getSystemSettings()
  } catch (e: any) {
    settingsError.value = e.message || '获取系统设置失败'
  } finally {
    loadingSettings.value = false
  }
}

// 切换用户管理员状态
const toggleAdmin = async (targetUser: User) => {
  if (targetUser.id === user.value?.id) {
    return // 不能修改自己
  }
  
  try {
    await setUserRole(targetUser.id, !targetUser.is_admin)
    await fetchUsers()
  } catch (e: any) {
    userError.value = e.message || '设置用户角色失败'
  }
}

// 保存系统设置
const saveSettings = async () => {
  savingSettings.value = true
  settingsError.value = ''
  try {
    await updateSystemSettings(settings.value)
  } catch (e: any) {
    settingsError.value = e.message || '保存设置失败'
  } finally {
    savingSettings.value = false
  }
}

// 初始化 - 只在客户端执行
onMounted(() => {
  clientReady.value = true
  if (hasAccess.value) {
    fetchUsers()
    fetchSettings()
  }
})

// 监听权限变化
watch(hasAccess, (newVal) => {
  if (newVal && clientReady.value) {
    fetchUsers()
    fetchSettings()
  }
})
</script>

<template>
  <div class="flex-1 bg-gray-50 dark:bg-gray-900 p-6">
    <ClientOnly>
      <div class="max-w-4xl mx-auto space-y-6">
        <!-- 标题 -->
        <div class="flex items-center gap-3">
          <div class="w-12 h-12 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
            <UIcon name="i-heroicons-cog-6-tooth" class="w-6 h-6 text-primary-500" />
          </div>
          <div>
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white">系统管理</h1>
            <p class="text-gray-600 dark:text-gray-400">管理用户和系统设置</p>
          </div>
        </div>

        <!-- 无权限提示 -->
        <UCard v-if="!hasAccess" class="text-center py-8">
          <UIcon name="i-heroicons-shield-exclamation" class="w-16 h-16 mx-auto text-gray-400 mb-4" />
          <h2 class="text-lg font-medium text-gray-900 dark:text-white mb-2">权限不足</h2>
          <p class="text-gray-600 dark:text-gray-400 mb-6">只有系统管理员可以访问此页面。</p>
          <NuxtLink to="/">
            <UButton color="primary" size="lg">
              <UIcon name="i-heroicons-home" class="w-4 h-4 mr-2" />
              返回首页
            </UButton>
          </NuxtLink>
        </UCard>

        <!-- 系统设置 -->
        <UCard v-if="hasAccess">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-cog-8-tooth" class="w-5 h-5 text-primary-500" />
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">系统设置</h2>
            </div>
          </template>

          <div v-if="loadingSettings" class="flex justify-center py-8">
            <UIcon name="i-heroicons-arrow-path" class="w-8 h-8 text-primary-500 animate-spin" />
          </div>

          <div v-else-if="settingsError">
            <UAlert color="error" :title="settingsError" icon="i-heroicons-exclamation-circle" />
          </div>

          <div v-else class="space-y-4">
            <div class="flex items-center justify-between p-4 rounded-lg bg-gray-50 dark:bg-gray-800">
              <div>
                <h3 class="font-medium text-gray-900 dark:text-white">允许用户注册</h3>
                <p class="text-sm text-gray-600 dark:text-gray-400">开启后，新用户可以自行注册账户</p>
              </div>
              <USwitch v-model="settings.allow_register" @update:model-value="saveSettings" :loading="savingSettings" />
            </div>
          </div>
        </UCard>

        <!-- 用户管理 -->
        <UCard v-if="hasAccess">
          <template #header>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-users" class="w-5 h-5 text-primary-500" />
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">用户管理</h2>
              </div>
              <UButton size="sm" variant="ghost" @click="fetchUsers" :loading="loadingUsers">
                <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" />
              </UButton>
            </div>
          </template>

          <div v-if="loadingUsers" class="flex justify-center py-8">
            <UIcon name="i-heroicons-arrow-path" class="w-8 h-8 text-primary-500 animate-spin" />
          </div>

          <div v-else-if="userError">
            <UAlert color="error" :title="userError" icon="i-heroicons-exclamation-circle" />
          </div>

          <div v-else class="divide-y divide-gray-200 dark:divide-gray-700">
            <div v-for="u in users" :key="u.id" class="flex items-center justify-between py-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
                  <UIcon name="i-heroicons-user" class="w-5 h-5 text-primary-500" />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-gray-900 dark:text-white">{{ u.username }}</span>
                    <UBadge v-if="u.is_admin" color="primary" size="xs">管理员</UBadge>
                    <UBadge v-if="u.id === user?.id" color="neutral" size="xs">当前用户</UBadge>
                  </div>
                  <span class="text-sm text-gray-500">ID: {{ u.id }}</span>
                </div>
              </div>
              <div>
                <UButton
                  v-if="u.id !== user?.id"
                  size="sm"
                  :color="u.is_admin ? 'error' : 'primary'"
                  variant="soft"
                  @click="toggleAdmin(u)"
                >
                  {{ u.is_admin ? '撤销管理员' : '设为管理员' }}
                </UButton>
              </div>
            </div>
          </div>
        </UCard>
      </div>

      <template #fallback>
        <div class="max-w-4xl mx-auto flex justify-center py-16">
          <UIcon name="i-heroicons-arrow-path" class="w-8 h-8 text-primary-500 animate-spin" />
        </div>
      </template>
    </ClientOnly>
  </div>
</template>
