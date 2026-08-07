package logging

import (
	"path/filepath"
	"testing"
	"time"

	"basegoapp/config"
	"basegoapp/internal/model"
)

func TestSetRetentionDaysTriggersCleanup(t *testing.T) {
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

	entries := []model.ApiRequestLog{
		{RequestID: "expired", OccurredAt: time.Now().AddDate(0, 0, -31), Method: "GET", Route: "/expired", StatusCode: 200},
		{RequestID: "current", OccurredAt: time.Now().AddDate(0, 0, -29), Method: "GET", Route: "/current", StatusCode: 200},
	}
	if err := manager.DB.Create(&entries).Error; err != nil {
		t.Fatal(err)
	}

	manager.SetRetentionDays(30)
	deadline := time.Now().Add(2 * time.Second)
	for {
		var count int64
		if err := manager.DB.Model(&model.ApiRequestLog{}).Where("request_id = ?", "expired").Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expired log was not cleaned after retention setting changed")
		}
		time.Sleep(10 * time.Millisecond)
	}

	var currentCount int64
	if err := manager.DB.Model(&model.ApiRequestLog{}).Where("request_id = ?", "current").Count(&currentCount).Error; err != nil {
		t.Fatal(err)
	}
	if currentCount != 1 {
		t.Fatalf("current log count = %d, want 1", currentCount)
	}
}
