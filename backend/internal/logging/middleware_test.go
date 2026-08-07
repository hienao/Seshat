package logging

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"seshat/config"
	"seshat/internal/model"

	"github.com/gin-gonic/gin"
)

func TestAccessLogPersistsWebhookContextAndError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager, err := NewManager(&config.Config{
		APILogEnabled:   true,
		APILogPath:      filepath.Join(t.TempDir(), "api-logs.db"),
		APILogQueueSize: 100,
		APILogBatchSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	router := gin.New()
	router.Use(RequestIDMiddleware(), AccessLogMiddleware(manager))
	router.POST("/hooks/v1/:endpointKey", func(c *gin.Context) {
		c.Set("app_code", "generic")
		c.Set("integration_id", uint(12))
		c.Set("event_id", uint(34))
		c.Set("error_code", "webhook_signature_invalid")
		c.Set("error_message", "Webhook 签名验证失败")
		c.Set("request_headers", `{"X-Hub-Signature-256":"[REDACTED]"}`)
		c.Set("request_body", `{"event_type":"test"}`)
		c.Status(http.StatusUnauthorized)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/hooks/v1/secret-endpoint", strings.NewReader(`{"event_type":"test"}`))
	router.ServeHTTP(recorder, request)
	manager.flush(0)

	var entry model.ApiRequestLog
	if err := manager.DB.First(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.Route != "/hooks/v1/:endpointKey" {
		t.Fatalf("route = %q", entry.Route)
	}
	if entry.AppCode != "generic" || entry.IntegrationID != 12 || entry.EventID != 34 {
		t.Fatalf("unexpected webhook context: %+v", entry)
	}
	if entry.ErrorCode != "webhook_signature_invalid" || entry.ErrorMessage != "Webhook 签名验证失败" {
		t.Fatalf("unexpected webhook error: %+v", entry)
	}
	if entry.RequestHeaders == "" || entry.RequestBody != `{"event_type":"test"}` {
		t.Fatalf("request snapshot was not persisted: %+v", entry)
	}
}
