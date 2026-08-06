package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"basegoapp/internal/model"
	"basegoapp/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	if err := service.Ingest(created.EndpointKey, map[string]string{"X-Webhook-Secret": created.Secret}, body, "application/json"); err != nil {
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
	var presentation map[string]interface{}
	if err := json.Unmarshal(event.Presentation, &presentation); err != nil {
		t.Fatal(err)
	}
	if presentation["schema_version"] != float64(1) {
		t.Fatal("missing presentation schema version")
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
	if err := service.Ingest(created.EndpointKey, headers, body, "application/json"); err != nil {
		t.Fatal(err)
	}
	if err := service.Ingest(created.EndpointKey, headers, body, "application/json"); err != nil {
		t.Fatal(err)
	}
	var count int64
	database.GetDB().Model(&model.WebhookEvent{}).Count(&count)
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
}
