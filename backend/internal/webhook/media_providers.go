package webhook

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

type mediaServerProvider struct {
	definition    AppDefinition
	aliases       map[string]string
	requireSecret bool
}

func NewJellyfinProvider() Provider {
	return newMediaServerProvider(
		"jellyfin",
		"Jellyfin",
		"Jellyfin 媒体库、播放、用户与系统事件",
		true,
		map[string]string{
			"itemadded": "media_added", "librarynew": "media_added", "itemdeleted": "media_deleted", "librarydeleted": "media_deleted",
			"playbackstart": "playback_started", "playbackprogress": "playback_progress", "playbackstop": "playback_stopped",
			"authenticationsuccess": "authentication_success", "authenticationfailure": "authentication_failure", "sessionstart": "session_started",
			"pendingrestart": "server_restart_required", "taskcompleted": "task_completed", "subtitledownloadfailure": "subtitle_download_failed",
			"plugininstalling": "plugin_installing", "plugininstalled": "plugin_installed", "plugininstallationfailed": "plugin_install_failed",
			"plugininstallationcancelled": "plugin_install_cancelled", "pluginupdated": "plugin_updated", "pluginuninstalled": "plugin_uninstalled",
			"usercreated": "user_created", "userupdated": "user_updated", "userdeleted": "user_deleted", "userlockedout": "user_locked_out",
			"userpasswordchanged": "user_password_changed", "userdatasaved": "user_data_saved", "generic": "generic",
		},
	)
}

func NewEmbyProvider() Provider {
	return newMediaServerProvider(
		"emby",
		"Emby",
		"Emby 媒体库、播放、用户与系统事件",
		false,
		map[string]string{
			"librarynew": "media_added", "mediaadded": "media_added", "itemadded": "media_added",
			"librarydeleted": "media_deleted", "mediadeleted": "media_deleted", "itemdeleted": "media_deleted", "itemremoved": "media_deleted",
			"playbackstart": "playback_started", "mediaplay": "playback_started",
			"playbackprogress": "playback_progress", "mediapause": "playback_progress", "mediaresume": "playback_progress", "playbackpause": "playback_progress", "playbackresume": "playback_progress",
			"playbackstop": "playback_stopped", "mediastop": "playback_stopped",
			"authenticationsuccess": "authentication_success", "userauthenticated": "authentication_success",
			"authenticationfailure": "authentication_failure", "userauthenticationfailed": "authentication_failure", "sessionstart": "session_started",
			"pendingrestart": "server_restart_required", "serverrestartrequired": "server_restart_required", "taskcompleted": "task_completed",
			"subtitledownloadfailure": "subtitle_download_failed", "subtitledownloadfailed": "subtitle_download_failed",
			"plugininstalling": "plugin_installing", "plugininstalled": "plugin_installed", "plugininstallationfailed": "plugin_install_failed",
			"plugininstallationcancelled": "plugin_install_cancelled", "pluginupdated": "plugin_updated", "pluginuninstalled": "plugin_uninstalled",
			"usercreated": "user_created", "userupdated": "user_updated", "userconfigurationupdated": "user_updated", "userpolicyupdated": "user_updated",
			"userdeleted": "user_deleted", "userlockedout": "user_locked_out", "userpasswordchanged": "user_password_changed", "userdatasaved": "user_data_saved",
			"generic": "generic",
		},
	)
}

func newMediaServerProvider(code, name, description string, requireSecret bool, aliases map[string]string) Provider {
	return &mediaServerProvider{definition: AppDefinition{
		Code: code, Name: name, Description: description, DefaultEventType: DefaultEventType,
		AuthMode:   map[bool]string{true: "secret_header", false: "endpoint_url"}[requireSecret],
		EventTypes: mediaServerEventTypes(),
	}, aliases: aliases, requireSecret: requireSecret}
}

func mediaServerEventTypes() []EventTypeDefinition {
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

func (p *mediaServerProvider) Code() string              { return p.definition.Code }
func (p *mediaServerProvider) Definition() AppDefinition { return p.definition }
func (p *mediaServerProvider) Verify(secret string, request IncomingRequest) bool {
	if !p.requireSecret {
		return true
	}
	return verifyRequest(secret, request)
}
func (p *mediaServerProvider) DetectType(request IncomingRequest) string {
	payload := decodePayload(request.Body)
	if value := firstString(payload, "NotificationType", "notification_type", "Event", "event", "Type", "type"); value != "" {
		return value
	}
	return "unknown"
}
func (p *mediaServerProvider) MapType(sourceEventType string) string {
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
func (p *mediaServerProvider) ExternalEventID(request IncomingRequest) string {
	payload := decodePayload(request.Body)
	return firstString(payload, "NotificationId", "notification_id", "EventId", "event_id", "IdempotencyKey")
}
func (p *mediaServerProvider) Normalize(eventType string, request IncomingRequest) Presentation {
	payload := decodePayload(request.Body)
	item := nestedMap(payload, "Item", "item")
	user := nestedMap(payload, "User", "user")
	playState := nestedMap(payload, "PlaybackInfo", "playback_info", "PlayState", "play_state")
	task := nestedMap(payload, "TaskInfo", "task_info", "Task", "task")

	category := eventCategory(eventType)
	media, actor, playback, system := map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}
	if category == "media" || category == "playback" {
		media = normalizeMedia(payload, item)
	}
	if category == "playback" || category == "security" || category == "user" {
		actor = normalizeActor(payload, user)
	}
	if category == "playback" {
		playback = normalizePlayback(payload, playState, media)
	}
	if category == "system" {
		system = normalizeSystem(eventType, payload, task)
	}
	label := eventTypeName(eventType)
	sourceType := p.DetectType(request)

	data := map[string]interface{}{"category": category, "event_label": label, "source_event_type": sourceType}
	if category == "raw" {
		data["raw_preview"] = bodyPreview(request.Body, 600)
	}
	if len(media) > 0 {
		data["media"] = media
	}
	if len(actor) > 0 {
		data["actor"] = actor
	}
	if len(playback) > 0 {
		data["playback"] = playback
	}
	if len(system) > 0 {
		data["system"] = system
	}

	subject := presentationSubject(category, media, actor, system)
	title := p.definition.Name + " · " + label
	if subject != "" {
		title += " · " + subject
	}
	summary := presentationSummary(p.definition.Name, label, category, media, actor, playback, system)
	facts := presentationFacts(category, media, actor, playback, system)
	links := presentationLinks(payload)
	tags := presentationTags(media, playback)
	return Presentation{SchemaVersion: 1, Title: title, Summary: summary, Severity: eventSeverity(eventType, system), Tags: tags, Facts: facts, Links: links, Data: data}
}

func decodePayload(body []byte) map[string]interface{} {
	payload := map[string]interface{}{}
	_ = json.Unmarshal(body, &payload)
	return payload
}

func compactEventType(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.TrimSpace(value))
}

func nestedMap(payload map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		if value, ok := lookup(payload, key); ok {
			if result, ok := value.(map[string]interface{}); ok {
				return result
			}
		}
	}
	return map[string]interface{}{}
}

func lookup(payload map[string]interface{}, key string) (interface{}, bool) {
	if value, ok := payload[key]; ok {
		return value, true
	}
	for candidate, value := range payload {
		if strings.EqualFold(candidate, key) {
			return value, true
		}
	}
	return nil, false
}

func firstStringFrom(payloads []map[string]interface{}, keys ...string) string {
	for _, payload := range payloads {
		for _, key := range keys {
			value, ok := lookup(payload, key)
			if !ok || value == nil {
				continue
			}
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
				}
			case json.Number:
				return typed.String()
			case float64:
				return strconv.FormatFloat(typed, 'f', -1, 64)
			}
		}
	}
	return ""
}

func firstFloat(payloads []map[string]interface{}, keys ...string) (float64, bool) {
	for _, payload := range payloads {
		for _, key := range keys {
			value, ok := lookup(payload, key)
			if !ok {
				continue
			}
			switch typed := value.(type) {
			case float64:
				return typed, true
			case string:
				parsed, err := strconv.ParseFloat(typed, 64)
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func firstBool(payloads []map[string]interface{}, keys ...string) (bool, bool) {
	for _, payload := range payloads {
		for _, key := range keys {
			value, ok := lookup(payload, key)
			if !ok {
				continue
			}
			switch typed := value.(type) {
			case bool:
				return typed, true
			case string:
				parsed, err := strconv.ParseBool(typed)
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return false, false
}

func normalizeMedia(payload, item map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{item, payload}
	media := map[string]interface{}{}
	putString(media, "id", firstStringFrom(values, "Id", "ItemId", "item_id"))
	putString(media, "name", firstStringFrom(values, "Name", "ItemName", "item_name"))
	putString(media, "type", firstStringFrom(values, "Type", "ItemType", "item_type"))
	putString(media, "series", firstStringFrom(values, "SeriesName", "series_name"))
	putString(media, "season", firstStringFrom(values, "SeasonNumber", "ParentIndexNumber", "season_number"))
	putString(media, "episode", firstStringFrom(values, "EpisodeNumber", "IndexNumber", "episode_number"))
	putString(media, "year", firstStringFrom(values, "ProductionYear", "Year", "year"))
	putString(media, "overview", firstStringFrom(values, "Overview", "Description", "overview"))
	putString(media, "library", firstStringFrom(values, "LibraryName", "CollectionName", "library_name"))
	putString(media, "image_url", safeURL(firstStringFrom(values, "ImageUrl", "PosterUrl", "PrimaryImageUrl", "image_url")))
	if ticks, ok := firstFloat(values, "RunTimeTicks", "RuntimeTicks", "run_time_ticks"); ok && ticks > 0 {
		seconds := ticks / 10_000_000
		media["duration_seconds"] = math.Round(seconds)
		media["duration_label"] = formatDuration(seconds)
	}
	media["display_name"] = mediaDisplayName(media)
	if media["display_name"] == "" {
		delete(media, "display_name")
	}
	return media
}

func normalizeActor(payload, user map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{user, payload}
	actor := map[string]interface{}{}
	putString(actor, "user_id", firstStringFrom(values, "Id", "UserId", "user_id"))
	putString(actor, "username", firstStringFrom(values, "Name", "NotificationUsername", "UserName", "Username", "user_name"))
	putString(actor, "device", firstStringFrom(values, "DeviceName", "device_name"))
	putString(actor, "client", firstStringFrom(values, "ClientName", "AppName", "client_name"))
	putString(actor, "remote_ip", firstStringFrom(values, "RemoteEndPoint", "RemoteEndpoint", "RemoteAddress", "remote_ip"))
	return actor
}

func normalizePlayback(payload, playState, media map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{playState, payload}
	playback := map[string]interface{}{}
	putString(playback, "method", firstStringFrom(values, "PlayMethod", "play_method"))
	if paused, ok := firstBool(values, "IsPaused", "Paused", "is_paused"); ok {
		playback["paused"] = paused
	}
	if completed, ok := firstBool(values, "PlayedToCompletion", "IsCompleted", "played_to_completion"); ok {
		playback["completed"] = completed
	}
	positionTicks, hasPosition := firstFloat(values, "PlaybackPositionTicks", "PositionTicks", "position_ticks")
	if hasPosition {
		position := positionTicks / 10_000_000
		playback["position_seconds"] = math.Round(position)
		playback["position_label"] = formatDuration(position)
	}
	duration, _ := media["duration_seconds"].(float64)
	if hasPosition && duration > 0 {
		playback["percent"] = math.Round(math.Min(100, math.Max(0, (positionTicks/10_000_000)/duration*100))*10) / 10
	}
	return playback
}

func normalizeSystem(eventType string, payload, task map[string]interface{}) map[string]interface{} {
	values := []map[string]interface{}{task, payload}
	system := map[string]interface{}{}
	if strings.HasPrefix(eventType, "plugin_") {
		putString(system, "name", firstStringFrom(values, "PluginName", "PackageName", "Name"))
		putString(system, "version", firstStringFrom(values, "PluginVersion", "Version", "version"))
	} else if eventType == "task_completed" {
		putString(system, "name", firstStringFrom(values, "Name", "TaskName", "name"))
	}
	putString(system, "status", firstStringFrom(values, "Status", "State", "status"))
	putString(system, "error", firstStringFrom(values, "ErrorMessage", "Error", "FailureMessage", "error"))
	putString(system, "server_name", firstStringFrom([]map[string]interface{}{payload}, "ServerName", "server_name"))
	putString(system, "server_version", firstStringFrom([]map[string]interface{}{payload}, "ServerVersion", "server_version"))
	return system
}

func putString(target map[string]interface{}, key, value string) {
	if value != "" {
		target[key] = value
	}
}

func mediaDisplayName(media map[string]interface{}) string {
	name, _ := media["name"].(string)
	series, _ := media["series"].(string)
	season, _ := media["season"].(string)
	episode, _ := media["episode"].(string)
	year, _ := media["year"].(string)
	if series != "" {
		index := ""
		if season != "" || episode != "" {
			index = fmt.Sprintf("S%02sE%02s", season, episode)
		}
		return strings.Trim(strings.Join(nonEmptyStrings(series, index, name), " · "), " ·")
	}
	if name != "" && year != "" {
		return name + "（" + year + "）"
	}
	return name
}

func formatDuration(seconds float64) string {
	total := int64(math.Round(seconds))
	if total < 0 {
		total = 0
	}
	hours, minutes, secs := total/3600, total%3600/60, total%60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func presentationSubject(category string, media, actor, system map[string]interface{}) string {
	if category == "media" || category == "playback" {
		return stringValue(media, "display_name")
	}
	if category == "security" || category == "user" {
		return stringValue(actor, "username")
	}
	return stringValue(system, "name")
}

func presentationSummary(app, label, category string, media, actor, playback, system map[string]interface{}) string {
	if category == "playback" {
		parts := nonEmptyStrings(stringValue(actor, "username"), stringValue(actor, "device"), stringValue(playback, "method"))
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
	}
	if category == "media" {
		parts := nonEmptyStrings(stringValue(media, "type"), stringValue(media, "library"), stringValue(media, "duration_label"))
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
	}
	if category == "security" || category == "user" {
		parts := nonEmptyStrings(stringValue(actor, "client"), stringValue(actor, "device"))
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
	}
	if errorMessage := stringValue(system, "error"); errorMessage != "" {
		return errorMessage
	}
	return "收到 " + app + " " + label + "事件"
}

func presentationFacts(category string, media, actor, playback, system map[string]interface{}) []map[string]string {
	facts := []map[string]string{}
	addFact := func(label, value string) {
		if value != "" && len(facts) < 6 {
			facts = append(facts, map[string]string{"label": label, "value": value})
		}
	}
	if category == "media" || category == "playback" {
		addFact("媒体", stringValue(media, "display_name"))
		addFact("类型", stringValue(media, "type"))
		addFact("用户", stringValue(actor, "username"))
		addFact("设备", stringValue(actor, "device"))
		addFact("播放方式", stringValue(playback, "method"))
		addFact("进度", playbackProgressLabel(playback, media))
	} else if category == "security" || category == "user" {
		addFact("用户", stringValue(actor, "username"))
		addFact("客户端", stringValue(actor, "client"))
		addFact("设备", stringValue(actor, "device"))
		addFact("来源", stringValue(actor, "remote_ip"))
	} else {
		addFact("名称", stringValue(system, "name"))
		addFact("版本", stringValue(system, "version"))
		addFact("状态", stringValue(system, "status"))
		addFact("服务端", stringValue(system, "server_name"))
		addFact("服务版本", stringValue(system, "server_version"))
		addFact("错误", stringValue(system, "error"))
	}
	return facts
}

func presentationLinks(payload map[string]interface{}) []map[string]string {
	if itemURL := safeURL(firstString(payload, "ItemUrl", "item_url")); itemURL != "" {
		return []map[string]string{{"label": "打开媒体", "url": itemURL}}
	}
	if serverURL := safeURL(firstString(payload, "ServerUrl", "server_url")); serverURL != "" {
		return []map[string]string{{"label": "打开服务", "url": serverURL}}
	}
	return nil
}

func presentationTags(media, playback map[string]interface{}) []string {
	tags := nonEmptyStrings(stringValue(media, "type"), stringValue(playback, "method"))
	if paused, ok := playback["paused"].(bool); ok && paused {
		tags = append(tags, "已暂停")
	}
	if completed, ok := playback["completed"].(bool); ok && completed {
		tags = append(tags, "已完成")
	}
	return tags
}

func playbackProgressLabel(playback, media map[string]interface{}) string {
	position := stringValue(playback, "position_label")
	duration := stringValue(media, "duration_label")
	if position != "" && duration != "" {
		return position + " / " + duration
	}
	return position
}

func eventCategory(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "media_"):
		return "media"
	case strings.HasPrefix(eventType, "playback_"):
		return "playback"
	case strings.HasPrefix(eventType, "authentication_") || eventType == "user_locked_out" || eventType == "session_started":
		return "security"
	case strings.HasPrefix(eventType, "user_"):
		return "user"
	case eventType == DefaultEventType || eventType == "generic":
		return "raw"
	default:
		return "system"
	}
}

func eventTypeName(eventType string) string {
	for _, definition := range mediaServerEventTypes() {
		if definition.Code == eventType {
			return definition.Name
		}
	}
	return "其他消息"
}

func eventSeverity(eventType string, system map[string]interface{}) string {
	status := strings.ToLower(stringValue(system, "status"))
	if strings.Contains(status, "fail") || stringValue(system, "error") != "" {
		return "error"
	}
	switch eventType {
	case "authentication_failure", "user_locked_out", "plugin_install_failed", "subtitle_download_failed":
		return "error"
	case "media_deleted", "plugin_install_cancelled", "server_restart_required", "user_deleted", "user_password_changed":
		return "warning"
	case "media_added", "authentication_success", "plugin_installed", "plugin_updated", "user_created":
		return "success"
	default:
		return "info"
	}
}

func stringValue(values map[string]interface{}, key string) string {
	value, _ := values[key].(string)
	return value
}

func nonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

func safeURL(value string) string {
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return parsed.String()
}
