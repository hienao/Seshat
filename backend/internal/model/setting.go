package model

// SystemSetting 系统设置模型
type SystemSetting struct {
	ID    uint   `gorm:"primarykey" json:"id"`
	Key   string `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value string `gorm:"size:500;not null" json:"value"`
}

// TableName 指定表名
func (SystemSetting) TableName() string {
	return "system_settings"
}

// UserSetting 用户设置模型（预留）
type UserSetting struct {
	ID     uint   `gorm:"primarykey" json:"id"`
	UserID uint   `gorm:"index;not null" json:"user_id"`
	Key    string `gorm:"size:100;not null" json:"key"`
	Value  string `gorm:"size:500;not null" json:"value"`
}

// TableName 指定表名
func (UserSetting) TableName() string {
	return "user_settings"
}
