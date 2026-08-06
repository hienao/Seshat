package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID                 uint           `gorm:"primarykey" json:"id"`
	Username           string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password           string         `gorm:"size:255;not null" json:"-"`
	IsAdmin            bool           `gorm:"default:false" json:"is_admin"`
	RequiresAdminSetup bool           `gorm:"default:false;not null" json:"requires_admin_setup"`
	TokenVersion       int            `gorm:"default:1;not null" json:"-"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
