package service

import (
	"errors"
	"testing"

	"basegoapp/internal/model"
	"basegoapp/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type retentionUpdaterStub struct {
	days int
}

func (s *retentionUpdaterStub) SetRetentionDays(days int) {
	s.days = days
}

func setupSettingTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:setting-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.SystemSetting{}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.DB = nil })
}

func TestSettingServiceInitializesDefaultLogRetention(t *testing.T) {
	setupSettingTestDB(t)
	updater := &retentionUpdaterStub{}
	settingService := NewSettingService(updater)

	if err := settingService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	settings, err := settingService.GetSystemSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.APILogRetentionDays != DefaultAPILogRetentionDays {
		t.Fatalf("retention days = %d, want %d", settings.APILogRetentionDays, DefaultAPILogRetentionDays)
	}
	if updater.days != DefaultAPILogRetentionDays {
		t.Fatalf("updater days = %d, want %d", updater.days, DefaultAPILogRetentionDays)
	}
}

func TestSettingServiceUpdatesLogRetentionImmediately(t *testing.T) {
	setupSettingTestDB(t)
	updater := &retentionUpdaterStub{}
	settingService := NewSettingService(updater)
	if err := settingService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}

	days := 90
	if err := settingService.UpdateSystemSettings(&UpdateSystemSettingsRequest{APILogRetentionDays: &days}); err != nil {
		t.Fatal(err)
	}
	settings, err := settingService.GetSystemSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.APILogRetentionDays != days || updater.days != days {
		t.Fatalf("stored days = %d, updater days = %d, want %d", settings.APILogRetentionDays, updater.days, days)
	}
}

func TestSettingServiceRejectsInvalidLogRetention(t *testing.T) {
	setupSettingTestDB(t)
	settingService := NewSettingService()
	if err := settingService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}

	days := 0
	err := settingService.UpdateSystemSettings(&UpdateSystemSettingsRequest{APILogRetentionDays: &days})
	if !errors.Is(err, ErrInvalidAPILogRetentionDays) {
		t.Fatalf("error = %v, want ErrInvalidAPILogRetentionDays", err)
	}
}
