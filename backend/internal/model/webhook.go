package model

import (
	"time"

	"gorm.io/datatypes"
)

// AppIntegration 是用户配置的一个 Webhook 接入实例。
type AppIntegration struct {
	ID                    uint           `gorm:"primarykey" json:"id"`
	OwnerID               uint           `gorm:"index;not null" json:"owner_id"`
	AppCode               string         `gorm:"size:50;index;not null" json:"app_code"`
	Name                  string         `gorm:"size:100;not null" json:"name"`
	EndpointKey           string         `gorm:"uniqueIndex;size:100;not null" json:"endpoint_key"`
	Secret                string         `gorm:"size:100;not null" json:"-"`
	Config                datatypes.JSON `gorm:"type:json" json:"config"`
	SecretConfig          datatypes.JSON `gorm:"type:json" json:"-"`
	Enabled               bool           `gorm:"default:true;not null" json:"enabled"`
	NotificationChannelID *uint          `gorm:"index" json:"notification_channel_id,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

func (AppIntegration) TableName() string { return "app_integrations" }

// WebhookEvent 是一次收到的 Webhook 消息及其归一化结果。
type WebhookEvent struct {
	ID                  uint           `gorm:"primarykey" json:"id"`
	PublicToken         string         `gorm:"uniqueIndex;size:64;default:null" json:"-"`
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

// MediaMetadataCache 持久化外部媒体平台返回的展示信息，避免重复查询。
type MediaMetadataCache struct {
	ID         uint   `gorm:"primarykey"`
	CacheKey   string `gorm:"uniqueIndex;size:220;not null"`
	Provider   string `gorm:"index;size:30;not null"`
	ExternalID string `gorm:"index;size:100;not null"`
	MediaType  string `gorm:"size:30"`
	Title      string `gorm:"size:500"`
	Overview   string `gorm:"type:text"`
	ImageURL   string `gorm:"size:1000"`
	ImageToken string `gorm:"index;size:64"`
	ImageData  []byte
	ImageType  string         `gorm:"size:100"`
	Metadata   datatypes.JSON `gorm:"type:json"`
	SourceURL  string         `gorm:"size:1000"`
	ExpiresAt  time.Time      `gorm:"index;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (MediaMetadataCache) TableName() string { return "media_metadata_cache" }

type NotificationChannel struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	OwnerID        uint           `gorm:"index;not null" json:"owner_id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Type           string         `gorm:"size:20;index;not null" json:"type"`
	Enabled        bool           `gorm:"default:false;not null" json:"enabled"`
	Config         datatypes.JSON `gorm:"type:json" json:"config"`
	SecretConfig   datatypes.JSON `gorm:"type:json" json:"-"`
	LastTestStatus string         `gorm:"size:20" json:"last_test_status,omitempty"`
	LastTestAt     *time.Time     `json:"last_test_at,omitempty"`
	LastTestError  string         `gorm:"size:1000" json:"last_test_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (NotificationChannel) TableName() string { return "notification_channels" }

type IntegrationNotificationRule struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	IntegrationID uint      `gorm:"uniqueIndex:idx_integration_event_type;not null" json:"integration_id"`
	EventType     string    `gorm:"uniqueIndex:idx_integration_event_type;size:150;not null" json:"event_type"`
	Enabled       bool      `gorm:"default:false;not null" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (IntegrationNotificationRule) TableName() string { return "integration_notification_rules" }

type NotificationDelivery struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID        uint       `gorm:"uniqueIndex:idx_delivery_event_channel;index;not null" json:"event_id"`
	IntegrationID  uint       `gorm:"index:idx_delivery_integration_created;not null" json:"integration_id"`
	ChannelID      uint       `gorm:"uniqueIndex:idx_delivery_event_channel;index:idx_delivery_channel_created;not null" json:"channel_id"`
	ChannelName    string     `gorm:"size:100;not null" json:"channel_name"`
	ChannelType    string     `gorm:"size:20;not null" json:"channel_type"`
	EventType      string     `gorm:"size:150;index;not null" json:"event_type"`
	Status         string     `gorm:"size:20;index:idx_delivery_status_next;not null" json:"status"`
	AttemptCount   int        `gorm:"not null;default:0" json:"attempt_count"`
	NextAttemptAt  *time.Time `gorm:"index:idx_delivery_status_next" json:"next_attempt_at,omitempty"`
	LastStatusCode int        `json:"last_status_code,omitempty"`
	LastError      string     `gorm:"size:1000" json:"last_error,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	CreatedAt      time.Time  `gorm:"index:idx_delivery_integration_created;index:idx_delivery_channel_created" json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (NotificationDelivery) TableName() string { return "notification_deliveries" }
