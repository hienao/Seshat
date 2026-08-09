package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotificationChannelNotFound = errors.New("推送渠道不存在")
	ErrNotificationChannelInUse    = errors.New("推送渠道仍被接入实例使用")
	ErrInvalidNotificationChannel  = errors.New("推送渠道配置无效")
)

type NotificationChannelRequest struct {
	Name        string                 `json:"name" binding:"required,max=100"`
	Type        string                 `json:"type" binding:"required"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	Credentials map[string]interface{} `json:"credentials"`
}

type NotificationChannelResponse struct {
	ID             uint                   `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Enabled        bool                   `json:"enabled"`
	Config         map[string]interface{} `json:"config"`
	HasCredentials bool                   `json:"has_credentials"`
	BindingCount   int64                  `json:"binding_count"`
	LastTestStatus string                 `json:"last_test_status,omitempty"`
	LastTestAt     *time.Time             `json:"last_test_at,omitempty"`
	LastTestError  string                 `json:"last_test_error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type NotificationEventTypeSetting struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	Notify    bool   `json:"notify"`
}

type IntegrationNotificationSettings struct {
	IntegrationID  uint                           `json:"integration_id"`
	AppCode        string                         `json:"app_code"`
	ChannelID      *uint                          `json:"channel_id,omitempty"`
	ChannelEnabled bool                           `json:"channel_enabled"`
	EventTypes     []NotificationEventTypeSetting `json:"event_types"`
}

type UpdateIntegrationNotificationSettingsRequest struct {
	ChannelID         *uint    `json:"channel_id"`
	EnabledEventTypes []string `json:"enabled_event_types"`
}

type NotificationService struct{ registry *webhook.Registry }

type NotificationDeliveryResponse struct {
	model.NotificationDelivery
	IntegrationName string `json:"integration_name"`
}

type EventNotificationStatus struct {
	State     string                        `json:"state"`
	Reason    string                        `json:"reason,omitempty"`
	Delivery  *NotificationDeliveryResponse `json:"delivery,omitempty"`
	ChannelID *uint                         `json:"channel_id,omitempty"`
}

func NewNotificationService() *NotificationService {
	return &NotificationService{registry: webhook.NewRegistry()}
}

func (s *NotificationService) ListChannels(ownerID uint) ([]NotificationChannelResponse, error) {
	var items []model.NotificationChannel
	if err := database.GetDB().Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]NotificationChannelResponse, 0, len(items))
	for i := range items {
		response, err := channelResponse(database.GetDB(), &items[i])
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return result, nil
}

func (s *NotificationService) CreateChannel(ownerID uint, request *NotificationChannelRequest) (*NotificationChannelResponse, error) {
	channel, err := channelFromRequest(ownerID, nil, request)
	if err != nil {
		return nil, err
	}
	if err := database.GetDB().Create(channel).Error; err != nil {
		return nil, err
	}
	response, err := channelResponse(database.GetDB(), channel)
	return &response, err
}

func (s *NotificationService) UpdateChannel(ownerID, id uint, request *NotificationChannelRequest) (*NotificationChannelResponse, error) {
	var existing model.NotificationChannel
	if err := database.GetDB().Where("id = ? AND owner_id = ?", id, ownerID).First(&existing).Error; err != nil {
		return nil, ErrNotificationChannelNotFound
	}
	channel, err := channelFromRequest(ownerID, &existing, request)
	if err != nil {
		return nil, err
	}
	channel.ID = existing.ID
	channel.CreatedAt = existing.CreatedAt
	channel.LastTestAt = existing.LastTestAt
	channel.LastTestStatus = existing.LastTestStatus
	channel.LastTestError = existing.LastTestError
	if err := database.GetDB().Save(channel).Error; err != nil {
		return nil, err
	}
	response, err := channelResponse(database.GetDB(), channel)
	return &response, err
}

func (s *NotificationService) DeleteChannel(ownerID, id uint) error {
	var channel model.NotificationChannel
	if err := database.GetDB().Where("id = ? AND owner_id = ?", id, ownerID).First(&channel).Error; err != nil {
		return ErrNotificationChannelNotFound
	}
	var count int64
	if err := database.GetDB().Model(&model.AppIntegration{}).Where("notification_channel_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrNotificationChannelInUse
	}
	if err := database.GetDB().Model(&model.NotificationDelivery{}).Where("channel_id = ? AND status IN ?", id, []string{"pending", "sending", "retrying"}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrNotificationChannelInUse
	}
	return database.GetDB().Delete(&channel).Error
}

func (s *NotificationService) ListDeliveries(ownerID uint, status string, limit int) ([]NotificationDeliveryResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := database.GetDB().Table("notification_deliveries").Select("notification_deliveries.*, app_integrations.name AS integration_name").Joins("JOIN app_integrations ON app_integrations.id = notification_deliveries.integration_id").Where("app_integrations.owner_id = ?", ownerID)
	if status != "" {
		query = query.Where("notification_deliveries.status = ?", status)
	}
	var items []NotificationDeliveryResponse
	return items, query.Order("notification_deliveries.created_at DESC").Limit(limit).Scan(&items).Error
}

func (s *NotificationService) RetryDelivery(ownerID uint, id uint64) error {
	var delivery model.NotificationDelivery
	err := database.GetDB().Joins("JOIN app_integrations ON app_integrations.id = notification_deliveries.integration_id").Where("notification_deliveries.id = ? AND app_integrations.owner_id = ?", id, ownerID).First(&delivery).Error
	if err != nil {
		return errors.New("推送记录不存在")
	}
	if delivery.Status != "failed" {
		return errors.New("只有最终失败的推送可以手动重试")
	}
	var channel model.NotificationChannel
	if err := database.GetDB().Where("id = ? AND owner_id = ?", delivery.ChannelID, ownerID).First(&channel).Error; err != nil {
		return errors.New("原推送渠道已不存在，无法重试")
	}
	if !channel.Enabled {
		return errors.New("原推送渠道已停用，请先启用后再重试")
	}
	now := time.Now()
	return database.GetDB().Model(&model.NotificationDelivery{}).Where("id = ?", delivery.ID).Updates(map[string]interface{}{"status": "retrying", "attempt_count": 0, "next_attempt_at": now, "last_error": ""}).Error
}

func (s *NotificationService) GetEventNotificationStatus(ownerID, eventID uint) (*EventNotificationStatus, error) {
	var event model.WebhookEvent
	if err := database.GetDB().Table("webhook_events").Select("webhook_events.*").Joins("JOIN app_integrations ON app_integrations.id = webhook_events.integration_id").Where("webhook_events.id = ? AND app_integrations.owner_id = ?", eventID, ownerID).First(&event).Error; err != nil {
		return nil, errors.New("消息不存在")
	}
	var delivery NotificationDeliveryResponse
	err := database.GetDB().Table("notification_deliveries").Select("notification_deliveries.*, app_integrations.name AS integration_name").Joins("JOIN app_integrations ON app_integrations.id = notification_deliveries.integration_id").Where("notification_deliveries.event_id = ? AND app_integrations.owner_id = ?", event.ID, ownerID).Order("notification_deliveries.created_at DESC").First(&delivery).Error
	if err == nil {
		return &EventNotificationStatus{State: delivery.Status, Delivery: &delivery, ChannelID: &delivery.ChannelID}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var integration model.AppIntegration
	if err := database.GetDB().Where("id = ? AND owner_id = ?", event.IntegrationID, ownerID).First(&integration).Error; err != nil {
		return nil, errors.New("接入实例不存在")
	}
	if integration.NotificationChannelID == nil {
		return &EventNotificationStatus{State: "not_configured", Reason: "接入实例未绑定推送渠道"}, nil
	}
	var channel model.NotificationChannel
	if err := database.GetDB().Where("id = ? AND owner_id = ?", *integration.NotificationChannelID, ownerID).First(&channel).Error; err != nil {
		return &EventNotificationStatus{State: "not_configured", Reason: "绑定的推送渠道不存在"}, nil
	}
	if !channel.Enabled {
		return &EventNotificationStatus{State: "channel_disabled", Reason: "绑定的推送渠道已停用", ChannelID: &channel.ID}, nil
	}
	var count int64
	if err := database.GetDB().Model(&model.IntegrationNotificationRule{}).Where("integration_id = ? AND event_type = ? AND enabled = ?", integration.ID, event.DisplayEventType, true).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return &EventNotificationStatus{State: "not_matched", Reason: "该消息类型未开启推送", ChannelID: &channel.ID}, nil
	}
	return &EventNotificationStatus{State: "not_created", Reason: "消息接收时尚未创建推送任务", ChannelID: &channel.ID}, nil
}

func (s *NotificationService) GetIntegrationSettings(ownerID, integrationID uint) (*IntegrationNotificationSettings, error) {
	var integration model.AppIntegration
	if err := database.GetDB().Where("id = ? AND owner_id = ?", integrationID, ownerID).First(&integration).Error; err != nil {
		return nil, errors.New("接入实例不存在")
	}
	provider, ok := s.registry.Get(integration.AppCode)
	if !ok {
		return nil, errors.New("App 类型不可用")
	}
	var rules []model.IntegrationNotificationRule
	if err := database.GetDB().Where("integration_id = ? AND enabled = ?", integration.ID, true).Find(&rules).Error; err != nil {
		return nil, err
	}
	enabled := make(map[string]bool, len(rules))
	for _, rule := range rules {
		enabled[rule.EventType] = true
	}
	definition := provider.Definition()
	settings := &IntegrationNotificationSettings{IntegrationID: integration.ID, AppCode: integration.AppCode, ChannelID: integration.NotificationChannelID}
	if integration.NotificationChannelID != nil {
		var channel model.NotificationChannel
		if database.GetDB().Where("id = ? AND owner_id = ?", *integration.NotificationChannelID, ownerID).First(&channel).Error == nil {
			settings.ChannelEnabled = channel.Enabled
		}
	}
	for _, eventType := range definition.EventTypes {
		settings.EventTypes = append(settings.EventTypes, NotificationEventTypeSetting{Code: eventType.Code, Name: eventType.Name, IsDefault: eventType.Code == definition.DefaultEventType, Notify: enabled[eventType.Code]})
	}
	return settings, nil
}

func (s *NotificationService) UpdateIntegrationSettings(ownerID, integrationID uint, request *UpdateIntegrationNotificationSettingsRequest) (*IntegrationNotificationSettings, error) {
	var integration model.AppIntegration
	if err := database.GetDB().Where("id = ? AND owner_id = ?", integrationID, ownerID).First(&integration).Error; err != nil {
		return nil, errors.New("接入实例不存在")
	}
	provider, ok := s.registry.Get(integration.AppCode)
	if !ok {
		return nil, errors.New("App 类型不可用")
	}
	allowed := make(map[string]bool)
	for _, eventType := range provider.Definition().EventTypes {
		allowed[eventType.Code] = true
	}
	seen := make(map[string]bool)
	for _, eventType := range request.EnabledEventTypes {
		if !allowed[eventType] {
			return nil, errors.New("包含无效的消息类型: " + eventType)
		}
		seen[eventType] = true
	}
	if request.ChannelID != nil {
		var count int64
		if err := database.GetDB().Model(&model.NotificationChannel{}).Where("id = ? AND owner_id = ?", *request.ChannelID, ownerID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrNotificationChannelNotFound
		}
	}
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&integration).Update("notification_channel_id", request.ChannelID).Error; err != nil {
			return err
		}
		if err := tx.Where("integration_id = ?", integration.ID).Delete(&model.IntegrationNotificationRule{}).Error; err != nil {
			return err
		}
		for eventType := range seen {
			rule := model.IntegrationNotificationRule{IntegrationID: integration.ID, EventType: eventType, Enabled: true}
			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetIntegrationSettings(ownerID, integrationID)
}

func queueNotificationDelivery(tx *gorm.DB, integration *model.AppIntegration, event *model.WebhookEvent) error {
	if integration.NotificationChannelID == nil {
		return nil
	}
	var channel model.NotificationChannel
	if err := tx.Where("id = ? AND owner_id = ? AND enabled = ?", *integration.NotificationChannelID, integration.OwnerID, true).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var count int64
	if err := tx.Model(&model.IntegrationNotificationRule{}).Where("integration_id = ? AND event_type = ? AND enabled = ?", integration.ID, event.DisplayEventType, true).Count(&count).Error; err != nil || count == 0 {
		return err
	}
	now := time.Now()
	delivery := model.NotificationDelivery{EventID: event.ID, IntegrationID: integration.ID, ChannelID: channel.ID, ChannelName: channel.Name, ChannelType: channel.Type, EventType: event.DisplayEventType, Status: "pending", NextAttemptAt: &now}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&delivery).Error
}

func channelFromRequest(ownerID uint, existing *model.NotificationChannel, request *NotificationChannelRequest) (*model.NotificationChannel, error) {
	name := strings.TrimSpace(request.Name)
	typeName := strings.ToLower(strings.TrimSpace(request.Type))
	adapter, ok := defaultNotificationChannelRegistry.Get(typeName)
	if name == "" || !ok {
		return nil, ErrInvalidNotificationChannel
	}
	if existing != nil && existing.Type != typeName {
		return nil, errors.New("不能修改渠道类型")
	}
	publicConfig, credentials, err := adapter.Sanitize(request.Config, request.Credentials)
	if err != nil {
		return nil, err
	}
	config, err := json.Marshal(publicConfig)
	if err != nil {
		return nil, ErrInvalidNotificationChannel
	}
	secretValues := map[string]interface{}{}
	if existing != nil && len(existing.SecretConfig) > 0 {
		_ = json.Unmarshal(existing.SecretConfig, &secretValues)
	}
	if len(credentials) > 0 {
		for key, value := range credentials {
			if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
				continue
			}
			secretValues[key] = value
		}
	}
	secret, err := json.Marshal(secretValues)
	if err != nil {
		return nil, ErrInvalidNotificationChannel
	}
	if err := adapter.Validate(publicConfig, secretValues); err != nil {
		return nil, err
	}
	return &model.NotificationChannel{OwnerID: ownerID, Name: name, Type: typeName, Enabled: request.Enabled, Config: datatypes.JSON(config), SecretConfig: datatypes.JSON(secret)}, nil
}

func channelResponse(db *gorm.DB, channel *model.NotificationChannel) (NotificationChannelResponse, error) {
	config := map[string]interface{}{}
	_ = json.Unmarshal(channel.Config, &config)
	var count int64
	if err := db.Model(&model.AppIntegration{}).Where("notification_channel_id = ?", channel.ID).Count(&count).Error; err != nil {
		return NotificationChannelResponse{}, err
	}
	return NotificationChannelResponse{ID: channel.ID, Name: channel.Name, Type: channel.Type, Enabled: channel.Enabled, Config: config, HasCredentials: len(channel.SecretConfig) > 2, BindingCount: count, LastTestStatus: channel.LastTestStatus, LastTestAt: channel.LastTestAt, LastTestError: channel.LastTestError, CreatedAt: channel.CreatedAt, UpdatedAt: channel.UpdatedAt}, nil
}
