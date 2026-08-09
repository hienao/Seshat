package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"seshat/internal/model"
	"seshat/pkg/database"
)

func githubSignature(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func setupWebhookTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:webhook-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	if err := db.AutoMigrate(&model.AppIntegration{}, &model.WebhookEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.WebhookEvent{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.AppIntegration{}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.DB = nil })
}

func TestWebhookUsesAppDefaultEventType(t *testing.T) {
	setupWebhookTestDB(t)
	service := NewWebhookService()
	created, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "generic", Name: "测试接入"})
	if err != nil {
		t.Fatal(err)
	}
	var integration model.AppIntegration
	if err := database.GetDB().First(&integration, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if integration.Secret != created.Secret {
		t.Fatal("webhook secret was not stored as plaintext")
	}
	integrationJSON, err := json.Marshal(integration)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(integrationJSON, []byte(integration.Secret)) {
		t.Fatal("webhook secret was exposed by integration JSON")
	}
	body := []byte(`{"event_type":"not-registered","message":"hello"}`)
	result, err := service.Ingest(created.EndpointKey, map[string]string{"X-Webhook-Secret": created.Secret}, body, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var event model.WebhookEvent
	if err := database.GetDB().First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.SourceEventType != "not-registered" {
		t.Fatalf("source type = %q", event.SourceEventType)
	}
	if event.DisplayEventType != "__default__" || !event.IsFallback {
		t.Fatalf("expected default fallback, got type=%q fallback=%v", event.DisplayEventType, event.IsFallback)
	}
	if event.RawBody != string(body) {
		t.Fatal("raw body was not preserved")
	}
	if len(event.PublicToken) < 32 {
		t.Fatal("public event token was not generated")
	}
	if result.AppCode != "generic" || result.IntegrationID != integration.ID || result.EventID != event.ID {
		t.Fatalf("unexpected ingest result: %+v", result)
	}
	var presentation map[string]interface{}
	if err := json.Unmarshal(event.Presentation, &presentation); err != nil {
		t.Fatal(err)
	}
	if presentation["schema_version"] != float64(1) {
		t.Fatal("missing presentation schema version")
	}
	publicEvent, err := service.GetPublicEvent(event.PublicToken)
	if err != nil {
		t.Fatal(err)
	}
	if publicEvent.Presentation.Data["raw_preview"] != nil {
		t.Fatal("public event exposed raw preview")
	}
	publicJSON, _ := json.Marshal(publicEvent)
	if bytes.Contains(publicJSON, []byte("hello")) || bytes.Contains(publicJSON, []byte(event.PublicToken)) || bytes.Contains(publicJSON, []byte(`"integration_id"`)) {
		t.Fatalf("public event exposed private data: %s", publicJSON)
	}
}

func TestPublicEventRejectsInvalidTokens(t *testing.T) {
	setupWebhookTestDB(t)
	if _, err := NewWebhookService().GetPublicEvent("1"); !errors.Is(err, ErrPublicEventNotFound) {
		t.Fatalf("error = %v, want ErrPublicEventNotFound", err)
	}
}

func TestEnsurePublicEventTokenSupportsEventsCreatedBeforeMigration(t *testing.T) {
	setupWebhookTestDB(t)
	integration := model.AppIntegration{OwnerID: 7, AppCode: "generic", Name: "Generic", EndpointKey: "legacy-event", Secret: "secret", Enabled: true}
	if err := database.GetDB().Create(&integration).Error; err != nil {
		t.Fatal(err)
	}
	event := model.WebhookEvent{IntegrationID: integration.ID, AppCode: "generic", DisplayEventType: "__default__", DedupeKey: "legacy", Status: "processed", PresentationVersion: 1, Presentation: []byte(`{"schema_version":1,"title":"Legacy","severity":"info","data":{"category":"raw","raw_preview":"private"}}`), ReceivedAt: time.Now()}
	if err := database.GetDB().Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensurePublicEventToken(&event); err != nil {
		t.Fatal(err)
	}
	if len(event.PublicToken) < 32 {
		t.Fatal("legacy event did not receive a public token")
	}
	publicEvent, err := NewWebhookService().GetPublicEvent(event.PublicToken)
	if err != nil || publicEvent.Presentation.Data["raw_preview"] != nil {
		t.Fatalf("legacy public event = %+v, error = %v", publicEvent, err)
	}
}

func TestIntegrationSecretCanOnlyBeReadByOwner(t *testing.T) {
	setupWebhookTestDB(t)
	service := NewWebhookService()
	created, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "jellyfin", Name: "Jellyfin"})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := service.GetIntegrationSecret(7, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if secret.Secret != created.Secret {
		t.Fatalf("secret = %q, want created secret", secret.Secret)
	}
	if _, err := service.GetIntegrationSecret(8, created.ID); err == nil {
		t.Fatal("another owner should not be able to read the secret")
	}
}

func TestIntegrationMediaSettingsAreRequiredReturnedAndKeptOutOfIntegrationJSON(t *testing.T) {
	setupWebhookTestDB(t)
	service := NewWebhookService()
	created, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "jellyfin", Name: "家庭影院"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateIntegrationMediaSettings(7, created.ID, &UpdateIntegrationMediaSettingsRequest{ServerURL: "", APIKey: "key"}); !errors.Is(err, ErrInvalidMediaAPIURL) {
		t.Fatalf("empty server URL error = %v", err)
	}
	if _, err := service.UpdateIntegrationMediaSettings(7, created.ID, &UpdateIntegrationMediaSettingsRequest{ServerURL: "http://jellyfin.example:8096", APIKey: ""}); !errors.Is(err, ErrInvalidMediaAPIKey) {
		t.Fatalf("empty API key error = %v", err)
	}
	updated, err := service.UpdateIntegrationMediaSettings(7, created.ID, &UpdateIntegrationMediaSettingsRequest{ServerURL: "http://jellyfin.example:8096/", APIKey: "media-secret-key"})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Configured || updated.ServerURL != "http://jellyfin.example:8096" || updated.APIKey != "media-secret-key" {
		t.Fatalf("unexpected updated settings: %+v", updated)
	}
	loaded, err := service.GetIntegrationMediaSettings(7, created.ID)
	if err != nil || loaded.APIKey != "media-secret-key" {
		t.Fatalf("loaded settings = %+v, error = %v", loaded, err)
	}
	if _, err := service.GetIntegrationMediaSettings(8, created.ID); err == nil {
		t.Fatal("another owner should not be able to read media API settings")
	}
	items, err := service.ListIntegrations(7)
	if err != nil || len(items) != 1 || !items[0].MediaAPIConfigured {
		t.Fatalf("integration response did not report configured media API: %+v, %v", items, err)
	}
	encoded, _ := json.Marshal(items)
	if bytes.Contains(encoded, []byte("media-secret-key")) {
		t.Fatalf("media API key was exposed by integration list: %s", encoded)
	}
	if err := service.TestIntegrationMediaSettings(context.Background(), 7, created.ID, &UpdateIntegrationMediaSettingsRequest{ServerURL: "https://user:password@example.com", APIKey: "key"}); !errors.Is(err, ErrInvalidMediaAPIURL) {
		t.Fatalf("credential-bearing server URL error = %v", err)
	}

	generic, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "generic", Name: "Generic"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetIntegrationMediaSettings(7, generic.ID); !errors.Is(err, ErrMediaAPIUnsupported) {
		t.Fatalf("generic media settings error = %v", err)
	}
}

func TestListEventsFiltersByIntegrationAndPaginates(t *testing.T) {
	setupWebhookTestDB(t)
	webhookService := NewWebhookService()
	first, err := webhookService.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "jellyfin", Name: "客厅影院"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := webhookService.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "jellyfin", Name: "卧室影院"})
	if err != nil {
		t.Fatal(err)
	}
	otherOwner, err := webhookService.CreateIntegration(8, &CreateIntegrationRequest{AppCode: "jellyfin", Name: "其他用户"})
	if err != nil {
		t.Fatal(err)
	}
	baseTime := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	events := []model.WebhookEvent{
		{IntegrationID: first.ID, AppCode: "jellyfin", DisplayEventType: "playback_progress", DedupeKey: "first-progress-new", Status: "processed", PresentationVersion: 1, Presentation: []byte(`{}`), ReceivedAt: baseTime.Add(2 * time.Minute)},
		{IntegrationID: first.ID, AppCode: "jellyfin", DisplayEventType: "playback_started", DedupeKey: "first-started", Status: "processed", PresentationVersion: 1, Presentation: []byte(`{}`), ReceivedAt: baseTime.Add(time.Minute)},
		{IntegrationID: first.ID, AppCode: "jellyfin", DisplayEventType: "playback_progress", DedupeKey: "first-progress-old", Status: "processed", PresentationVersion: 1, Presentation: []byte(`{}`), ReceivedAt: baseTime},
		{IntegrationID: second.ID, AppCode: "jellyfin", DisplayEventType: "playback_progress", DedupeKey: "second-progress", Status: "processed", PresentationVersion: 1, Presentation: []byte(`{}`), ReceivedAt: baseTime.Add(3 * time.Minute)},
		{IntegrationID: otherOwner.ID, AppCode: "jellyfin", DisplayEventType: "playback_progress", DedupeKey: "other-owner", Status: "processed", PresentationVersion: 1, Presentation: []byte(`{}`), ReceivedAt: baseTime.Add(4 * time.Minute)},
	}
	if err := database.GetDB().Create(&events).Error; err != nil {
		t.Fatal(err)
	}

	page, err := webhookService.ListEvents(7, EventListFilter{IntegrationID: first.ID, EventType: "playback_progress", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].DedupeKey != "first-progress-old" || page.Limit != 1 || page.Offset != 1 || page.HasMore {
		t.Fatalf("unexpected filtered event page: %+v", page)
	}
	foreign, err := webhookService.ListEvents(7, EventListFilter{IntegrationID: otherOwner.ID, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if foreign.Total != 0 || len(foreign.Items) != 0 {
		t.Fatalf("another owner's App events were exposed: %+v", foreign)
	}
}

func TestEmbyCredentialRotationChangesTheEndpoint(t *testing.T) {
	setupWebhookTestDB(t)
	service := NewWebhookService()
	created, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "emby", Name: "Emby"})
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := service.RotateSecret(7, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.WebhookPath == created.WebhookPath {
		t.Fatal("Emby endpoint URL did not change after credential rotation")
	}
	if _, err := service.Ingest(created.EndpointKey, nil, []byte(`{"Event":"library.new"}`), "application/json"); err == nil {
		t.Fatal("old Emby endpoint should stop accepting messages after rotation")
	}
}

func TestWebhookDeduplicatesExternalEventID(t *testing.T) {
	setupWebhookTestDB(t)
	service := NewWebhookService()
	created, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "github", Name: "GitHub"})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"repository":{"full_name":"demo/repo"}}`)
	headers := map[string]string{"X-Hub-Signature-256": githubSignature(created.Secret, body), "X-GitHub-Event": "push", "X-GitHub-Delivery": "delivery-1"}
	first, err := service.Ingest(created.EndpointKey, headers, body, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Ingest(created.EndpointKey, headers, body, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	database.GetDB().Model(&model.WebhookEvent{}).Count(&count)
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
	if first.EventID == 0 || second.EventID != first.EventID {
		t.Fatalf("deduplicated event IDs = %d and %d", first.EventID, second.EventID)
	}
}

func TestEmbyWebhookPreservesSourceTypeAndUsesNormalizedDisplayType(t *testing.T) {
	setupWebhookTestDB(t)
	service := NewWebhookService()
	created, err := service.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "emby", Name: "Emby"})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"Event":"library.new","Item":{"Name":"Dune","Type":"Movie"}}`)
	if _, err := service.Ingest(created.EndpointKey, map[string]string{"X-Webhook-Secret": created.Secret}, body, "application/json"); err != nil {
		t.Fatal(err)
	}
	var event model.WebhookEvent
	if err := database.GetDB().First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.SourceEventType != "library.new" || event.DisplayEventType != "media_added" || event.IsFallback {
		t.Fatalf("unexpected event types: source=%q display=%q fallback=%v", event.SourceEventType, event.DisplayEventType, event.IsFallback)
	}
	var presentation map[string]interface{}
	if err := json.Unmarshal(event.Presentation, &presentation); err != nil {
		t.Fatal(err)
	}
	data := presentation["data"].(map[string]interface{})
	if data["category"] != "media" || data["source_event_type"] != "library.new" {
		t.Fatalf("unexpected normalized data: %+v", data)
	}
}

func TestWebhookFailureReturnsLogContext(t *testing.T) {
	setupWebhookTestDB(t)
	webhookService := NewWebhookService()
	created, err := webhookService.CreateIntegration(7, &CreateIntegrationRequest{AppCode: "generic", Name: "测试接入"})
	if err != nil {
		t.Fatal(err)
	}

	result, err := webhookService.Ingest(created.EndpointKey, map[string]string{"X-Webhook-Secret": "wrong-secret"}, []byte(`{"event_type":"test"}`), "application/json")
	if err == nil {
		t.Fatal("expected signature verification error")
	}
	var ingestError *WebhookIngestError
	if !errors.As(err, &ingestError) {
		t.Fatalf("error type = %T, want *WebhookIngestError", err)
	}
	if ingestError.Code != "webhook_signature_invalid" {
		t.Fatalf("error code = %q", ingestError.Code)
	}
	if result.AppCode != "generic" || result.IntegrationID != created.ID || result.EventID != 0 {
		t.Fatalf("unexpected failure context: %+v", result)
	}
}
