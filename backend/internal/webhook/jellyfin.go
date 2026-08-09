package webhook

import (
	"math"
	"strings"
)

type jellyfinProvider struct {
	definition AppDefinition
	aliases    map[string]string
}

func NewJellyfinProvider() Provider {
	return &jellyfinProvider{
		definition: AppDefinition{
			Code:             "jellyfin",
			Name:             "Jellyfin",
			Description:      "Jellyfin 媒体库、播放、用户与系统事件",
			AuthMode:         "secret_header",
			DefaultEventType: DefaultEventType,
			EventTypes:       jellyfinEventTypes(),
		},
		aliases: map[string]string{
			"itemadded": "media_added", "librarynew": "media_added", "itemdeleted": "media_deleted", "librarydeleted": "media_deleted",
			"playbackstart": "playback_started", "playbackprogress": "playback_progress", "playbackstop": "playback_stopped",
			"authenticationsuccess": "authentication_success", "authenticationfailure": "authentication_failure", "sessionstart": "session_started",
			"pendingrestart": "server_restart_required", "taskcompleted": "task_completed", "subtitledownloadfailure": "subtitle_download_failed",
			"plugininstalling": "plugin_installing", "plugininstalled": "plugin_installed", "plugininstallationfailed": "plugin_install_failed",
			"plugininstallationcancelled": "plugin_install_cancelled", "pluginupdated": "plugin_updated", "pluginuninstalled": "plugin_uninstalled",
			"usercreated": "user_created", "userupdated": "user_updated", "userdeleted": "user_deleted", "userlockedout": "user_locked_out",
			"userpasswordchanged": "user_password_changed", "userdatasaved": "user_data_saved", "generic": "generic",
		},
	}
}

func jellyfinEventTypes() []EventTypeDefinition {
	return []EventTypeDefinition{
		{Code: "media_added", Name: "新增媒体", RenderMode: "custom"},
		{Code: "media_deleted", Name: "删除媒体", RenderMode: "custom"},
		{Code: "playback_started", Name: "开始播放", RenderMode: "custom"},
		{Code: "playback_progress", Name: "播放进度", RenderMode: "custom"},
		{Code: "playback_stopped", Name: "停止播放", RenderMode: "custom"},
		{Code: "authentication_success", Name: "登录成功", RenderMode: "custom"},
		{Code: "authentication_failure", Name: "登录失败", RenderMode: "custom"},
		{Code: "session_started", Name: "会话开始", RenderMode: "custom"},
		{Code: "server_restart_required", Name: "服务等待重启", RenderMode: "custom"},
		{Code: "task_completed", Name: "计划任务完成", RenderMode: "custom"},
		{Code: "subtitle_download_failed", Name: "字幕下载失败", RenderMode: "custom"},
		{Code: "plugin_installing", Name: "正在安装插件", RenderMode: "custom"},
		{Code: "plugin_installed", Name: "插件安装完成", RenderMode: "custom"},
		{Code: "plugin_install_failed", Name: "插件安装失败", RenderMode: "custom"},
		{Code: "plugin_install_cancelled", Name: "插件安装取消", RenderMode: "custom"},
		{Code: "plugin_updated", Name: "插件已更新", RenderMode: "custom"},
		{Code: "plugin_uninstalled", Name: "插件已卸载", RenderMode: "custom"},
		{Code: "user_created", Name: "用户已创建", RenderMode: "custom"},
		{Code: "user_updated", Name: "用户信息更新", RenderMode: "custom"},
		{Code: "user_deleted", Name: "用户已删除", RenderMode: "custom"},
		{Code: "user_locked_out", Name: "用户被锁定", RenderMode: "custom"},
		{Code: "user_password_changed", Name: "用户密码已修改", RenderMode: "custom"},
		{Code: "user_data_saved", Name: "用户数据已保存", RenderMode: "custom"},
		{Code: "generic", Name: "通用通知", RenderMode: "custom"},
		{Code: DefaultEventType, Name: "其他消息", RenderMode: "raw"},
	}
}

func (p *jellyfinProvider) Code() string              { return p.definition.Code }
func (p *jellyfinProvider) Definition() AppDefinition { return p.definition }
func (p *jellyfinProvider) Verify(secret string, request IncomingRequest) bool {
	return verifyRequest(secret, request)
}
func (p *jellyfinProvider) DetectType(request IncomingRequest) string {
	payload := decodePayload(request.Body)
	if value := firstString(payload, "NotificationType", "notification_type", "Event", "event", "Type", "type"); value != "" {
		return value
	}
	return "unknown"
}
func (p *jellyfinProvider) MapType(sourceEventType string) string {
	compact := compactEventType(sourceEventType)
	if eventType, ok := p.aliases[compact]; ok {
		return eventType
	}
	for _, definition := range p.definition.EventTypes {
		if compact == compactEventType(definition.Code) {
			return definition.Code
		}
	}
	return sourceEventType
}
func (p *jellyfinProvider) ExternalEventID(request IncomingRequest) string {
	payload := decodePayload(request.Body)
	return firstString(payload, "NotificationId", "notification_id", "EventId", "event_id", "IdempotencyKey")
}
func (p *jellyfinProvider) Normalize(eventType string, request IncomingRequest) Presentation {
	payload := decodePayload(request.Body)
	item := nestedMap(payload, "Item", "item")
	user := nestedMap(payload, "User", "user")
	session := nestedMap(payload, "Session", "session")
	playState := nestedMap(payload, "PlaybackInfo", "playback_info", "PlayState", "play_state")
	if len(playState) == 0 {
		playState = nestedMap(session, "PlayState", "play_state")
	}
	task := nestedMap(payload, "TaskInfo", "task_info", "Task", "task")

	category := eventCategory(eventType)
	media, actor, playback, system := map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}
	if category == "media" || category == "playback" {
		media = normalizeJellyfinMedia(payload, item)
	}
	if category == "playback" || category == "security" || category == "user" {
		actor = normalizeJellyfinActor(payload, user, session)
	}
	if category == "playback" {
		playback = normalizeJellyfinPlayback(payload, playState, media)
	}
	if category == "system" {
		system = normalizeJellyfinSystem(eventType, payload, task)
	}
	return buildMediaPresentation(p.definition.Name, p.definition.EventTypes, eventType, p.DetectType(request), request.Body, payload, media, actor, playback, system)
}

func normalizeJellyfinMedia(payload, item map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{item, payload}
	media := map[string]interface{}{}
	putString(media, "id", firstStringFrom(values, "Id", "ItemId", "item_id"))
	putString(media, "name", firstStringFrom(values, "Name", "ItemName", "item_name"))
	putString(media, "type", firstStringFrom(values, "Type", "ItemType", "item_type"))
	putString(media, "series", firstStringFrom(values, "SeriesName", "series_name"))
	putString(media, "season", firstStringFrom(values, "SeasonNumber", "SeasonNumber00", "ParentIndexNumber", "season_number"))
	putString(media, "episode", firstStringFrom(values, "EpisodeNumber", "EpisodeNumber00", "IndexNumber", "episode_number"))
	putString(media, "year", firstStringFrom(values, "ProductionYear", "Year", "year"))
	if premiereDate := firstStringFrom(values, "SeriesPremiereDate", "series_premiere_date"); len(premiereDate) >= 4 && digitsOnly(premiereDate[:4]) {
		media["series_year"] = premiereDate[:4]
	}
	putString(media, "overview", firstStringFrom(values, "Overview", "Description", "overview"))
	putString(media, "library", firstStringFrom(values, "LibraryName", "CollectionName", "library_name"))
	putString(media, "image_url", safeURL(firstStringFrom(values, "ImageUrl", "PosterUrl", "PrimaryImageUrl", "image_url")))
	if providers := normalizeJellyfinProviderIDs(payload, item); len(providers) > 0 {
		media["provider_ids"] = providers
	}
	if video := normalizeJellyfinVideo(payload, item); len(video) > 0 {
		media["video"] = video
	}
	if ticks, ok := firstFloat(values, "RunTimeTicks", "RuntimeTicks", "run_time_ticks"); ok && ticks > 0 {
		seconds := ticks / 10_000_000
		media["duration_seconds"] = math.Round(seconds)
		media["duration_label"] = formatDuration(seconds)
	}
	putString(media, "display_name", mediaDisplayName(media))
	return media
}

func normalizeJellyfinActor(payload, user, session map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{user, session, payload}
	actor := map[string]interface{}{}
	putString(actor, "user_id", firstStringFrom(values, "Id", "UserId", "user_id"))
	putString(actor, "username", firstStringFrom(values, "Name", "NotificationUsername", "UserName", "Username", "user_name"))
	putString(actor, "device", firstStringFrom(values, "DeviceName", "device_name"))
	putString(actor, "client", firstStringFrom(values, "ClientName", "Client", "AppName", "client_name"))
	putString(actor, "remote_ip", firstStringFrom(values, "RemoteEndPoint", "RemoteEndpoint", "RemoteAddress", "remote_ip"))
	return actor
}

func normalizeJellyfinProviderIDs(payload, item map[string]interface{}) map[string]interface{} {
	providers := map[string]interface{}{}
	providerMaps := []map[string]interface{}{nestedMap(item, "ProviderIds", "provider_ids"), nestedMap(payload, "ProviderIds", "provider_ids")}
	for provider, keys := range map[string][]string{
		"tmdb": {"Tmdb", "TMDB", "tmdb", "Provider_tmdb"},
		"tvdb": {"Tvdb", "TVDB", "tvdb", "Provider_tvdb"},
		"imdb": {"Imdb", "IMDB", "imdb", "Provider_imdb"},
	} {
		if value := firstStringFrom(append(providerMaps, item, payload), keys...); value != "" {
			providers[provider] = value
		}
	}
	return providers
}

func normalizeJellyfinVideo(payload, item map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{item, payload}
	video := map[string]interface{}{}
	putString(video, "title", firstStringFrom(values, "Video_0_Title", "VideoTitle"))
	putString(video, "codec", firstStringFrom(values, "Video_0_Codec", "VideoCodec"))
	putString(video, "profile", firstStringFrom(values, "Video_0_Profile", "VideoProfile"))
	putString(video, "width", firstStringFrom(values, "Video_0_Width", "VideoWidth"))
	putString(video, "height", firstStringFrom(values, "Video_0_Height", "VideoHeight"))
	putString(video, "range", firstStringFrom(values, "Video_0_VideoRange", "VideoRange", "VideoRangeType"))
	for _, container := range []map[string]interface{}{item, payload} {
		streams, ok := lookup(container, "MediaStreams")
		if !ok {
			continue
		}
		items, ok := streams.([]interface{})
		if !ok {
			continue
		}
		for _, candidate := range items {
			stream, ok := candidate.(map[string]interface{})
			if !ok || !strings.EqualFold(firstStringFrom([]map[string]interface{}{stream}, "Type"), "Video") {
				continue
			}
			putJellyfinStringIfEmpty(video, "title", firstStringFrom([]map[string]interface{}{stream}, "DisplayTitle", "Title"))
			putJellyfinStringIfEmpty(video, "codec", firstStringFrom([]map[string]interface{}{stream}, "Codec"))
			putJellyfinStringIfEmpty(video, "profile", firstStringFrom([]map[string]interface{}{stream}, "Profile"))
			putJellyfinStringIfEmpty(video, "width", firstStringFrom([]map[string]interface{}{stream}, "Width"))
			putJellyfinStringIfEmpty(video, "height", firstStringFrom([]map[string]interface{}{stream}, "Height"))
			putJellyfinStringIfEmpty(video, "range", firstStringFrom([]map[string]interface{}{stream}, "VideoRange", "VideoRangeType"))
			break
		}
	}
	putString(video, "display_label", videoDisplayLabel(video))
	return video
}

func putJellyfinStringIfEmpty(target map[string]interface{}, key, value string) {
	if stringValue(target, key) == "" {
		putString(target, key, value)
	}
}

func normalizeJellyfinPlayback(payload, playState, media map[string]interface{}) map[string]interface{} {
	return normalizePlaybackValues([]map[string]interface{}{playState, payload}, media)
}

func normalizeJellyfinSystem(eventType string, payload, task map[string]interface{}) map[string]interface{} {
	return normalizeSystemValues(eventType, []map[string]interface{}{task, payload}, payload)
}
