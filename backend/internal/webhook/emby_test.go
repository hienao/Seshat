package webhook

import "testing"

func TestEmbyProviderMapsAliasesAndNestedItem(t *testing.T) {
	provider := NewEmbyProvider()
	if _, ok := provider.(*embyProvider); !ok {
		t.Fatalf("Emby must use its own provider implementation, got %T", provider)
	}
	request := IncomingRequest{Body: []byte(`{"Event":"library.new","Item":{"Id":"item-1","Name":"Dune","Type":"Movie","ProductionYear":2021,"RunTimeTicks":93000000000},"ServerName":"Media"}`)}
	sourceType := provider.DetectType(request)
	if sourceType != "library.new" || provider.MapType(sourceType) != "media_added" {
		t.Fatalf("unexpected event mapping: source=%q mapped=%q", sourceType, provider.MapType(sourceType))
	}
	presentation := provider.Normalize("media_added", request)
	media := presentation.Data["media"].(map[string]interface{})
	if media["id"] != "item-1" || media["display_name"] != "Dune（2021）" {
		t.Fatalf("unexpected media presentation: %+v", media)
	}
}

func TestEmbyUsesTheRandomEndpointAsItsCredential(t *testing.T) {
	provider := NewEmbyProvider()
	if !provider.Verify("unused-secret", IncomingRequest{Body: []byte(`{"Event":"library.new"}`)}) {
		t.Fatal("Emby requests should be accepted after the random endpoint key has matched")
	}
	if provider.Definition().AuthMode != "endpoint_url" {
		t.Fatalf("auth mode = %q", provider.Definition().AuthMode)
	}
}

func TestEmbyProviderMapsPlaybackAndUserAliases(t *testing.T) {
	provider := NewEmbyProvider()
	tests := map[string]string{
		"media.play":                "playback_started",
		"playback.pause":            "playback_progress",
		"media.stop":                "playback_stopped",
		"user.authenticationfailed": "authentication_failure",
		"UserPolicyUpdated":         "user_updated",
	}
	for source, expected := range tests {
		if actual := provider.MapType(source); actual != expected {
			t.Errorf("MapType(%q) = %q, want %q", source, actual, expected)
		}
	}
}

func TestEmbyUnknownEventUsesItsRawPresentation(t *testing.T) {
	provider := NewEmbyProvider()
	request := IncomingRequest{Body: []byte(`{"Event":"custom.event","value":42}`)}
	if mapped := provider.MapType(provider.DetectType(request)); mapped != "custom.event" {
		t.Fatalf("unknown event must remain unmapped, got %q", mapped)
	}
	presentation := provider.Normalize(DefaultEventType, request)
	if presentation.Data["category"] != "raw" || presentation.Data["raw_preview"] == "" {
		t.Fatalf("unexpected raw presentation: %+v", presentation.Data)
	}
}

func TestEmbyNormalizesEpisodeSessionAndVideoStream(t *testing.T) {
	provider := NewEmbyProvider()
	request := IncomingRequest{Body: []byte(`{"Event":"playback.start","User":{"Name":"mark"},"Session":{"Client":"Emby for Android TV","DeviceName":"客厅电视"},"Item":{"Id":"episode-1","Name":"Pilot","Type":"Episode","SeriesName":"Severance","ParentIndexNumber":1,"IndexNumber":1,"Overview":"团队发现了新的线索。","ProviderIds":{"Tmdb":"95396","Tvdb":"7889611","Imdb":"tt11280740"},"MediaStreams":[{"Type":"Video","DisplayTitle":"4K HEVC HDR","Codec":"hevc","Width":3840,"Height":2160,"VideoRange":"HDR"}]}}`)}
	presentation := provider.Normalize("playback_started", request)
	media := presentation.Data["media"].(map[string]interface{})
	actor := presentation.Data["actor"].(map[string]interface{})
	video := media["video"].(map[string]interface{})
	if media["display_name"] != "Severance · S01E01 · Pilot" || media["overview"] != "团队发现了新的线索。" {
		t.Fatalf("unexpected media: %+v", media)
	}
	if actor["client"] != "Emby for Android TV" || actor["device"] != "客厅电视" {
		t.Fatalf("unexpected actor: %+v", actor)
	}
	if video["display_label"] != "4K HEVC HDR" || video["width"] != "3840" || video["height"] != "2160" {
		t.Fatalf("unexpected video: %+v", video)
	}
}
