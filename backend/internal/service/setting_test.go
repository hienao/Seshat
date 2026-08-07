package service

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"seshat/internal/model"
	"seshat/pkg/database"
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

func TestSettingServiceStoresAndRedactsHTTPProxy(t *testing.T) {
	setupSettingTestDB(t)
	settingService := NewSettingService()
	if err := settingService.InitDefaultSettings(); err != nil {
		t.Fatal(err)
	}
	proxyURL := "http://proxy-user:proxy-password@127.0.0.1:7890"
	if err := settingService.UpdateSystemSettings(&UpdateSystemSettingsRequest{HTTPProxyURL: &proxyURL}); err != nil {
		t.Fatal(err)
	}
	settings, err := settingService.GetSystemSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.HTTPProxyConfigured || settings.HTTPProxyDisplay != "http://127.0.0.1:7890" {
		t.Fatalf("unexpected proxy settings: %+v", settings)
	}
	if settingService.HTTPProxyURL() != proxyURL {
		t.Fatal("stored proxy URL was not available to the notification sender")
	}
	clear := true
	if err := settingService.UpdateSystemSettings(&UpdateSystemSettingsRequest{ClearHTTPProxy: &clear}); err != nil {
		t.Fatal(err)
	}
	if settingService.HTTPProxyURL() != "" {
		t.Fatal("proxy URL was not cleared")
	}
}

func TestSettingServiceRejectsInvalidHTTPProxy(t *testing.T) {
	setupSettingTestDB(t)
	settingService := NewSettingService()
	invalid := "socks5://127.0.0.1:1080"
	if err := settingService.UpdateSystemSettings(&UpdateSystemSettingsRequest{HTTPProxyURL: &invalid}); !errors.Is(err, ErrInvalidHTTPProxyURL) {
		t.Fatalf("error = %v, want ErrInvalidHTTPProxyURL", err)
	}
}
