import { embyMediaSettings } from './emby'
import { jellyfinMediaSettings } from './jellyfin'
import type { MediaSettingsDefinition } from './types'

const definitions = new Map<string, MediaSettingsDefinition>([
	[jellyfinMediaSettings.appCode, jellyfinMediaSettings],
	[embyMediaSettings.appCode, embyMediaSettings],
])

export function mediaSettingsDefinition(appCode: string) {
	return definitions.get(appCode)
}
