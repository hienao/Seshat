package service

import (
	"context"
	"errors"
	"strings"

	"seshat/internal/model"
)

type barkNotificationAdapter struct{}

func (barkNotificationAdapter) Type() string { return "bark" }

func (barkNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, []string{"base_url", "group", "sound"}, []string{"device_key"})
}

func (barkNotificationAdapter) Validate(config, credentials map[string]interface{}) error {
	if !requiredNotificationString(config, "base_url") || !requiredNotificationString(credentials, "device_key") {
		return errors.New("Bark 服务地址和 Device Key 不能为空")
	}
	return nil
}

func (a barkNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (barkNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	config, credentials := notificationChannelConfiguration(channel)
	baseURL, _ := config["base_url"].(string)
	payload := map[string]interface{}{"device_key": credentials["device_key"], "title": message.Title, "body": notificationBody(message)}
	if group, _ := config["group"].(string); group != "" {
		payload["group"] = group
	}
	if sound, _ := config["sound"].(string); sound != "" {
		payload["sound"] = sound
	}
	return jsonNotificationRequest(strings.TrimRight(baseURL, "/")+"/push", payload, nil, true, false)
}

func (barkNotificationAdapter) ValidateResponse(_ int, body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	if notificationResponseNumber(response, "code") != 200 {
		return notificationAPIFailure()
	}
	return nil
}
