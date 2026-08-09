import type { MediaSettingsDefinition } from './types'

export const jellyfinMediaSettings: MediaSettingsDefinition = {
	appCode: 'jellyfin',
	appName: 'Jellyfin',
	serverPlaceholder: 'http://jellyfin.example.com:8096',
	serverHelp: '填写 Seshat 后端能够访问的 Jellyfin 地址；若配置了 Base URL，请一并填写。',
	apiKeyHelp: '在 Jellyfin 控制台 → 高级 → API 密钥中创建。媒体详情和图片优先从该实例获取。',
}
