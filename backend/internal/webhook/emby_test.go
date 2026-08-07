package webhook

import "testing"

func TestEmbyProviderMapsAliasesAndNestedItem(t *testing.T) {
	provider := NewEmbyProvider()
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
