package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"basegoapp/internal/model"
	"basegoapp/internal/webhook"
	"basegoapp/pkg/database"
	"gorm.io/datatypes"
)

const maxWebhookBody = 2 << 20

type WebhookService struct {
	registry *webhook.Registry
}

func NewWebhookService() *WebhookService {
	return &WebhookService{registry: webhook.NewRegistry()}
}

type CreateIntegrationRequest struct {
	AppCode string `json:"app_code" binding:"required"`
	Name    string `json:"name" binding:"required,max=100"`
}

type IntegrationResponse struct {
	model.AppIntegration
	WebhookPath string `json:"webhook_path"`
}

type IntegrationCreatedResponse struct {
	IntegrationResponse
	Secret string `json:"secret"`
}

type EventListResponse struct {
	Items []model.WebhookEvent `json:"items"`
	Total int64                `json:"total"`
}

type WebhookEventDetail struct {
	model.WebhookEvent
	RawBody string `json:"raw_body,omitempty"`
}

func (s *WebhookService) Catalog() []webhook.AppDefinition { return s.registry.Definitions() }

func (s *WebhookService) CreateIntegration(ownerID uint, req *CreateIntegrationRequest) (*IntegrationCreatedResponse, error) {
	provider, ok := s.registry.Get(req.AppCode)
	if !ok {
		return nil, errors.New("不支持的 App 类型")
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	item := &model.AppIntegration{OwnerID: ownerID, AppCode: provider.Code(), Name: strings.TrimSpace(req.Name), EndpointKey: randomEndpointKey(), Secret: secret, Config: datatypes.JSON([]byte("{}")), Enabled: true}
	if item.Name == "" {
		return nil, errors.New("接入名称不能为空")
	}
	if err := database.GetDB().Create(item).Error; err != nil {
		return nil, err
	}
	return &IntegrationCreatedResponse{IntegrationResponse: s.integrationResponse(item), Secret: secret}, nil
}

func (s *WebhookService) ListIntegrations(ownerID uint) ([]IntegrationResponse, error) {
	var items []model.AppIntegration
	if err := database.GetDB().Where("owner_id = ?", ownerID).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]IntegrationResponse, 0, len(items))
	for i := range items {
		result = append(result, s.integrationResponse(&items[i]))
	}
	return result, nil
}

func (s *WebhookService) RotateSecret(ownerID, id uint) (*IntegrationCreatedResponse, error) {
	var item model.AppIntegration
	if err := database.GetDB().Where("id = ? AND owner_id = ?", id, ownerID).First(&item).Error; err != nil {
		return nil, err
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	item.Secret = secret
	if err := database.GetDB().Save(&item).Error; err != nil {
		return nil, err
	}
	return &IntegrationCreatedResponse{IntegrationResponse: s.integrationResponse(&item), Secret: secret}, nil
}

func (s *WebhookService) Ingest(endpointKey string, headers map[string]string, body []byte, contentType string) error {
	if len(body) > maxWebhookBody {
		return errors.New("Webhook 消息体过大")
	}
	var integration model.AppIntegration
	if err := database.GetDB().Where("endpoint_key = ? AND enabled = ?", endpointKey, true).First(&integration).Error; err != nil {
		return errors.New("Webhook 接入不存在或已禁用")
	}
	provider, ok := s.registry.Get(integration.AppCode)
	if !ok {
		return errors.New("Webhook App 类型不可用")
	}
	request := webhook.IncomingRequest{Headers: headers, Body: body}
	if !provider.Verify(integration.Secret, request) {
		return errors.New("Webhook 签名验证失败")
	}

	eventType := provider.DetectType(request)
	definition := provider.Definition()
	displayType, isFallback := eventType, false
	if !containsEventType(definition.EventTypes, eventType) {
		displayType, isFallback = definition.DefaultEventType, true
	}
	externalID := provider.ExternalEventID(request)
	dedupeKey := externalID
	if dedupeKey == "" {
		sum := sha256.Sum256(body)
		dedupeKey = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	var existing model.WebhookEvent
	if database.GetDB().Where("integration_id = ? AND dedupe_key = ?", integration.ID, dedupeKey).First(&existing).Error == nil {
		return nil
	}
	presentation := provider.Normalize(eventType, request)
	presentation.SchemaVersion = 1
	presentationJSON, _ := json.Marshal(presentation)
	title, summary, severity := presentation.Title, presentation.Summary, presentation.Severity
	now := time.Now()
	event := &model.WebhookEvent{IntegrationID: integration.ID, AppCode: integration.AppCode, SourceEventType: eventType, DisplayEventType: displayType, ExternalEventID: externalID, DedupeKey: dedupeKey, Status: "processed", IsFallback: isFallback, Title: title, Summary: summary, Severity: severity, PresentationVersion: 1, Presentation: datatypes.JSON(presentationJSON), RawBody: string(body), ContentType: contentType, SafeHeaders: datatypes.JSON([]byte(`{}`)), ReceivedAt: now}
	if err := database.GetDB().Create(event).Error; err != nil {
		return err
	}
	return nil
}

func (s *WebhookService) ListEvents(ownerID uint, appCode, eventType string, limit, offset int) (*EventListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}
	query := database.GetDB().Model(&model.WebhookEvent{}).Joins("JOIN app_integrations ON app_integrations.id = webhook_events.integration_id").Where("app_integrations.owner_id = ?", ownerID)
	if appCode != "" {
		query = query.Where("webhook_events.app_code = ?", appCode)
	}
	if eventType != "" {
		query = query.Where("webhook_events.display_event_type = ?", eventType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []model.WebhookEvent
	if err := query.Order("webhook_events.received_at desc").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, err
	}
	return &EventListResponse{Items: items, Total: total}, nil
}

func (s *WebhookService) GetEvent(ownerID, id uint) (*WebhookEventDetail, error) {
	var item model.WebhookEvent
	err := database.GetDB().Joins("JOIN app_integrations ON app_integrations.id = webhook_events.integration_id").Where("webhook_events.id = ? AND app_integrations.owner_id = ?", id, ownerID).First(&item).Error
	return &WebhookEventDetail{WebhookEvent: item, RawBody: item.RawBody}, err
}

func (s *WebhookService) integrationResponse(item *model.AppIntegration) IntegrationResponse {
	return IntegrationResponse{AppIntegration: *item, WebhookPath: "/hooks/v1/" + item.EndpointKey}
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func randomEndpointKey() string {
	b := make([]byte, 18)
	_, _ = io.ReadFull(rand.Reader, b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func containsEventType(items []webhook.EventTypeDefinition, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}
