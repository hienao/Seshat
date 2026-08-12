package webhook

import "testing"

func TestJellyfinProviderDetectsMapsAndNormalizesEvents(t *testing.T) {
	provider := NewJellyfinProvider()
	if _, ok := provider.(*jellyfinProvider); !ok {
		t.Fatalf("Jellyfin must use its own provider implementation, got %T", provider)
	}
	known := IncomingRequest{Body: []byte(`{"NotificationType":"ItemAdded","Name":"Arrival","ItemType":"Movie","ProductionYear":2016,"RunTimeTicks":69600000000}`)}
	sourceType := provider.DetectType(known)
	if sourceType != "ItemAdded" {
		t.Fatalf("source event type = %q", sourceType)
	}
	if eventType := provider.MapType(sourceType); eventType != "media_added" {
		t.Fatalf("mapped event type = %q", eventType)
	}
	presentation := provider.Normalize("media_added", known)
	if presentation.Title != "Jellyfin · 新增媒体 · Arrival（2016）" || presentation.Severity != "success" {
		t.Fatalf("unexpected presentation: %+v", presentation)
	}
	if presentation.Data["category"] != "media" || presentation.Data["source_event_type"] != "ItemAdded" {
		t.Fatalf("unexpected presentation data: %+v", presentation.Data)
	}
	unknown := IncomingRequest{Body: []byte(`{"NotificationType":"PluginCustomEvent"}`)}
	if eventType := provider.DetectType(unknown); eventType != "PluginCustomEvent" {
		t.Fatalf("unknown source event type = %q", eventType)
	}
	if eventType := provider.MapType("PluginCustomEvent"); eventType != "PluginCustomEvent" {
		t.Fatalf("unknown mapped event type = %q", eventType)
	}
	definition := provider.Definition()
	if definition.DefaultEventType != DefaultEventType {
		t.Fatalf("default event type = %q", definition.DefaultEventType)
	}
	if len(definition.EventTypes) != 25 {
		t.Fatalf("event type count = %d, want 25", len(definition.EventTypes))
	}
}

func TestJellyfinPlaybackPresentation(t *testing.T) {
	provider := NewJellyfinProvider()
	request := IncomingRequest{Body: []byte(`{"NotificationType":"PlaybackProgress","Name":"Pilot","SeriesName":"Severance","SeriesPremiereDate":"2022-02-18","SeasonNumber00":"01","EpisodeNumber00":"01","ItemType":"Episode","Overview":"团队发现了新的线索。","RunTimeTicks":36000000000,"PlaybackPositionTicks":9000000000,"NotificationUsername":"mark","ClientName":"Jellyfin Web","DeviceName":"Living Room","PlayMethod":"Transcode","IsPaused":true,"Video_0_Title":"1080p H264 SDR","Provider_tmdb":"95396","Provider_tvdb":"7889611","Provider_imdb":"tt11280740"}`)}
	presentation := provider.Normalize("playback_progress", request)
	playback := presentation.Data["playback"].(map[string]interface{})
	if playback["percent"] != float64(25) || playback["paused"] != true {
		t.Fatalf("unexpected playback: %+v", playback)
	}
	if presentation.Data["category"] != "playback" {
		t.Fatalf("category = %v", presentation.Data["category"])
	}
	media := presentation.Data["media"].(map[string]interface{})
	actor := presentation.Data["actor"].(map[string]interface{})
	video := media["video"].(map[string]interface{})
	providers := media["provider_ids"].(map[string]interface{})
	if media["display_name"] != "Severance · S01E01 · Pilot" || media["overview"] != "团队发现了新的线索。" || media["series_year"] != "2022" {
		t.Fatalf("unexpected media: %+v", media)
	}
	if actor["client"] != "Jellyfin Web" || actor["device"] != "Living Room" || video["display_label"] != "1080p H264 SDR" {
		t.Fatalf("unexpected playback details: actor=%+v video=%+v", actor, video)
	}
	if providers["tmdb"] != "95396" || providers["tvdb"] != "7889611" || providers["imdb"] != "tt11280740" {
		t.Fatalf("unexpected provider IDs: %+v", providers)
	}
}

func TestJellyfinItemAddedEpisodePayloadKeepsMediaLookupIdentity(t *testing.T) {
	provider := NewJellyfinProvider()
	request := IncomingRequest{Body: []byte(`{"ServerId":"71fb2932097a4c21bfb1ed685db098e6","ServerName":"Hienao的影院","NotificationType":"ItemAdded","Name":"苟师猛踹老流氓","Overview":"廖师傅欺软怕硬。","ItemId":"05717b8bfec2c871b7d37f2644e84f97","ItemType":"Episode","Year":2026,"SeriesName":"主角","SeriesId":"a049c405030967f7ad186708d0793d87","SeriesPremiereDate":"2026-05-10","SeasonNumber":1,"EpisodeNumber":15}`)}
	presentation := provider.Normalize("media_added", request)
	media := presentation.Data["media"].(map[string]interface{})
	if media["id"] != "05717b8bfec2c871b7d37f2644e84f97" || media["series"] != "主角" || media["season"] != "1" || media["episode"] != "15" {
		t.Fatalf("unexpected ItemAdded lookup identity: %+v", media)
	}
	if media["name"] != "苟师猛踹老流氓" || media["overview"] != "廖师傅欺软怕硬。" || media["display_name"] != "主角 · S01E15 · 苟师猛踹老流氓" {
		t.Fatalf("unexpected ItemAdded presentation: %+v", media)
	}
}

func TestJellyfinMediaPresentationDoesNotRepeatMediaType(t *testing.T) {
	provider := NewJellyfinProvider()
	request := IncomingRequest{Body: []byte(`{"NotificationType":"ItemAdded","Name":"我的团长我的团 · S01E04 · 团长龙文章现身带领众人","ItemType":"Episode","Overview":"自称团长的家伙把他们带出了板房。"}`)}
	presentation := provider.Normalize("media_added", request)
	if presentation.Summary != "收到 Jellyfin 新增媒体事件" {
		t.Fatalf("summary = %q", presentation.Summary)
	}
	if len(presentation.Tags) != 0 {
		t.Fatalf("media type must not be repeated as tags: %+v", presentation.Tags)
	}
	if len(presentation.Facts) != 1 || presentation.Facts[0]["label_key"] != "events.facts.media_type" || presentation.Facts[0]["label"] != "类型" || presentation.Facts[0]["value"] != "Episode" {
		t.Fatalf("media type must remain as one fact: %+v", presentation.Facts)
	}
}
