package service

import (
	"context"
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
	appLogging "seshat/internal/logging"
	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"
)

const maxWebhookBody = 2 << 20

type WebhookService struct {
	registry      *webhook.Registry
	mediaMetadata *MediaMetadataService
}

func NewWebhookService() *WebhookService {
	return &WebhookService{registry: webhook.NewRegistry(), mediaMetadata: NewMediaMetadataService()}
}

type CreateIntegrationRequest struct {
	AppCode string `json:"app_code" binding:"required"`
	Name    string `json:"name" binding:"required,max=100"`
}

type IntegrationResponse struct {
	model.AppIntegration
	WebhookPath        string `json:"webhook_path"`
	MediaAPIConfigured bool   `json:"media_api_configured"`
}

type IntegrationCreatedResponse struct {
	IntegrationResponse
	Secret string `json:"secret"`
}

type IntegrationSecretResponse struct {
	Secret string `json:"secret"`
}

type IntegrationMediaSettingsResponse struct {
	ServerURL  string `json:"server_url"`
	APIKey     string `json:"api_key"`
	Configured bool   `json:"configured"`
}

type UpdateIntegrationMediaSettingsRequest struct {
	ServerURL string `json:"server_url" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
}

type EventListResponse struct {
	Items   []model.WebhookEvent `json:"items"`
	Total   int64                `json:"total"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
	HasMore bool                 `json:"has_more"`
}

type EventListFilter struct {
	AppCode       string
	IntegrationID uint
	EventType     string
	Limit         int
	Offset        int
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
	item := &model.AppIntegration{OwnerID: ownerID, AppCode: provider.Code(), Name: strings.TrimSpace(req.Name), EndpointKey: randomEndpointKey(), Secret: secret, Config: datatypes.JSON([]byte("{}")), SecretConfig: datatypes.JSON([]byte("{}")), Enabled: true}
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

func (s *WebhookService) GetIntegrationMediaSettings(ownerID, id uint) (*IntegrationMediaSettingsResponse, error) {
	item, err := s.integrationForOwner(ownerID, id)
	if err != nil {
		return nil, err
	}
	if _, ok := s.mediaMetadata.adapter(item.AppCode); !ok {
		return nil, ErrMediaAPIUnsupported
	}
	settings, configured := mediaSettingsFromIntegration(item)
	return &IntegrationMediaSettingsResponse{ServerURL: settings.ServerURL, APIKey: settings.APIKey, Configured: configured}, nil
}

func (s *WebhookService) UpdateIntegrationMediaSettings(ownerID, id uint, req *UpdateIntegrationMediaSettingsRequest) (*IntegrationMediaSettingsResponse, error) {
	item, err := s.integrationForOwner(ownerID, id)
	if err != nil {
		return nil, err
	}
	if _, ok := s.mediaMetadata.adapter(item.AppCode); !ok {
		return nil, ErrMediaAPIUnsupported
	}
	settings, err := normalizeMediaServerSettings(req.ServerURL, req.APIKey)
	if err != nil {
		return nil, err
	}
	config := jsonObject(item.Config)
	secretConfig := jsonObject(item.SecretConfig)
	config["media_server_url"] = settings.ServerURL
	secretConfig["media_api_key"] = settings.APIKey
	item.Config, _ = json.Marshal(config)
	item.SecretConfig, _ = json.Marshal(secretConfig)
	if err := database.GetDB().Model(item).Select("config", "secret_config", "updated_at").Updates(item).Error; err != nil {
		return nil, err
	}
	return &IntegrationMediaSettingsResponse{ServerURL: settings.ServerURL, APIKey: settings.APIKey, Configured: true}, nil
}

func (s *WebhookService) TestIntegrationMediaSettings(ctx context.Context, ownerID, id uint, req *UpdateIntegrationMediaSettingsRequest) error {
	item, err := s.integrationForOwner(ownerID, id)
	if err != nil {
		return err
	}
	return s.mediaMetadata.TestMediaServer(ctx, item.AppCode, req.ServerURL, req.APIKey)
}

func (s *WebhookService) integrationForOwner(ownerID, id uint) (*model.AppIntegration, error) {
	var item model.AppIntegration
	if err := database.GetDB().Where("id = ? AND owner_id = ?", id, ownerID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func mediaSettingsFromIntegration(item *model.AppIntegration) (mediaServerSettings, bool) {
	if item == nil {
		return mediaServerSettings{}, false
	}
	config := jsonObject(item.Config)
	secretConfig := jsonObject(item.SecretConfig)
	serverURL, _ := config["media_server_url"].(string)
	apiKey, _ := secretConfig["media_api_key"].(string)
	settings, err := normalizeMediaServerSettings(serverURL, apiKey)
	return settings, err == nil
}

func jsonObject(data []byte) map[string]interface{} {
	result := map[string]interface{}{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &result)
	}
	return result
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
	publicToken, err := randomSecret()
	if err != nil {
		return result, newWebhookIngestError("webhook_public_token_failed", http.StatusInternalServerError, "生成消息访问标识失败", err)
	}
	now := time.Now()
	event := &model.WebhookEvent{PublicToken: publicToken, IntegrationID: integration.ID, AppCode: integration.AppCode, SourceEventType: sourceEventType, DisplayEventType: displayType, ExternalEventID: externalID, DedupeKey: dedupeKey, Status: "processed", IsFallback: isFallback, RawBody: string(body), ContentType: contentType, SafeHeaders: datatypes.JSON([]byte(`{}`)), ReceivedAt: now}
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

func (s *WebhookService) ListEvents(ownerID uint, filter EventListFilter) (*EventListResponse, error) {
	return s.ListEventsContext(context.Background(), ownerID, filter)
}

func (s *WebhookService) ListEventsContext(ctx context.Context, ownerID uint, filter EventListFilter) (*EventListResponse, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 30
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	query := database.GetDB().Model(&model.WebhookEvent{}).Joins("JOIN app_integrations ON app_integrations.id = webhook_events.integration_id").Where("app_integrations.owner_id = ?", ownerID)
	if filter.AppCode != "" {
		query = query.Where("webhook_events.app_code = ?", filter.AppCode)
	}
	if filter.IntegrationID > 0 {
		query = query.Where("webhook_events.integration_id = ?", filter.IntegrationID)
	}
	if filter.EventType != "" {
		query = query.Where("webhook_events.display_event_type = ?", filter.EventType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []model.WebhookEvent
	if err := query.Order("webhook_events.received_at desc, webhook_events.id desc").Limit(filter.Limit).Offset(filter.Offset).Find(&items).Error; err != nil {
		return nil, err
	}
	if err := s.materializeEvents(ctx, items); err != nil {
		return nil, err
	}
	return &EventListResponse{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset, HasMore: int64(filter.Offset+len(items)) < total}, nil
}

func (s *WebhookService) GetEvent(ownerID, id uint) (*WebhookEventDetail, error) {
	return s.GetEventContext(context.Background(), ownerID, id)
}

func (s *WebhookService) GetEventContext(ctx context.Context, ownerID, id uint) (*WebhookEventDetail, error) {
	var item model.WebhookEvent
	err := database.GetDB().Joins("JOIN app_integrations ON app_integrations.id = webhook_events.integration_id").Where("webhook_events.id = ? AND app_integrations.owner_id = ?", id, ownerID).First(&item).Error
	if err != nil {
		return &WebhookEventDetail{WebhookEvent: item}, err
	}
	var integration model.AppIntegration
	if err := database.GetDB().First(&integration, item.IntegrationID).Error; err != nil {
		return &WebhookEventDetail{WebhookEvent: item}, err
	}
	if err := s.materializeEvent(ctx, &item, &integration); err != nil {
		return &WebhookEventDetail{WebhookEvent: item}, err
	}
	return &WebhookEventDetail{WebhookEvent: item, RawBody: item.RawBody}, err
}

func (s *WebhookService) materializeEvents(ctx context.Context, items []model.WebhookEvent) error {
	if len(items) == 0 {
		return nil
	}
	integrationIDs := make([]uint, 0, len(items))
	seen := make(map[uint]struct{}, len(items))
	for i := range items {
		if _, ok := seen[items[i].IntegrationID]; !ok {
			seen[items[i].IntegrationID] = struct{}{}
			integrationIDs = append(integrationIDs, items[i].IntegrationID)
		}
	}
	var integrations []model.AppIntegration
	if err := database.GetDB().Where("id IN ?", integrationIDs).Find(&integrations).Error; err != nil {
		return err
	}
	byID := make(map[uint]*model.AppIntegration, len(integrations))
	for i := range integrations {
		byID[integrations[i].ID] = &integrations[i]
	}
	for i := range items {
		integration := byID[items[i].IntegrationID]
		if integration == nil {
			return errors.New("Webhook 接入实例不存在")
		}
		if err := s.materializeEvent(ctx, &items[i], integration); err != nil {
			return err
		}
	}
	return nil
}

func (s *WebhookService) materializeEvent(ctx context.Context, event *model.WebhookEvent, integration *model.AppIntegration) error {
	if event == nil || integration == nil {
		return errors.New("Webhook 消息或接入实例不存在")
	}
	provider, ok := s.registry.Get(event.AppCode)
	if !ok {
		return errors.New("Webhook App 类型不可用")
	}
	if len(event.RawBody) == 0 && len(event.Presentation) > 0 {
		return nil
	}
	request := webhook.IncomingRequest{Body: []byte(event.RawBody), SourceEventType: event.SourceEventType}
	presentation := provider.Normalize(event.DisplayEventType, request)
	presentation.SchemaVersion = 1
	if event.DisplayEventType != "media_deleted" {
		if err := s.mediaMetadata.EnrichContext(ctx, &presentation, integration); err != nil {
			appLogging.Warn("webhook", "展示媒体信息补充失败", appLogging.Fields{"app_code": integration.AppCode, "integration_id": integration.ID, "event_id": event.ID, "error": err})
		}
	}
	presentationJSON, err := json.Marshal(presentation)
	if err != nil {
		return err
	}
	event.Title = presentation.Title
	event.Summary = presentation.Summary
	event.Severity = presentation.Severity
	event.PresentationVersion = presentation.SchemaVersion
	event.Presentation = datatypes.JSON(presentationJSON)
	return nil
}

func (s *WebhookService) GetCachedMediaImage(token string) (*model.MediaMetadataCache, error) {
	return s.mediaMetadata.CachedImage(token)
}

func (s *WebhookService) integrationResponse(item *model.AppIntegration) IntegrationResponse {
	_, configured := mediaSettingsFromIntegration(item)
	return IntegrationResponse{AppIntegration: *item, WebhookPath: "/hooks/v1/" + item.EndpointKey, MediaAPIConfigured: configured}
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
