package model

import (
	"time"

	"gorm.io/datatypes"
)

// AppIntegration 是用户配置的一个 Webhook 接入实例。
type AppIntegration struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	OwnerID          uint           `gorm:"index;not null" json:"owner_id"`
	AppCode          string         `gorm:"size:50;index;not null" json:"app_code"`
	Name             string         `gorm:"size:100;not null" json:"name"`
	EndpointKey      string         `gorm:"uniqueIndex;size:100;not null" json:"endpoint_key"`
	SecretCiphertext string         `gorm:"size:500;not null" json:"-"`
	Config           datatypes.JSON `gorm:"type:json" json:"config"`
	Enabled          bool           `gorm:"default:true;not null" json:"enabled"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (AppIntegration) TableName() string { return "app_integrations" }

// WebhookEvent 是一次收到的 Webhook 消息及其归一化结果。
type WebhookEvent struct {
	ID                  uint           `gorm:"primarykey" json:"id"`
	IntegrationID       uint           `gorm:"index;not null" json:"integration_id"`
	AppCode             string         `gorm:"size:50;index;not null" json:"app_code"`
	SourceEventType     string         `gorm:"size:150;index" json:"source_event_type"`
	DisplayEventType    string         `gorm:"size:150;index;not null" json:"display_event_type"`
	ExternalEventID     string         `gorm:"size:200" json:"external_event_id"`
	DedupeKey           string         `gorm:"size:128;not null" json:"-"`
	Status              string         `gorm:"size:30;index;not null" json:"status"`
	IsFallback          bool           `gorm:"not null;default:false" json:"is_fallback"`
	Title               string         `gorm:"size:300" json:"title"`
	Summary             string         `gorm:"size:1000" json:"summary"`
	Severity            string         `gorm:"size:20" json:"severity"`
	OccurredAt          *time.Time     `json:"occurred_at"`
	PresentationVersion int            `gorm:"not null;default:1" json:"presentation_version"`
	Presentation        datatypes.JSON `gorm:"type:json" json:"presentation"`
	RawBody             string         `gorm:"type:text" json:"-"`
	ContentType         string         `gorm:"size:150" json:"content_type"`
	SafeHeaders         datatypes.JSON `gorm:"type:json" json:"safe_headers,omitempty"`
	ProcessError        string         `gorm:"size:1000" json:"process_error,omitempty"`
	ReceivedAt          time.Time      `gorm:"index;not null" json:"received_at"`
	CreatedAt           time.Time      `json:"created_at"`
}

func (WebhookEvent) TableName() string { return "webhook_events" }
