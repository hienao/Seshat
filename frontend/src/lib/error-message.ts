import { i18n } from '@/i18n'

export function errorMessage(error: unknown, fallback = i18n.t('common.feedback.operationFailed')) {
  return error instanceof Error && error.message ? error.message : fallback
}
