package repository

import (
	"basegoapp/internal/model"
	"basegoapp/pkg/database"

	"gorm.io/gorm"
)

// SettingRepository 设置数据访问层
type SettingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 创建设置仓库实例
func NewSettingRepository() *SettingRepository {
	return &SettingRepository{db: database.GetDB()}
}

// GetSystemSetting 获取系统设置
func (r *SettingRepository) GetSystemSetting(key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// SetSystemSetting 设置系统设置（存在则更新，不存在则创建）
func (r *SettingRepository) SetSystemSetting(key, value string) error {
	var setting model.SystemSetting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		setting = model.SystemSetting{Key: key, Value: value}
		return r.db.Create(&setting).Error
	}
	if err != nil {
		return err
	}
	setting.Value = value
	return r.db.Save(&setting).Error
}

// GetAllSystemSettings 获取所有系统设置
func (r *SettingRepository) GetAllSystemSettings() ([]model.SystemSetting, error) {
	var settings []model.SystemSetting
	err := r.db.Find(&settings).Error
	return settings, err
}

// InitDefaultSettings 初始化默认设置
func (r *SettingRepository) InitDefaultSettings() error {
	defaults := map[string]string{
		"allow_register": "false",
	}

	for key, value := range defaults {
		var count int64
		r.db.Model(&model.SystemSetting{}).Where("key = ?", key).Count(&count)
		if count == 0 {
			if err := r.db.Create(&model.SystemSetting{Key: key, Value: value}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
