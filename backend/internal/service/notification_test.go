package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"seshat/internal/model"
	"seshat/pkg/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupNotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.NotificationChannel{}, &model.AppIntegration{}, &model.WebhookEvent{}, &model.IntegrationNotificationRule{}, &model.NotificationDelivery{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = nil })
	return db
}

func createNotificationFixture(t *testing.T, db *gorm.DB) (model.NotificationChannel, model.AppIntegration) {
	t.Helper()
	channel := model.NotificationChannel{OwnerID: 7, Name: "测试渠道", Type: "webhook", Enabled: true, Config: []byte(`{}`), SecretConfig: []byte(`{"url":"https://example.com/notify"}`)}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	integration := model.AppIntegration{OwnerID: 7, AppCode: "jellyfin", Name: "Jellyfin", EndpointKey: "notification-test", Secret: "secret", Enabled: true, NotificationChannelID: &channel.ID}
	if err := db.Create(&integration).Error; err != nil {
		t.Fatal(err)
	}
	return channel, integration
}

func TestIntegrationNotificationRulesDefaultToDisabledAndIncludeDefaultType(t *testing.T) {
	db := setupNotificationTestDB(t)
	_, integration := createNotificationFixture(t, db)
	service := NewNotificationService()

	settings, err := service.GetIntegrationSettings(7, integration.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.ChannelID == nil || len(settings.EventTypes) != 5 {
		t.Fatalf("unexpected settings: %+v", settings)
	}
	defaultFound := false
	for _, eventType := range settings.EventTypes {
		if eventType.Notify {
			t.Fatalf("event type %q should be disabled by default", eventType.Code)
		}
		if eventType.IsDefault && eventType.Code == "__default__" {
			defaultFound = true
		}
	}
	if !defaultFound {
		t.Fatal("default event type was not returned")
	}
}

func TestQueueNotificationDeliveryUsesRulesAndDeduplicates(t *testing.T) {
	db := setupNotificationTestDB(t)
	channel, integration := createNotificationFixture(t, db)
	event := model.WebhookEvent{IntegrationID: integration.ID, AppCode: "jellyfin", SourceEventType: "ItemAdded", DisplayEventType: "media_added", DedupeKey: "event-1", Status: "processed", ReceivedAt: time.Now()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}

	if err := queueNotificationDelivery(db, &integration, &event); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.NotificationDelivery{}).Count(&count)
	if count != 0 {
		t.Fatalf("delivery count = %d before rule is enabled", count)
	}
	status, err := NewNotificationService().GetEventNotificationStatus(7, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "not_matched" {
		t.Fatalf("notification status = %q, want not_matched", status.State)
	}
	if err := db.Create(&model.IntegrationNotificationRule{IntegrationID: integration.ID, EventType: "media_added", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := queueNotificationDelivery(db, &integration, &event); err != nil {
		t.Fatal(err)
	}
	if err := queueNotificationDelivery(db, &integration, &event); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.NotificationDelivery{}).Count(&count)
	if count != 1 {
		t.Fatalf("delivery count = %d, want 1", count)
	}
	var delivery model.NotificationDelivery
	if err := db.First(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	if delivery.ChannelID != channel.ID || delivery.Status != "pending" {
		t.Fatalf("unexpected delivery: %+v", delivery)
	}
}

func TestUpdateChannelMergesCredentialsWithoutReturningSecrets(t *testing.T) {
	setupNotificationTestDB(t)
	service := NewNotificationService()
	created, err := service.CreateChannel(7, &NotificationChannelRequest{Name: "Webhook", Type: "webhook", Config: map[string]interface{}{"use_proxy": true}, Credentials: map[string]interface{}{"url": "https://example.com/one", "headers": map[string]interface{}{"Authorization": "Bearer old"}}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateChannel(7, created.ID, &NotificationChannelRequest{Name: "Webhook 2", Type: "webhook", Config: map[string]interface{}{"use_proxy": true}, Credentials: map[string]interface{}{"url": "https://example.com/two"}})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.HasCredentials {
		t.Fatal("response should indicate that credentials exist")
	}
	if updated.Config["use_proxy"] != true {
		t.Fatalf("channel proxy setting was not persisted: %+v", updated.Config)
	}
	encoded, _ := json.Marshal(updated)
	if string(encoded) == "" || containsSecret(string(encoded), "Bearer old") {
		t.Fatal("channel response exposed secret credentials")
	}
	var channel model.NotificationChannel
	if err := database.GetDB().First(&channel, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	var credentials map[string]interface{}
	if err := json.Unmarshal(channel.SecretConfig, &credentials); err != nil {
		t.Fatal(err)
	}
	if credentials["url"] != "https://example.com/two" || credentials["headers"] == nil {
		t.Fatalf("credentials were not merged: %+v", credentials)
	}
}

func TestNotificationResourcesAreOwnerScoped(t *testing.T) {
	db := setupNotificationTestDB(t)
	channel, integration := createNotificationFixture(t, db)
	service := NewNotificationService()
	items, err := service.ListChannels(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("other owner can list channels: %+v", items)
	}
	if _, err := service.UpdateChannel(8, channel.ID, &NotificationChannelRequest{Name: "stolen", Type: "webhook"}); err != ErrNotificationChannelNotFound {
		t.Fatalf("other owner update error = %v", err)
	}
	if _, err := service.GetIntegrationSettings(8, integration.ID); err == nil {
		t.Fatal("other owner can read integration notification settings")
	}
}

func TestValidateOutboundURLPrivateNetworkPolicy(t *testing.T) {
	if err := validateOutboundURL("https://127.0.0.1/notify", false); err == nil {
		t.Fatal("loopback target should be blocked by default")
	}
	if err := validateOutboundURL("http://127.0.0.1/notify", true); err != nil {
		t.Fatalf("private HTTP target should be allowed when enabled: %v", err)
	}
}

func TestWebhookSenderUsesNormalizedPayloadAndDoesNotLeakResponseBody(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer channel-token" {
			t.Errorf("authorization header = %q", request.Header.Get("Authorization"))
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		if payload["type"] != "seshat.notification" || payload["event"] == nil || payload["raw_body"] != nil {
			t.Errorf("unexpected payload: %+v", payload)
		}
		if requests == 2 {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("token-should-not-be-persisted"))
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	channel := model.NotificationChannel{Type: "webhook", Config: []byte(`{}`), SecretConfig: []byte(fmt.Sprintf(`{"url":%q,"headers":{"Authorization":"Bearer channel-token"}}`, server.URL))}
	message := outboundMessage{Title: "新增媒体", Body: "电影已加入", Severity: "info", AppCode: "jellyfin", IntegrationID: 2, IntegrationName: "家庭媒体库", EventID: 3, EventType: "media_added", ReceivedAt: time.Now()}
	if err := sendChannel(t.Context(), &channel, message, true, ""); err != nil {
		t.Fatal(err)
	}
	err := sendChannel(t.Context(), &channel, message, true, "")
	var sendErr *deliveryError
	if err == nil || !errors.As(err, &sendErr) || !sendErr.retryable {
		t.Fatalf("expected retryable delivery error, got %v", err)
	}
	if containsSecret(err.Error(), "token-should-not-be-persisted") {
		t.Fatal("downstream response body leaked into delivery error")
	}
}

func TestNotificationSenderUsesConfiguredHTTPProxy(t *testing.T) {
	proxyRequests := 0
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxyRequests++
		if request.URL.Host != "127.0.0.1:34567" {
			t.Errorf("proxy target = %q", request.URL.String())
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()
	channel := model.NotificationChannel{Type: "webhook", Config: []byte(`{"use_proxy":true}`), SecretConfig: []byte(`{"url":"http://127.0.0.1:34567/notify"}`)}
	if err := sendChannel(t.Context(), &channel, outboundMessage{Title: "代理测试", Body: "消息"}, true, proxy.URL); err != nil {
		t.Fatal(err)
	}
	if proxyRequests != 1 {
		t.Fatalf("proxy request count = %d, want 1", proxyRequests)
	}
}

func TestChannelProxyPolicyRequiresSystemProxy(t *testing.T) {
	setupNotificationTestDB(t)
	settingService := NewSettingService()
	if err := settingService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	channel := model.NotificationChannel{Config: []byte(`{"use_proxy":true}`)}
	if _, _, err := notificationNetworkPolicyForChannel(&channel); err == nil {
		t.Fatal("channel using proxy should fail when the system proxy is not configured")
	}
	proxyURL := "http://127.0.0.1:7890"
	if err := settingService.UpdateSystemSettings(&UpdateSystemSettingsRequest{HTTPProxyURL: &proxyURL}); err != nil {
		t.Fatal(err)
	}
	_, actualProxy, err := notificationNetworkPolicyForChannel(&channel)
	if err != nil || actualProxy != proxyURL {
		t.Fatalf("proxy policy = %q, %v", actualProxy, err)
	}
}

func TestNotificationWorkerRecoversSendingDeliveries(t *testing.T) {
	db := setupNotificationTestDB(t)
	channel, integration := createNotificationFixture(t, db)
	event := model.WebhookEvent{IntegrationID: integration.ID, AppCode: "jellyfin", DisplayEventType: "media_added", DedupeKey: "event-recovery", Status: "processed", ReceivedAt: time.Now()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	delivery := model.NotificationDelivery{EventID: event.ID, IntegrationID: integration.ID, ChannelID: channel.ID, ChannelName: channel.Name, ChannelType: channel.Type, EventType: event.DisplayEventType, Status: "sending"}
	if err := db.Create(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&delivery).Update("updated_at", time.Now().Add(-2*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	worker := NewNotificationWorker()
	worker.Start()
	worker.Close()
	if err := db.First(&delivery, delivery.ID).Error; err != nil {
		t.Fatal(err)
	}
	if delivery.Status != "retrying" || delivery.NextAttemptAt == nil {
		t.Fatalf("orphaned delivery was not recovered: %+v", delivery)
	}
}

func containsSecret(value, secret string) bool {
	for i := 0; i+len(secret) <= len(value); i++ {
		if value[i:i+len(secret)] == secret {
			return true
		}
	}
	return false
}
