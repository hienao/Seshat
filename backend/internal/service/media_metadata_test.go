package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTMDBConnectionUsesCurrentAPIKeyAndProxyWithoutSaving(t *testing.T) {
	apiKey := "current-test-api-key"
	proxyRequests := 0
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxyRequests++
		if request.URL.Host != "tmdb.example" || request.URL.Path != "/3/configuration" {
			t.Errorf("unexpected TMDB test request: %s", request.URL.String())
		}
		if request.URL.Query().Get("api_key") != apiKey {
			t.Errorf("api_key query = %q", request.URL.Query().Get("api_key"))
		}
		if request.Header.Get("Authorization") != "" {
			t.Errorf("authorization header must be empty, got %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"images":{}}`))
	}))
	defer proxy.Close()

	metadataService := &MediaMetadataService{apiBaseURL: "http://tmdb.example/3", httpClient: &http.Client{Timeout: time.Second}, settings: NewSettingService()}
	if err := metadataService.TestConnection(&TestTMDBConnectionRequest{TMDBAPIKey: apiKey, HTTPProxyURL: proxy.URL, UseProxy: true}); err != nil {
		t.Fatal(err)
	}
	if proxyRequests != 1 {
		t.Fatalf("proxy requests = %d, want 1", proxyRequests)
	}
	if err := metadataService.TestConnection(&TestTMDBConnectionRequest{TMDBAPIKey: apiKey, HTTPProxyURL: "socks5://127.0.0.1:1080", UseProxy: true}); !errors.Is(err, ErrInvalidHTTPProxyURL) {
		t.Fatalf("invalid proxy error = %v", err)
	}
}

func TestHTTPProxyConnectionDoesNotRequireTMDBAPIKey(t *testing.T) {
	proxyRequests := 0
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxyRequests++
		if request.URL.Host != "probe.example" || request.URL.Path != "/health" {
			t.Errorf("unexpected proxy test request: %s", request.URL.String())
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()

	metadataService := &MediaMetadataService{proxyTestURL: "http://probe.example/health", httpClient: &http.Client{Timeout: time.Second}, settings: NewSettingService()}
	if err := metadataService.TestHTTPProxy(&TestHTTPProxyRequest{HTTPProxyURL: proxy.URL}); err != nil {
		t.Fatal(err)
	}
	if proxyRequests != 1 {
		t.Fatalf("proxy requests = %d, want 1", proxyRequests)
	}
	if err := metadataService.TestHTTPProxy(&TestHTTPProxyRequest{HTTPProxyURL: "socks5://127.0.0.1:1080"}); !errors.Is(err, ErrInvalidHTTPProxyURL) {
		t.Fatalf("invalid proxy error = %v", err)
	}
}

func TestTMDBConnectionRejectsInvalidAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	metadataService := &MediaMetadataService{apiBaseURL: server.URL, httpClient: server.Client(), settings: NewSettingService()}
	if err := metadataService.TestConnection(&TestTMDBConnectionRequest{TMDBAPIKey: "invalid"}); !errors.Is(err, ErrInvalidTMDBAPIKey) {
		t.Fatalf("invalid API key error = %v", err)
	}
}

func setupMediaMetadataTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.MediaMetadataCache{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = nil })
}

func TestMediaMetadataEnrichmentUsesTMDBAndPersistentCache(t *testing.T) {
	setupMediaMetadataTestDB(t)
	settings := NewSettingService()
	if err := settings.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	apiKey := "test-api-key"
	if err := settings.UpdateSystemSettings(&UpdateSystemSettingsRequest{TMDBAPIKey: &apiKey}); err != nil {
		t.Fatal(err)
	}

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Query().Get("api_key") != apiKey {
			t.Errorf("api_key query = %q", request.URL.Query().Get("api_key"))
		}
		if request.Header.Get("Authorization") != "" {
			t.Errorf("authorization header must be empty, got %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/search/tv":
			if request.URL.Query().Get("query") != "Severance" || request.URL.Query().Get("first_air_date_year") != "2022" || request.URL.Query().Get("language") != "zh-CN" {
				t.Errorf("unexpected TMDB search: %s", request.URL.String())
			}
			_, _ = writer.Write([]byte(`{"results":[{"id":95396,"name":"Severance"}]}`))
		case "/tv/95396/season/1/episode/1":
			_, _ = writer.Write([]byte(`{"id":1452857,"name":"Pilot","overview":"TMDB 剧集简介","still_path":"/episode.jpg"}`))
		default:
			t.Errorf("unexpected TMDB request: %s", request.URL.String())
			writer.WriteHeader(http.StatusNotFound)
		}
	}))

	metadataService := &MediaMetadataService{apiBaseURL: server.URL, httpClient: server.Client(), settings: settings}
	presentation := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{
		"type": "Episode", "series": "Severance", "series_year": "2022", "season": "01", "episode": "01", "provider_ids": map[string]interface{}{"imdb": "tt11280740", "tvdb": "7889611", "tmdb": "1452857"},
	}}}
	if err := metadataService.Enrich(&presentation); err != nil {
		t.Fatal(err)
	}
	media := presentation.Data["media"].(map[string]interface{})
	if media["overview"] != "TMDB 剧集简介" || media["image_url"] != "https://image.tmdb.org/t/p/w342/episode.jpg" || media["metadata_source"] != "tmdb" {
		t.Fatalf("unexpected enriched media: %+v", media)
	}
	if len(presentation.Links) != 1 || presentation.Links[0]["url"] != "https://www.themoviedb.org/tv/95396/season/1/episode/1" {
		t.Fatalf("unexpected metadata links: %+v", presentation.Links)
	}
	var cached model.MediaMetadataCache
	if err := database.GetDB().First(&cached).Error; err != nil {
		t.Fatal(err)
	}
	if cached.CacheKey != "tmdb:episode:1452857" || cached.Provider != "tmdb" || cached.ExpiresAt.Before(time.Now().Add(6*24*time.Hour)) {
		t.Fatalf("unexpected cache: %+v", cached)
	}

	server.Close()
	second := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{
		"type": "Episode", "series": "Severance", "series_year": "2022", "season": "1", "episode": "1", "provider_ids": map[string]interface{}{"tmdb": "1452857"},
	}}}
	if err := metadataService.Enrich(&second); err != nil {
		t.Fatalf("cached enrichment failed after TMDB became unavailable: %v", err)
	}
	if requests != 2 {
		t.Fatalf("TMDB request count = %d, want 2", requests)
	}
}

func TestEpisodeMetadataUsesOnlyTMDBProviderID(t *testing.T) {
	setupMediaMetadataTestDB(t)
	settings := NewSettingService()
	if err := settings.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	apiKey := "test-api-key"
	if err := settings.UpdateSystemSettings(&UpdateSystemSettingsRequest{TMDBAPIKey: &apiKey}); err != nil {
		t.Fatal(err)
	}

	requestedPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPaths = append(requestedPaths, request.URL.Path)
		if request.URL.Query().Get("external_source") != "" {
			t.Errorf("external provider lookup must not be used: %s", request.URL.String())
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/search/tv":
			if request.URL.Query().Get("query") != "擅长捉弄的高木同学" || request.URL.Query().Get("first_air_date_year") != "2018" {
				t.Errorf("unexpected TMDB search: %s", request.URL.String())
			}
			_, _ = writer.Write([]byte(`{"results":[{"id":75865,"name":"擅长捉弄的高木同学"}]}`))
		case "/tv/75865/season/1/episode/10":
			_, _ = writer.Write([]byte(`{"id":1452857,"name":"Episode 10","overview":"TMDB matched","still_path":"/takagi-episode.jpg"}`))
		default:
			t.Errorf("unexpected TMDB request: %s", request.URL.String())
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	metadataService := &MediaMetadataService{apiBaseURL: server.URL, httpClient: server.Client(), settings: settings}
	presentation := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{
		"type": "Episode", "series": "擅长捉弄的高木同学", "series_year": "2018", "season": "1", "episode": "10", "provider_ids": map[string]interface{}{"tmdb": "1452857", "tvdb": "6493171", "imdb": "tt8150114"},
	}}}
	if err := metadataService.Enrich(&presentation); err != nil {
		t.Fatal(err)
	}
	media := presentation.Data["media"].(map[string]interface{})
	if strings.Join(requestedPaths, ",") != "/search/tv,/tv/75865/season/1/episode/10" || media["image_url"] != "https://image.tmdb.org/t/p/w342/takagi-episode.jpg" {
		t.Fatalf("TMDB-only episode enrichment failed: paths=%v media=%+v", requestedPaths, media)
	}
}

func TestMediaMetadataDoesNotCreateIMDbOrTVDBLookups(t *testing.T) {
	media := map[string]interface{}{
		"type": "Episode", "series": "Example", "season": "1", "episode": "1",
		"provider_ids": map[string]interface{}{"tvdb": "6493171", "imdb": "tt8150114"},
	}
	if lookup := metadataLookupForMedia(media); lookup != nil {
		t.Fatalf("non-TMDB IDs must not create a metadata lookup: %+v", lookup)
	}
}

func TestMediaMetadataCanUseTheSystemHTTPProxy(t *testing.T) {
	setupMediaMetadataTestDB(t)
	settings := NewSettingService()
	if err := settings.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	apiKey := "test-api-key"
	useProxy := true
	proxyRequests := 0
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxyRequests++
		if request.URL.Host != "tmdb.example" || request.URL.Path != "/3/movie/42" {
			t.Errorf("unexpected proxied TMDB request: %s", request.URL.String())
		}
		if request.URL.Query().Get("api_key") != apiKey {
			t.Errorf("api_key query = %q", request.URL.Query().Get("api_key"))
		}
		if request.Header.Get("Authorization") != "" {
			t.Errorf("authorization header must be empty, got %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":42,"title":"Arrival","overview":"TMDB 电影简介","poster_path":"/arrival.jpg","media_type":"movie"}`))
	}))
	defer proxy.Close()
	proxyURL := proxy.URL
	if err := settings.UpdateSystemSettings(&UpdateSystemSettingsRequest{TMDBAPIKey: &apiKey, HTTPProxyURL: &proxyURL, TMDBUseProxy: &useProxy}); err != nil {
		t.Fatal(err)
	}
	metadataService := &MediaMetadataService{apiBaseURL: "http://tmdb.example/3", httpClient: &http.Client{Timeout: time.Second}, settings: settings}
	presentation := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{
		"type": "Movie", "provider_ids": map[string]interface{}{"tmdb": "42"},
	}}}
	if err := metadataService.Enrich(&presentation); err != nil {
		t.Fatal(err)
	}
	media := presentation.Data["media"].(map[string]interface{})
	if proxyRequests != 1 || media["overview"] != "TMDB 电影简介" {
		t.Fatalf("TMDB proxy was not used: requests=%d media=%+v", proxyRequests, media)
	}
}

func TestJellyfinMetadataTakesPriorityAndCachesProtectedImage(t *testing.T) {
	setupMediaMetadataTestDB(t)
	settingsService := NewSettingService()
	if err := settingsService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	tmdbKey := "tmdb-key"
	if err := settingsService.UpdateSystemSettings(&UpdateSystemSettingsRequest{TMDBAPIKey: &tmdbKey}); err != nil {
		t.Fatal(err)
	}
	mediaRequests, tmdbRequests := 0, 0
	mediaTitle := "服务端标题"
	mediaServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mediaRequests++
		if request.Header.Get("X-Emby-Token") != "jellyfin-key" {
			t.Errorf("media API key header = %q", request.Header.Get("X-Emby-Token"))
		}
		switch request.URL.Path {
		case "/Items/item-10":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(fmt.Sprintf(`{"Id":"item-10","Name":%q,"Type":"Episode","SeriesName":"服务端剧集","ParentIndexNumber":2,"IndexNumber":3,"ProductionYear":2026,"Overview":"Jellyfin 简介","RunTimeTicks":6000000000,"ProviderIds":{"Tmdb":"88","Tvdb":"99"},"ImageTags":{"Primary":"tag"}}`, mediaTitle)))
		case "/Items/item-10/Images/Primary":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write([]byte("png-image-data"))
		default:
			t.Errorf("unexpected Jellyfin request: %s", request.URL.String())
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mediaServer.Close()
	tmdbServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		tmdbRequests++
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer tmdbServer.Close()

	metadataService := NewMediaMetadataService()
	metadataService.settings = settingsService
	metadataService.apiBaseURL = tmdbServer.URL
	metadataService.httpClient = tmdbServer.Client()
	metadataService.adapters["jellyfin"].(*jellyfinMediaMetadataAdapter).client = mediaServer.Client()
	config, _ := json.Marshal(map[string]string{"media_server_url": mediaServer.URL})
	secretConfig, _ := json.Marshal(map[string]string{"media_api_key": "jellyfin-key"})
	integration := &model.AppIntegration{ID: 12, AppCode: "jellyfin", Secret: "integration-secret", Config: config, SecretConfig: secretConfig}
	presentation := webhook.Presentation{Title: "Jellyfin · 播放进度 · 旧标题", Data: map[string]interface{}{
		"category": "playback", "event_label": "播放进度", "media": map[string]interface{}{"id": "item-10", "type": "Episode", "name": "旧标题", "provider_ids": map[string]interface{}{"tmdb": "1452857"}},
	}}
	if err := metadataService.Enrich(&presentation, integration); err != nil {
		t.Fatal(err)
	}
	media := presentation.Data["media"].(map[string]interface{})
	if media["name"] != "服务端标题" || media["series"] != "服务端剧集" || media["overview"] != "Jellyfin 简介" || media["metadata_source"] != "jellyfin" {
		t.Fatalf("Jellyfin metadata did not take priority: %+v", media)
	}
	imageURL, _ := media["image_url"].(string)
	if !strings.HasPrefix(imageURL, "/api/public/media-images/") {
		t.Fatalf("protected image was not cached: %q", imageURL)
	}
	token := strings.TrimPrefix(imageURL, "/api/public/media-images/")
	image, err := metadataService.CachedImage(token)
	if err != nil || string(image.ImageData) != "png-image-data" || image.ImageType != "image/png" {
		t.Fatalf("cached image = %+v, error = %v", image, err)
	}
	if tmdbRequests != 0 || mediaRequests != 2 {
		t.Fatalf("requests media=%d tmdb=%d, want media=2 tmdb=0", mediaRequests, tmdbRequests)
	}
	if presentation.Title != "Jellyfin · 播放进度 · 服务端剧集 · S02E03 · 服务端标题" {
		t.Fatalf("presentation title was not refreshed: %q", presentation.Title)
	}
	cachedPresentation := webhook.Presentation{Title: "Jellyfin · 播放进度 · Webhook 标题", Data: map[string]interface{}{
		"category": "playback", "event_label": "播放进度", "media": map[string]interface{}{"id": "item-10", "type": "Episode", "name": "Webhook 标题"},
	}}
	if err := metadataService.Enrich(&cachedPresentation, integration); err != nil {
		t.Fatal(err)
	}
	if mediaRequests != 2 {
		t.Fatalf("fresh media cache made %d requests, want 2", mediaRequests)
	}
	if err := database.GetDB().Model(&model.MediaMetadataCache{}).Where("provider = ? AND external_id = ?", "jellyfin", "item-10").Update("updated_at", time.Now().Add(-2*mediaServerFreshness)).Error; err != nil {
		t.Fatal(err)
	}
	mediaTitle = "刮削完成后的标题"
	refreshedPresentation := webhook.Presentation{Title: "Jellyfin · 新增媒体 · Webhook 标题", Data: map[string]interface{}{
		"category": "media", "event_label": "新增媒体", "media": map[string]interface{}{"id": "item-10", "type": "Episode", "name": "Webhook 标题"},
	}}
	if err := metadataService.Enrich(&refreshedPresentation, integration); err != nil {
		t.Fatal(err)
	}
	refreshedMedia := refreshedPresentation.Data["media"].(map[string]interface{})
	if mediaRequests != 4 || refreshedMedia["name"] != mediaTitle {
		t.Fatalf("stale media cache was not refreshed: requests=%d media=%+v", mediaRequests, refreshedMedia)
	}
}

func TestMediaServerFailureFallsBackToTMDB(t *testing.T) {
	setupMediaMetadataTestDB(t)
	settingsService := NewSettingService()
	if err := settingsService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	tmdbKey := "tmdb-key"
	if err := settingsService.UpdateSystemSettings(&UpdateSystemSettingsRequest{TMDBAPIKey: &tmdbKey}); err != nil {
		t.Fatal(err)
	}
	mediaServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusBadGateway) }))
	defer mediaServer.Close()
	tmdbRequests := 0
	tmdbServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tmdbRequests++
		if request.URL.Path != "/movie/42" || request.URL.Query().Get("api_key") != tmdbKey {
			t.Errorf("unexpected TMDB fallback request: %s", request.URL.String())
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":42,"title":"TMDB title","overview":"TMDB fallback","poster_path":"/fallback.jpg","media_type":"movie"}`))
	}))
	defer tmdbServer.Close()
	metadataService := NewMediaMetadataService()
	metadataService.settings = settingsService
	metadataService.apiBaseURL = tmdbServer.URL
	metadataService.httpClient = tmdbServer.Client()
	metadataService.adapters["jellyfin"].(*jellyfinMediaMetadataAdapter).client = mediaServer.Client()
	config, _ := json.Marshal(map[string]string{"media_server_url": mediaServer.URL})
	secretConfig, _ := json.Marshal(map[string]string{"media_api_key": "jellyfin-key"})
	presentation := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{"id": "item-42", "type": "Movie", "provider_ids": map[string]interface{}{"tmdb": "42"}}}}
	if err := metadataService.Enrich(&presentation, &model.AppIntegration{ID: 3, AppCode: "jellyfin", Secret: "integration-secret", Config: config, SecretConfig: secretConfig}); err != nil {
		t.Fatal(err)
	}
	media := presentation.Data["media"].(map[string]interface{})
	if tmdbRequests != 1 || media["overview"] != "TMDB fallback" || media["metadata_source"] != "tmdb" {
		t.Fatalf("TMDB fallback failed: requests=%d media=%+v", tmdbRequests, media)
	}
}

func TestEmbyAdapterUsesEmbyPrefixAndHeaderAuthentication(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.URL.Path)
		if request.Header.Get("X-Emby-Token") != "emby-key" {
			t.Errorf("Emby API key header = %q", request.Header.Get("X-Emby-Token"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"Id":"emby-item","Name":"Emby item","Type":"Movie"}`))
	}))
	defer server.Close()
	adapter := &embyMediaMetadataAdapter{client: server.Client()}
	settings, err := normalizeMediaServerSettings(server.URL, "emby-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Test(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Fetch(context.Background(), settings, "emby-item"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(requests, ",") != "/emby/System/Info,/emby/Items/emby-item" {
		t.Fatalf("unexpected Emby paths: %v", requests)
	}
}
