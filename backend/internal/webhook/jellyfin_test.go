package webhook

import "testing"

func TestJellyfinProviderDetectsMapsAndNormalizesEvents(t *testing.T) {
	provider := NewJellyfinProvider()
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
	request := IncomingRequest{Body: []byte(`{"NotificationType":"PlaybackProgress","Name":"Pilot","SeriesName":"Severance","SeasonNumber":1,"EpisodeNumber":1,"ItemType":"Episode","RunTimeTicks":36000000000,"PlaybackPositionTicks":9000000000,"NotificationUsername":"mark","DeviceName":"Living Room","PlayMethod":"Transcode","IsPaused":true}`)}
	presentation := provider.Normalize("playback_progress", request)
	playback := presentation.Data["playback"].(map[string]interface{})
	if playback["percent"] != float64(25) || playback["paused"] != true {
		t.Fatalf("unexpected playback: %+v", playback)
	}
	if presentation.Data["category"] != "playback" {
		t.Fatalf("category = %v", presentation.Data["category"])
	}
}
