package service

import (
	"path/filepath"
	"testing"
	"time"

	"seshat/config"
	"seshat/internal/logging"
	"seshat/internal/model"
)

func TestApplicationLogServiceFiltersByLevel(t *testing.T) {
	manager, err := logging.NewManager(&config.Config{
		APILogEnabled:   true,
		APILogPath:      filepath.Join(t.TempDir(), "application-logs.db"),
		APILogQueueSize: 100,
		APILogBatchSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	entries := []model.ApplicationLog{
		{OccurredAt: time.Now(), Level: "INFO", Source: "server", Message: "started", Fields: `{"port":"8080"}`},
		{OccurredAt: time.Now(), Level: "ERROR", Source: "webhook", Message: "persist failed", Fields: `{"event_id":12}`},
	}
	if err := manager.DB.Create(&entries).Error; err != nil {
		t.Fatal(err)
	}

	service := NewApplicationLogService(manager)
	result, err := service.List(ApplicationLogFilter{Level: "error"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Level != "ERROR" {
		t.Fatalf("unexpected filtered logs: %+v", result)
	}
	if result.Items[0].Fields != "" {
		t.Fatal("list response must omit structured fields")
	}
	detail, err := service.Get(entries[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Fields == "" {
		t.Fatal("detail response must include structured fields")
	}
}
