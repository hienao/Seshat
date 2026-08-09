package webhook

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

func buildMediaPresentation(appName string, definitions []EventTypeDefinition, eventType, sourceType string, body []byte, payload, media, actor, playback, system map[string]interface{}) Presentation {
	category := eventCategory(eventType)
	label := eventTypeName(definitions, eventType)
	data := map[string]interface{}{"category": category, "event_label": label, "source_event_type": sourceType}
	if category == "raw" {
		data["raw_preview"] = bodyPreview(body, 600)
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

	title := appName + " · " + label
	if subject := presentationSubject(category, media, actor, system); subject != "" {
		title += " · " + subject
	}
	return Presentation{
		SchemaVersion: 1,
		Title:         title,
		Summary:       presentationSummary(appName, label, category, media, actor, playback, system),
		Severity:      eventSeverity(eventType, system),
		Tags:          presentationTags(media, playback),
		Facts:         presentationFacts(category, media, actor, playback, system),
		Links:         presentationLinks(payload, media),
		Data:          data,
	}
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

func videoDisplayLabel(video map[string]interface{}) string {
	if title := stringValue(video, "title"); title != "" {
		return title
	}
	resolution := ""
	if width, height := stringValue(video, "width"), stringValue(video, "height"); width != "" && height != "" {
		resolution = width + "×" + height
	}
	return strings.Join(nonEmptyStrings(resolution, strings.ToUpper(stringValue(video, "codec")), stringValue(video, "range")), " · ")
}

func normalizePlaybackValues(values []map[string]interface{}, media map[string]interface{}) map[string]interface{} {
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

func normalizeSystemValues(eventType string, values []map[string]interface{}, payload map[string]interface{}) map[string]interface{} {
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
		if season != "" {
			index = formatMediaIndex("S", season)
		}
		if episode != "" {
			index += formatMediaIndex("E", episode)
		}
		return strings.Trim(strings.Join(nonEmptyStrings(series, index, name), " · "), " ·")
	}
	if name != "" && year != "" {
		return name + "（" + year + "）"
	}
	return name
}

func formatMediaIndex(prefix, value string) string {
	if number, err := strconv.Atoi(value); err == nil && number >= 0 {
		return fmt.Sprintf("%s%02d", prefix, number)
	}
	return prefix + value
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
		parts := nonEmptyStrings(stringValue(actor, "username"), stringValue(actor, "client"), stringValue(actor, "device"), stringValue(playback, "method"))
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
		if value != "" && len(facts) < 8 {
			facts = append(facts, map[string]string{"label": label, "value": value})
		}
	}
	if category == "media" || category == "playback" {
		addFact("类型", stringValue(media, "type"))
		addFact("用户", stringValue(actor, "username"))
		addFact("客户端", stringValue(actor, "client"))
		addFact("设备", stringValue(actor, "device"))
		addFact("播放方式", stringValue(playback, "method"))
		addFact("媒体信息", nestedStringValue(media, "video", "display_label"))
		addFact("进度", playbackProgressLabel(playback, media))
		addFact("外部 ID", providerIDsLabel(media))
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

func presentationLinks(payload, media map[string]interface{}) []map[string]string {
	links := []map[string]string{}
	if itemURL := safeURL(firstString(payload, "ItemUrl", "item_url")); itemURL != "" {
		links = append(links, map[string]string{"label": "打开媒体", "url": itemURL})
	}
	if serverURL := safeURL(firstString(payload, "ServerUrl", "server_url")); serverURL != "" {
		links = append(links, map[string]string{"label": "打开服务", "url": serverURL})
	}
	providers, _ := media["provider_ids"].(map[string]interface{})
	if imdb := stringValue(providers, "imdb"); validIMDbID(imdb) {
		links = append(links, map[string]string{"label": "IMDb", "url": "https://www.imdb.com/title/" + imdb})
	}
	if tvdb := stringValue(providers, "tvdb"); digitsOnly(tvdb) {
		links = append(links, map[string]string{"label": "TVDB", "url": "https://thetvdb.com/search?query=" + tvdb})
	}
	if tmdb := stringValue(providers, "tmdb"); digitsOnly(tmdb) {
		switch strings.ToLower(stringValue(media, "type")) {
		case "movie":
			links = append(links, map[string]string{"label": "TMDB", "url": "https://www.themoviedb.org/movie/" + tmdb})
		case "series":
			links = append(links, map[string]string{"label": "TMDB", "url": "https://www.themoviedb.org/tv/" + tmdb})
		}
	}
	return links
}

func nestedStringValue(values map[string]interface{}, container, key string) string {
	nested, _ := values[container].(map[string]interface{})
	return stringValue(nested, key)
}

func providerIDsLabel(media map[string]interface{}) string {
	providers, _ := media["provider_ids"].(map[string]interface{})
	parts := []string{}
	for _, provider := range []string{"tmdb", "tvdb", "imdb"} {
		if value := stringValue(providers, provider); value != "" {
			parts = append(parts, strings.ToUpper(provider)+": "+value)
		}
	}
	return strings.Join(parts, " · ")
}

func validIMDbID(value string) bool {
	return strings.HasPrefix(value, "tt") && len(value) > 2 && digitsOnly(value[2:])
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
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

func eventTypeName(definitions []EventTypeDefinition, eventType string) string {
	for _, definition := range definitions {
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
