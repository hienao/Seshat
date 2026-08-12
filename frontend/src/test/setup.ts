import '@testing-library/jest-dom/vitest'
import { beforeEach } from 'vitest'
import { i18n } from '@/i18n'
import { LANGUAGE_STORAGE_KEY } from '@/i18n/language-preference'

beforeEach(async () => {
  localStorage.setItem(LANGUAGE_STORAGE_KEY, 'zh-CN')
  await i18n.changeLanguage('zh-CN')
  document.documentElement.lang = 'zh-CN'
})
