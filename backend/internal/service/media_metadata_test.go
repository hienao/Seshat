package service

import (
	"errors"
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
