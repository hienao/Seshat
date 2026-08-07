package webhook

import "testing"

func TestJellyfinProviderDetectsKnownAndUnknownEvents(t *testing.T) {
	provider := NewJellyfinProvider()
	known := IncomingRequest{Body: []byte(`{"NotificationType":"ItemAdded","Name":"Arrival","ItemType":"Movie"}`)}
	if eventType := provider.DetectType(known); eventType != "media_added" {
		t.Fatalf("known event type = %q", eventType)
	}
	presentation := provider.Normalize("media_added", known)
	if presentation.Title != "Jellyfin · 新增媒体 · Arrival" || len(presentation.Facts) != 2 {
		t.Fatalf("unexpected presentation: %+v", presentation)
	}
	unknown := IncomingRequest{Body: []byte(`{"NotificationType":"PluginCustomEvent"}`)}
	if eventType := provider.DetectType(unknown); eventType != "PluginCustomEvent" {
		t.Fatalf("unknown source event type = %q", eventType)
	}
	definition := provider.Definition()
	if definition.DefaultEventType != DefaultEventType {
		t.Fatalf("default event type = %q", definition.DefaultEventType)
	}
}
