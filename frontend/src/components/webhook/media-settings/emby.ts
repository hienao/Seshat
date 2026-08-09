import type { MediaSettingsDefinition } from './types'

export const embyMediaSettings: MediaSettingsDefinition = {
	appCode: 'emby',
	appName: 'Emby',
	serverPlaceholder: 'http://emby.example.com:8096',
	serverHelp: 'Enter an Emby URL reachable by the Seshat backend.',
	apiKeyHelp: 'Create an API key in the Emby server dashboard.',
}
