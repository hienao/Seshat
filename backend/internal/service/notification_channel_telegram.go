package service

import (
	"context"
	"errors"
	"fmt"

	"seshat/internal/model"
)

type telegramNotificationAdapter struct{}

func (telegramNotificationAdapter) Type() string { return "telegram" }

func (telegramNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, []string{"chat_id", "message_thread_id", "silent"}, []string{"bot_token"})
}

func (telegramNotificationAdapter) Validate(config, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "bot_token") || !requiredNotificationString(config, "chat_id") {
		return errors.New("Telegram Bot Token 和 Chat ID 不能为空")
	}
	return nil
}

func (a telegramNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (telegramNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	config, credentials := notificationChannelConfiguration(channel)
	endpoint := "https://api.telegram.org/bot" + fmt.Sprint(credentials["bot_token"]) + "/sendMessage"
	payload := map[string]interface{}{"chat_id": config["chat_id"], "text": notificationPlainText(message), "disable_notification": config["silent"]}
	if threadID := config["message_thread_id"]; threadID != nil && fmt.Sprint(threadID) != "" {
		payload["message_thread_id"] = threadID
	}
	return jsonNotificationRequest(endpoint, payload, nil, true, true)
}

func (telegramNotificationAdapter) ValidateResponse(_ int, body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	success, _ := response["ok"].(bool)
	if !success {
		return notificationAPIFailure()
	}
	return nil
}
