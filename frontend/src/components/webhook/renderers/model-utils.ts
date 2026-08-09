import type { DisplayWebhookEvent } from '@/api/types'
import type { DataRecord, EventCategory } from './types'

export const appNames: Record<string, string> = { jellyfin: 'Jellyfin', emby: 'Emby', github: 'GitHub', generic: '通用 Webhook' }

export function resolveCategory(event: DisplayWebhookEvent, data: DataRecord): EventCategory {
  const explicit = text(data.category)
  if (['media', 'playback', 'security', 'user', 'system', 'raw'].includes(explicit)) return explicit as EventCategory
  if (event.is_fallback || event.display_event_type === '__default__') return 'raw'
  if (event.display_event_type.startsWith('media_')) return 'media'
  if (event.display_event_type.startsWith('playback_')) return 'playback'
  if (event.display_event_type.startsWith('authentication_') || event.display_event_type === 'user_locked_out') return 'security'
  if (event.display_event_type.startsWith('user_')) return 'user'
  return 'system'
}

export function eventLabel(eventType: string, fallback: boolean) {
  if (fallback) return '其他消息'
  const labels: Record<string, string> = {
    media_added: '新增媒体', media_deleted: '删除媒体', playback_started: '开始播放', playback_progress: '播放进度', playback_stopped: '停止播放',
    authentication_success: '登录成功', authentication_failure: '登录失败', session_started: '会话开始', server_restart_required: '服务等待重启',
    task_completed: '计划任务完成', subtitle_download_failed: '字幕下载失败', plugin_installing: '正在安装插件', plugin_installed: '插件安装完成',
    plugin_install_failed: '插件安装失败', plugin_install_cancelled: '插件安装取消', plugin_updated: '插件已更新', plugin_uninstalled: '插件已卸载',
    user_created: '用户已创建', user_updated: '用户信息更新', user_deleted: '用户已删除', user_locked_out: '用户被锁定',
    user_password_changed: '用户密码已修改', user_data_saved: '用户数据已保存', generic: '通用通知',
  }
  return labels[eventType] ?? eventType
}

export function record(value: unknown): DataRecord {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as DataRecord : {}
}

export function text(value: unknown) {
  return typeof value === 'string' ? value : ''
}

export function number(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

export function normalizedSummary(summary: string, fallback: string, category: EventCategory, media: DataRecord) {
  const value = summary.trim()
  if (category === 'media' && value && value === text(media.type)) return fallback
  return value || fallback
}

export function tagsWithoutMediaType(tags: string[], media: DataRecord) {
  const mediaType = text(media.type)
  return tags.filter((tag, index) => tag && tag !== mediaType && tags.indexOf(tag) === index)
}

export function safeWebUrl(value: string) {
	if (value.startsWith('/api/public/media-images/') && !value.startsWith('//')) return value
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : ''
  } catch { return '' }
}

export function formatRawPreview(value: string) {
  try { return JSON.stringify(JSON.parse(value), null, 2) }
  catch { return value }
}
