package service

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"seshat/internal/repository"
)

const (
	DefaultAPILogRetentionDays = 30
	MinAPILogRetentionDays     = 1
	MaxAPILogRetentionDays     = 3650
)

var (
	ErrInvalidAPILogRetentionDays = errors.New("接口日志保留天数必须在 1 到 3650 天之间")
	ErrInvalidHTTPProxyURL        = errors.New("HTTP 代理地址必须是有效的 http:// 或 https:// URL")
	ErrConflictingHTTPProxyUpdate = errors.New("不能同时设置和清空 HTTP 代理")
)

// APILogRetentionUpdater 由接口日志管理器实现，用于让设置立即生效。
type APILogRetentionUpdater interface {
	SetRetentionDays(days int)
}

// SettingService 设置服务
type SettingService struct {
	settingRepo      *repository.SettingRepository
	retentionUpdater APILogRetentionUpdater
}

// NewSettingService 创建设置服务实例
func NewSettingService(retentionUpdaters ...APILogRetentionUpdater) *SettingService {
	service := &SettingService{
		settingRepo: repository.NewSettingRepository(),
	}
	if len(retentionUpdaters) > 0 {
		service.retentionUpdater = retentionUpdaters[0]
	}
	return service
}

// SystemSettingsResponse 系统设置响应
type SystemSettingsResponse struct {
	AllowRegister                   bool   `json:"allow_register"`
	APILogRetentionDays             int    `json:"api_log_retention_days"`
	AllowPrivateNotificationTargets bool   `json:"allow_private_notification_targets"`
	HTTPProxyConfigured             bool   `json:"http_proxy_configured"`
	HTTPProxyDisplay                string `json:"http_proxy_display,omitempty"`
}

// GetSystemSettings 获取系统设置
func (s *SettingService) GetSystemSettings() (*SystemSettingsResponse, error) {
	settings, err := s.settingRepo.GetAllSystemSettings()
	if err != nil {
		return nil, err
	}

	response := &SystemSettingsResponse{APILogRetentionDays: DefaultAPILogRetentionDays}
	for _, setting := range settings {
		switch setting.Key {
		case "allow_register":
			response.AllowRegister = setting.Value == "true"
		case "api_log_retention_days":
			if days, parseErr := strconv.Atoi(setting.Value); parseErr == nil && validAPILogRetentionDays(days) {
				response.APILogRetentionDays = days
			}
		case "allow_private_notification_targets":
			response.AllowPrivateNotificationTargets = setting.Value == "true"
		case "http_proxy_url":
			if setting.Value != "" {
				response.HTTPProxyConfigured = true
				response.HTTPProxyDisplay = displayHTTPProxyURL(setting.Value)
			}
		}
	}
	return response, nil
}

// UpdateSystemSettingsRequest 更新系统设置请求
type UpdateSystemSettingsRequest struct {
	AllowRegister                   *bool   `json:"allow_register"`
	APILogRetentionDays             *int    `json:"api_log_retention_days"`
	AllowPrivateNotificationTargets *bool   `json:"allow_private_notification_targets"`
	HTTPProxyURL                    *string `json:"http_proxy_url"`
	ClearHTTPProxy                  *bool   `json:"clear_http_proxy"`
}

// UpdateSystemSettings 更新系统设置
func (s *SettingService) UpdateSystemSettings(req *UpdateSystemSettingsRequest) error {
	if req.APILogRetentionDays != nil && !validAPILogRetentionDays(*req.APILogRetentionDays) {
		return ErrInvalidAPILogRetentionDays
	}
	if req.HTTPProxyURL != nil && req.ClearHTTPProxy != nil && *req.ClearHTTPProxy {
		return ErrConflictingHTTPProxyUpdate
	}
	if req.HTTPProxyURL != nil {
		proxyURL := strings.TrimSpace(*req.HTTPProxyURL)
		if !validHTTPProxyURL(proxyURL) {
			return ErrInvalidHTTPProxyURL
		}
	}
	if req.AllowRegister != nil {
		value := "false"
		if *req.AllowRegister {
			value = "true"
		}
		if err := s.settingRepo.SetSystemSetting("allow_register", value); err != nil {
			return err
		}
	}
	if req.APILogRetentionDays != nil {
		days := *req.APILogRetentionDays
		if err := s.settingRepo.SetSystemSetting("api_log_retention_days", strconv.Itoa(days)); err != nil {
			return err
		}
		if s.retentionUpdater != nil {
			s.retentionUpdater.SetRetentionDays(days)
		}
	}
	if req.AllowPrivateNotificationTargets != nil {
		value := "false"
		if *req.AllowPrivateNotificationTargets {
			value = "true"
		}
		if err := s.settingRepo.SetSystemSetting("allow_private_notification_targets", value); err != nil {
			return err
		}
	}
	if req.HTTPProxyURL != nil {
		if err := s.settingRepo.SetSystemSetting("http_proxy_url", strings.TrimSpace(*req.HTTPProxyURL)); err != nil {
			return err
		}
	} else if req.ClearHTTPProxy != nil && *req.ClearHTTPProxy {
		if err := s.settingRepo.SetSystemSetting("http_proxy_url", ""); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingService) AllowPrivateNotificationTargets() bool {
	setting, err := s.settingRepo.GetSystemSetting("allow_private_notification_targets")
	return err == nil && setting.Value == "true"
}

func (s *SettingService) HTTPProxyURL() string {
	setting, err := s.settingRepo.GetSystemSetting("http_proxy_url")
	if err != nil || !validHTTPProxyURL(setting.Value) {
		return ""
	}
	return setting.Value
}

// IsRegistrationAllowed 检查是否允许注册
func (s *SettingService) IsRegistrationAllowed() bool {
	setting, err := s.settingRepo.GetSystemSetting("allow_register")
	if err != nil {
		return false
	}
	return setting.Value == "true"
}

// InitDefaultSettings 初始化默认设置
func (s *SettingService) InitDefaultSettings() error {
	if err := s.settingRepo.InitDefaultSettings(); err != nil {
		return err
	}
	settings, err := s.GetSystemSettings()
	if err != nil {
		return err
	}
	if s.retentionUpdater != nil {
		s.retentionUpdater.SetRetentionDays(settings.APILogRetentionDays)
	}
	return nil
}

func validAPILogRetentionDays(days int) bool {
	return days >= MinAPILogRetentionDays && days <= MaxAPILogRetentionDays
}

func validHTTPProxyURL(value string) bool {
	if value == "" || len(value) > 500 {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == "" && parsed.Fragment == ""
}

func displayHTTPProxyURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "已配置"
	}
	parsed.User = nil
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
