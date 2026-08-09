import { i18n } from './index'

export function currentLocale() {
  return i18n.resolvedLanguage === 'zh-CN' ? 'zh-CN' : 'en-US'
}

export function formatDateTime(value?: string | Date) {
  if (!value) return '-'
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat(currentLocale(), { dateStyle: 'medium', timeStyle: 'medium' }).format(date)
}
