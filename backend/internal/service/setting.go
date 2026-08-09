package service

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"seshat/internal/repository"
)

const (
	DefaultAPILogRetentionDays = 7
	MinAPILogRetentionDays     = 1
	MaxAPILogRetentionDays     = 30
)

var (
	ErrInvalidAPILogRetentionDays = errors.New("日志与媒体缓存保留天数必须在 1 到 30 天之间")
	ErrInvalidHTTPProxyURL        = errors.New("HTTP 代理地址必须是有效的 http:// 或 https:// URL")
	ErrConflictingHTTPProxyUpdate = errors.New("不能同时设置和清空 HTTP 代理")
	ErrConflictingTMDBTokenUpdate = errors.New("不能同时设置和清空 TMDB Read Access Token")
	ErrInvalidTMDBToken           = errors.New("TMDB Read Access Token 无效")
	ErrInvalidPublicBaseURL       = errors.New("对外访问地址必须是有效的 http:// 或 https:// URL")
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
	AllowRegister       bool   `json:"allow_register"`
	APILogRetentionDays int    `json:"api_log_retention_days"`
	HTTPProxyConfigured bool   `json:"http_proxy_configured"`
	HTTPProxyURL        string `json:"http_proxy_url"`
	TMDBConfigured      bool   `json:"tmdb_configured"`
	TMDBReadAccessToken string `json:"tmdb_read_access_token"`
	TMDBUseProxy        bool   `json:"tmdb_use_proxy"`
	PublicBaseURL       string `json:"public_base_url"`
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
		case "http_proxy_url":
			if setting.Value != "" {
				response.HTTPProxyConfigured = true
				response.HTTPProxyURL = setting.Value
			}
		case "tmdb_read_access_token":
			response.TMDBReadAccessToken = strings.TrimSpace(setting.Value)
			response.TMDBConfigured = response.TMDBReadAccessToken != ""
		case "tmdb_use_proxy":
			response.TMDBUseProxy = setting.Value == "true"
		case "public_base_url":
			if validPublicBaseURL(setting.Value) {
				response.PublicBaseURL = normalizePublicBaseURL(setting.Value)
			}
		}
	}
	return response, nil
}

// UpdateSystemSettingsRequest 更新系统设置请求
type UpdateSystemSettingsRequest struct {
	AllowRegister       *bool   `json:"allow_register"`
	APILogRetentionDays *int    `json:"api_log_retention_days"`
	HTTPProxyURL        *string `json:"http_proxy_url"`
	ClearHTTPProxy      *bool   `json:"clear_http_proxy"`
	TMDBReadAccessToken *string `json:"tmdb_read_access_token"`
	ClearTMDBToken      *bool   `json:"clear_tmdb_token"`
	TMDBUseProxy        *bool   `json:"tmdb_use_proxy"`
	PublicBaseURL       *string `json:"public_base_url"`
}

// UpdateSystemSettings 更新系统设置
func (s *SettingService) UpdateSystemSettings(req *UpdateSystemSettingsRequest) error {
	if req.APILogRetentionDays != nil && !validAPILogRetentionDays(*req.APILogRetentionDays) {
		return ErrInvalidAPILogRetentionDays
	}
	if req.HTTPProxyURL != nil && req.ClearHTTPProxy != nil && *req.ClearHTTPProxy {
		return ErrConflictingHTTPProxyUpdate
	}
	if req.TMDBReadAccessToken != nil && req.ClearTMDBToken != nil && *req.ClearTMDBToken {
		return ErrConflictingTMDBTokenUpdate
	}
	if req.HTTPProxyURL != nil {
		proxyURL := strings.TrimSpace(*req.HTTPProxyURL)
		if !validHTTPProxyURL(proxyURL) {
			return ErrInvalidHTTPProxyURL
		}
	}
	if req.TMDBReadAccessToken != nil {
		token := strings.TrimSpace(*req.TMDBReadAccessToken)
		if token == "" || len(token) > 2000 {
			return ErrInvalidTMDBToken
		}
	}
	if req.PublicBaseURL != nil && !validPublicBaseURL(*req.PublicBaseURL) {
		return ErrInvalidPublicBaseURL
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
	if req.HTTPProxyURL != nil {
		if err := s.settingRepo.SetSystemSetting("http_proxy_url", strings.TrimSpace(*req.HTTPProxyURL)); err != nil {
			return err
		}
	} else if req.ClearHTTPProxy != nil && *req.ClearHTTPProxy {
		if err := s.settingRepo.SetSystemSetting("http_proxy_url", ""); err != nil {
			return err
		}
	}
	if req.TMDBReadAccessToken != nil {
		token := strings.TrimSpace(*req.TMDBReadAccessToken)
		if err := s.settingRepo.SetSystemSetting("tmdb_read_access_token", token); err != nil {
			return err
		}
	} else if req.ClearTMDBToken != nil && *req.ClearTMDBToken {
		if err := s.settingRepo.SetSystemSetting("tmdb_read_access_token", ""); err != nil {
			return err
		}
	}
	if req.TMDBUseProxy != nil {
		value := "false"
		if *req.TMDBUseProxy {
			value = "true"
		}
		if err := s.settingRepo.SetSystemSetting("tmdb_use_proxy", value); err != nil {
			return err
		}
	}
	if req.PublicBaseURL != nil {
		if err := s.settingRepo.SetSystemSetting("public_base_url", normalizePublicBaseURL(*req.PublicBaseURL)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingService) HTTPProxyURL() string {
	setting, err := s.settingRepo.GetSystemSetting("http_proxy_url")
	if err != nil || !validHTTPProxyURL(setting.Value) {
		return ""
	}
	return setting.Value
}

func (s *SettingService) TMDBReadAccessToken() string {
	setting, err := s.settingRepo.GetSystemSetting("tmdb_read_access_token")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(setting.Value)
}

func (s *SettingService) TMDBUsesHTTPProxy() bool {
	setting, err := s.settingRepo.GetSystemSetting("tmdb_use_proxy")
	return err == nil && setting.Value == "true"
}

func (s *SettingService) PublicBaseURL() string {
	setting, err := s.settingRepo.GetSystemSetting("public_base_url")
	if err != nil || !validPublicBaseURL(setting.Value) {
		return ""
	}
	return normalizePublicBaseURL(setting.Value)
}

func (s *SettingService) RetentionDays() int {
	settings, err := s.GetSystemSettings()
	if err != nil {
		return DefaultAPILogRetentionDays
	}
	return settings.APILogRetentionDays
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

func validPublicBaseURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 500 {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return false
	}
	return (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == "" && parsed.Fragment == ""
}

func normalizePublicBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
