package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"
)

var ErrPublicEventNotFound = errors.New("公开消息不存在")

// PublicEventResponse 是无鉴权页面的最小响应，只包含标准化展示所需字段。
type PublicEventResponse struct {
	AppCode             string               `json:"app_code"`
	SourceEventType     string               `json:"source_event_type"`
	DisplayEventType    string               `json:"display_event_type"`
	IsFallback          bool                 `json:"is_fallback"`
	Title               string               `json:"title"`
	Summary             string               `json:"summary"`
	Severity            string               `json:"severity"`
	OccurredAt          *time.Time           `json:"occurred_at,omitempty"`
	PresentationVersion int                  `json:"presentation_version"`
	Presentation        webhook.Presentation `json:"presentation"`
	ReceivedAt          time.Time            `json:"received_at"`
}

func (s *WebhookService) GetPublicEvent(publicToken string) (*PublicEventResponse, error) {
	publicToken = strings.TrimSpace(publicToken)
	if len(publicToken) < 32 || len(publicToken) > 64 {
		return nil, ErrPublicEventNotFound
	}
	var event model.WebhookEvent
	if err := database.GetDB().Where("public_token = ?", publicToken).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicEventNotFound
		}
		return nil, err
	}
	var presentation webhook.Presentation
	if err := json.Unmarshal(event.Presentation, &presentation); err != nil {
		return nil, err
	}
	// 默认类型的 raw_preview 来自原始请求，不属于公开页可展示内容。
	delete(presentation.Data, "raw_preview")
	return &PublicEventResponse{
		AppCode: event.AppCode, SourceEventType: event.SourceEventType, DisplayEventType: event.DisplayEventType,
		IsFallback: event.IsFallback, Title: event.Title, Summary: event.Summary, Severity: event.Severity,
		OccurredAt: event.OccurredAt, PresentationVersion: event.PresentationVersion,
		Presentation: presentation, ReceivedAt: event.ReceivedAt,
	}, nil
}

func ensurePublicEventToken(event *model.WebhookEvent) error {
	if event.PublicToken != "" {
		return nil
	}
	for attempt := 0; attempt < 3; attempt++ {
		token, err := randomSecret()
		if err != nil {
			return err
		}
		result := database.GetDB().Model(&model.WebhookEvent{}).Where("id = ? AND (public_token IS NULL OR public_token = '')", event.ID).Update("public_token", token)
		if result.Error != nil {
			continue
		}
		if result.RowsAffected > 0 {
			event.PublicToken = token
			return nil
		}
		if err := database.GetDB().Select("public_token").First(event, event.ID).Error; err != nil {
			return err
		}
		if event.PublicToken != "" {
			return nil
		}
	}
	return errors.New("生成消息公开访问标识失败")
}
