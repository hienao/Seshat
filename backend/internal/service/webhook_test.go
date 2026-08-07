package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

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
