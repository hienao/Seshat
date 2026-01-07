<script setup lang="ts">
const currentYear = new Date().getFullYear()
</script>

<template>
  <footer class="border-t border-gray-200 dark:border-gray-700 bg-white/50 dark:bg-gray-800/50 backdrop-blur-sm">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="flex flex-col md:flex-row justify-between items-center gap-4">
        <!-- 左侧 Logo 和版权 -->
        <div class="flex items-center gap-3">
          <NuxtLink to="/" class="text-lg font-bold text-primary-500 hover:text-primary-600 transition-colors">
            BaseGoApp
          </NuxtLink>
          <span class="text-gray-400 dark:text-gray-500">|</span>
          <span class="text-gray-500 dark:text-gray-400 text-sm">
            © {{ currentYear }} BaseGoApp
          </span>
        </div>

        <!-- 右侧链接 -->
        <div class="flex items-center gap-6 text-sm">
          <span class="text-gray-500 dark:text-gray-400">
            使用 Nuxt 3 + Go Gin 构建
          </span>
          <a 
            href="/swagger/index.html" 
            target="_blank"
            class="text-primary-500 hover:text-primary-600 transition-colors flex items-center gap-1"
          >
            <UIcon name="i-heroicons-document-text" class="w-4 h-4" />
            API 文档
          </a>
        </div>
      </div>
    </div>
  </footer>
</template>
