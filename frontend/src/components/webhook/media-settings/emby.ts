import type { MediaSettingsDefinition } from './types'

export const embyMediaSettings: MediaSettingsDefinition = {
	appCode: 'emby',
	appName: 'Emby',
	serverPlaceholder: 'http://emby.example.com:8096',
	serverHelp: '填写 Seshat 后端能够访问的 Emby 服务器地址；无需在末尾添加 /emby。',
	apiKeyHelp: '在 Emby Server Dashboard → Advanced → Security 中创建 API Key。媒体详情和图片优先从该实例获取。',
}
