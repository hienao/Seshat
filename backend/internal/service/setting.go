package service

import (
	"basegoapp/internal/repository"
)

// SettingService 设置服务
type SettingService struct {
	settingRepo *repository.SettingRepository
}

// NewSettingService 创建设置服务实例
func NewSettingService() *SettingService {
	return &SettingService{
		settingRepo: repository.NewSettingRepository(),
	}
}

// SystemSettingsResponse 系统设置响应
type SystemSettingsResponse struct {
	AllowRegister bool `json:"allow_register"`
}

// GetSystemSettings 获取系统设置
func (s *SettingService) GetSystemSettings() (*SystemSettingsResponse, error) {
	settings, err := s.settingRepo.GetAllSystemSettings()
	if err != nil {
		return nil, err
	}

	response := &SystemSettingsResponse{}
	for _, setting := range settings {
		switch setting.Key {
		case "allow_register":
			response.AllowRegister = setting.Value == "true"
		}
	}
	return response, nil
}

// UpdateSystemSettingsRequest 更新系统设置请求
type UpdateSystemSettingsRequest struct {
	AllowRegister *bool `json:"allow_register"`
}

// UpdateSystemSettings 更新系统设置
func (s *SettingService) UpdateSystemSettings(req *UpdateSystemSettingsRequest) error {
	if req.AllowRegister != nil {
		value := "false"
		if *req.AllowRegister {
			value = "true"
		}
		if err := s.settingRepo.SetSystemSetting("allow_register", value); err != nil {
			return err
		}
	}
	return nil
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
	return s.settingRepo.InitDefaultSettings()
}
