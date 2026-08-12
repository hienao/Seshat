package model

import "time"

// ApiRequestLog 记录一次后端 HTTP 请求；Webhook 请求可附带脱敏 Header 和受限正文预览。
type ApiRequestLog struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestID      string    `gorm:"size:100;index;not null" json:"request_id"`
	OccurredAt     time.Time `gorm:"index;not null" json:"occurred_at"`
	Method         string    `gorm:"size:12;index;not null" json:"method"`
	Route          string    `gorm:"size:255;index;not null" json:"route"`
	StatusCode     int       `gorm:"index;not null" json:"status_code"`
	LatencyMs      int64     `json:"latency_ms"`
	RequestBytes   int64     `json:"request_bytes"`
	ResponseBytes  int64     `json:"response_bytes"`
	RequestHeaders string    `gorm:"type:text" json:"request_headers,omitempty"`
	RequestBody    string    `gorm:"type:text" json:"request_body,omitempty"`
	ClientIP       string    `gorm:"size:64" json:"client_ip"`
	UserID         uint      `gorm:"index" json:"user_id,omitempty"`
	Username       string    `gorm:"size:100" json:"username,omitempty"`
	AppCode        string    `gorm:"size:50;index" json:"app_code,omitempty"`
	IntegrationID  uint      `gorm:"index" json:"integration_id,omitempty"`
	EventID        uint      `gorm:"index" json:"event_id,omitempty"`
	ErrorCode      string    `gorm:"size:100" json:"error_code,omitempty"`
	ErrorMessage   string    `gorm:"size:1000" json:"error_message,omitempty"`
	UserAgent      string    `gorm:"size:500" json:"user_agent,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ApiRequestLog) TableName() string { return "api_request_logs" }

// ApplicationLog 保存代码中主动记录的结构化业务日志。
type ApplicationLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	OccurredAt    time.Time `gorm:"index;not null" json:"occurred_at"`
	Level         string    `gorm:"size:10;index;not null" json:"level"`
	Source        string    `gorm:"size:255;index" json:"source,omitempty"`
	Message       string    `gorm:"type:text;not null" json:"message"`
	Fields        string    `gorm:"type:text" json:"fields,omitempty"`
	RequestID     string    `gorm:"size:100;index" json:"request_id,omitempty"`
	UserID        uint      `gorm:"index" json:"user_id,omitempty"`
	AppCode       string    `gorm:"size:50;index" json:"app_code,omitempty"`
	IntegrationID uint      `gorm:"index" json:"integration_id,omitempty"`
	EventID       uint      `gorm:"index" json:"event_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ApplicationLog) TableName() string { return "application_logs" }

// AdminAuditLog 保存清空日志等高风险管理操作，不能通过日志页面清除。
type AdminAuditLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ActorID    uint      `gorm:"index;not null" json:"actor_id"`
	Action     string    `gorm:"size:100;index;not null" json:"action"`
	Target     string    `gorm:"size:100;not null" json:"target"`
	Detail     string    `gorm:"type:text" json:"detail"`
	OccurredAt time.Time `gorm:"index;not null" json:"occurred_at"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }
