package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"seshat/internal/model"
)

type appriseNotificationAdapter struct{}

func (appriseNotificationAdapter) Type() string { return "apprise" }

func (appriseNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, []string{"base_url", "tag"}, []string{"config_id"})
}

func (appriseNotificationAdapter) Validate(config, credentials map[string]interface{}) error {
	if !requiredNotificationString(config, "base_url") {
		return errors.New("Apprise Base URL 不能为空")
	}
	configID, _ := credentials["config_id"].(string)
	if !validAppriseConfigID(configID) {
		return errors.New("Apprise Config ID 必须为 1 到 128 位字母、数字、下划线或连字符")
	}
	return nil
}

func (a appriseNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (appriseNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	config, credentials := notificationChannelConfiguration(channel)
	baseURL, _ := config["base_url"].(string)
	configID, _ := credentials["config_id"].(string)
	endpoint := strings.TrimRight(baseURL, "/") + "/notify/" + url.PathEscape(configID)
	payload := map[string]interface{}{"title": message.Title, "body": notificationBody(message), "type": appriseSeverity(message.Severity), "format": "text"}
	if tag, _ := config["tag"].(string); strings.TrimSpace(tag) != "" {
		payload["tag"] = tag
	}
	return jsonNotificationRequest(endpoint, payload, nil, true, false)
}

func (appriseNotificationAdapter) ValidateResponse(_ []byte) error { return nil }

func validAppriseConfigID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func appriseSeverity(value string) string {
	switch strings.ToLower(value) {
	case "success":
		return "success"
	case "warning", "warn":
		return "warning"
	case "error", "critical":
		return "failure"
	default:
		return "info"
	}
}
