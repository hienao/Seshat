package logging

import (
	"path/filepath"
	"strings"
	"testing"

	"seshat/config"
	"seshat/internal/model"
)

func TestApplicationLogPersistsLevelContextAndRedactedFields(t *testing.T) {
	manager, err := NewManager(&config.Config{
		APILogEnabled:   true,
		APILogPath:      filepath.Join(t.TempDir(), "application-logs.db"),
		APILogQueueSize: 100,
		APILogBatchSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	SetDefaultManager(manager)
	defer SetDefaultManager(nil)

	Warn("webhook", "Webhook 消息被拒绝", Fields{
		"request_id":     "req-1",
		"app_code":       "generic",
		"integration_id": uint(12),
		"event_id":       uint(34),
		"webhook_secret": "must-not-be-stored",
	})
	manager.flush(0)

	var entry model.ApplicationLog
	if err := manager.DB.First(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.Level != "WARN" || entry.Source != "webhook" || entry.Message != "Webhook 消息被拒绝" {
		t.Fatalf("unexpected application log: %+v", entry)
	}
	if entry.RequestID != "req-1" || entry.AppCode != "generic" || entry.IntegrationID != 12 || entry.EventID != 34 {
		t.Fatalf("unexpected application context: %+v", entry)
	}
	if entry.Fields == "" || strings.Contains(entry.Fields, "must-not-be-stored") || !strings.Contains(entry.Fields, "[REDACTED]") {
		t.Fatalf("sensitive fields were not redacted: %s", entry.Fields)
	}
}
