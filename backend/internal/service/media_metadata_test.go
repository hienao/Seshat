package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
	token := "test-read-access-token"
	if err := settings.UpdateSystemSettings(&UpdateSystemSettingsRequest{TMDBReadAccessToken: &token}); err != nil {
		t.Fatal(err)
	}

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path != "/find/tt11280740" || request.URL.Query().Get("external_source") != "imdb_id" || request.URL.Query().Get("language") != "zh-CN" {
			t.Errorf("unexpected TMDB request: %s", request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("authorization header = %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"tv_episode_results":[{"id":1452857,"name":"Pilot","overview":"TMDB 剧集简介","still_path":"/episode.jpg","media_type":"tv_episode","show_id":95396,"season_number":1,"episode_number":1}]}`))
	}))

	metadataService := &MediaMetadataService{apiBaseURL: server.URL, httpClient: server.Client(), settings: settings}
	presentation := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{
		"type": "Episode", "provider_ids": map[string]interface{}{"imdb": "tt11280740", "tmdb": "1452857"},
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
	if cached.CacheKey != "imdb:tt11280740:episode" || cached.ExpiresAt.Before(time.Now().Add(6*24*time.Hour)) {
		t.Fatalf("unexpected cache: %+v", cached)
	}

	server.Close()
	second := webhook.Presentation{Data: map[string]interface{}{"media": map[string]interface{}{
		"type": "Episode", "provider_ids": map[string]interface{}{"imdb": "tt11280740"},
	}}}
	if err := metadataService.Enrich(&second); err != nil {
		t.Fatalf("cached enrichment failed after TMDB became unavailable: %v", err)
	}
	if requests != 1 {
		t.Fatalf("TMDB request count = %d, want 1", requests)
	}
}
