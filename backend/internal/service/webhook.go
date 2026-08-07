package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"
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

type IntegrationSecretResponse struct {
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

// WebhookIngestResult 保存可写入接口日志的 Webhook 处理关联信息。
// 即使处理失败，只要已经识别到接入实例，也会返回已知字段。
type WebhookIngestResult struct {
	AppCode       string
	IntegrationID uint
	EventID       uint
}

// WebhookIngestError 为 Webhook 接收失败提供稳定的日志错误码和 HTTP 状态。
type WebhookIngestError struct {
	Code       string
	StatusCode int
	Message    string
	Cause      error
}

func (e *WebhookIngestError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func newWebhookIngestError(code string, statusCode int, message string, cause error) error {
	return &WebhookIngestError{Code: code, StatusCode: statusCode, Message: message, Cause: cause}
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
	if provider, ok := s.registry.Get(item.AppCode); ok && provider.Definition().AuthMode == "endpoint_url" {
		item.EndpointKey = randomEndpointKey()
	}
	if err := database.GetDB().Save(&item).Error; err != nil {
		return nil, err
	}
	return &IntegrationCreatedResponse{IntegrationResponse: s.integrationResponse(&item), Secret: secret}, nil
}

func (s *WebhookService) GetIntegrationSecret(ownerID, id uint) (*IntegrationSecretResponse, error) {
	var item model.AppIntegration
	if err := database.GetDB().Select("secret").Where("id = ? AND owner_id = ?", id, ownerID).First(&item).Error; err != nil {
		return nil, err
	}
	return &IntegrationSecretResponse{Secret: item.Secret}, nil
}

func (s *WebhookService) Ingest(endpointKey string, headers map[string]string, body []byte, contentType string) (*WebhookIngestResult, error) {
	result := &WebhookIngestResult{}
	var integration model.AppIntegration
	if err := database.GetDB().Where("endpoint_key = ? AND enabled = ?", endpointKey, true).First(&integration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, newWebhookIngestError("webhook_integration_not_found", http.StatusUnauthorized, "Webhook 接入不存在或已禁用", nil)
		}
		return result, newWebhookIngestError("webhook_integration_lookup_failed", http.StatusInternalServerError, "查询 Webhook 接入失败", err)
	}
	result.AppCode = integration.AppCode
	result.IntegrationID = integration.ID
	if len(body) > maxWebhookBody {
		return result, newWebhookIngestError("webhook_body_too_large", http.StatusRequestEntityTooLarge, "Webhook 消息体过大", nil)
	}
	provider, ok := s.registry.Get(integration.AppCode)
	if !ok {
		return result, newWebhookIngestError("webhook_app_unavailable", http.StatusServiceUnavailable, "Webhook App 类型不可用", nil)
	}
	request := webhook.IncomingRequest{Headers: headers, Body: body}
	if !provider.Verify(integration.Secret, request) {
		return result, newWebhookIngestError("webhook_signature_invalid", http.StatusUnauthorized, "Webhook 签名验证失败", nil)
	}

	sourceEventType := provider.DetectType(request)
	eventType := provider.MapType(sourceEventType)
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
		result.EventID = existing.ID
		return result, nil
	}
	presentation := provider.Normalize(displayType, request)
	presentation.SchemaVersion = 1
	presentationJSON, _ := json.Marshal(presentation)
	title, summary, severity := presentation.Title, presentation.Summary, presentation.Severity
	now := time.Now()
	event := &model.WebhookEvent{IntegrationID: integration.ID, AppCode: integration.AppCode, SourceEventType: sourceEventType, DisplayEventType: displayType, ExternalEventID: externalID, DedupeKey: dedupeKey, Status: "processed", IsFallback: isFallback, Title: title, Summary: summary, Severity: severity, PresentationVersion: 1, Presentation: datatypes.JSON(presentationJSON), RawBody: string(body), ContentType: contentType, SafeHeaders: datatypes.JSON([]byte(`{}`)), ReceivedAt: now}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(event).Error; err != nil {
			return err
		}
		return queueNotificationDelivery(tx, &integration, event)
	}); err != nil {
		return result, newWebhookIngestError("webhook_event_persist_failed", http.StatusInternalServerError, "保存 Webhook 消息失败", err)
	}
	result.EventID = event.ID
	return result, nil
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
