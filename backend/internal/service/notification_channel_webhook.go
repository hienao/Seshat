package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"seshat/internal/model"
)

type webhookNotificationAdapter struct{}

func (webhookNotificationAdapter) Type() string { return "webhook" }

func (webhookNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	cleanConfig, cleanCredentials, err := sanitizeNotificationChannelFields(config, credentials, nil, []string{"url", "headers"})
	if err != nil {
		return nil, nil, err
	}
	if headers, ok := cleanCredentials["headers"]; ok {
		values, ok := headers.(map[string]interface{})
		if !ok {
			return nil, nil, errors.New("Webhook 请求头必须是 JSON 对象")
		}
		for key, value := range values {
			if _, ok := value.(string); !ok || isUnsafeNotificationHeader(key) {
				return nil, nil, errors.New("Webhook 请求头配置无效: " + key)
			}
		}
	}
	return cleanConfig, cleanCredentials, nil
}

func (webhookNotificationAdapter) Validate(_ map[string]interface{}, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "url") {
		return errors.New("Webhook URL 不能为空")
	}
	return nil
}

func (a webhookNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (webhookNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	_, credentials := notificationChannelConfiguration(channel)
	endpoint, _ := credentials["url"].(string)
	headers := map[string]string{"Content-Type": "application/json", "Accept": "application/json"}
	if values, ok := credentials["headers"].(map[string]interface{}); ok {
		for key, value := range values {
			headers[key] = fmt.Sprint(value)
		}
	}
	payload := map[string]interface{}{
		"type":        "seshat.notification",
		"app":         map[string]interface{}{"code": message.AppCode},
		"integration": map[string]interface{}{"id": message.IntegrationID, "name": message.IntegrationName},
		"event": map[string]interface{}{
			"id": message.EventID, "type": message.EventType, "title": message.Title,
			"summary": notificationBody(message), "severity": message.Severity, "received_at": message.ReceivedAt,
			"detail_url": message.DetailURL,
		},
	}
	return jsonNotificationRequest(endpoint, payload, headers, false, false)
}

func (webhookNotificationAdapter) ValidateResponse(_ int, _ []byte) error { return nil }

func isUnsafeNotificationHeader(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "host", "content-length", "transfer-encoding", "connection", "upgrade", "trailer":
		return true
	default:
		return false
	}
}
