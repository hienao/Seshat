package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	"seshat/internal/model"
)

type feishuNotificationAdapter struct{}

func (feishuNotificationAdapter) Type() string { return "feishu" }

func (feishuNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, nil, []string{"webhook_url", "signing_secret"})
}

func (feishuNotificationAdapter) Validate(_ map[string]interface{}, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "webhook_url") {
		return errors.New("飞书机器人 Webhook 地址不能为空")
	}
	return nil
}

func (a feishuNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (feishuNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	_, credentials := notificationChannelConfiguration(channel)
	endpoint, _ := credentials["webhook_url"].(string)
	payload := map[string]interface{}{"msg_type": "text", "content": map[string]interface{}{"text": notificationPlainText(message)}}
	if secret, _ := credentials["signing_secret"].(string); secret != "" {
		timestamp, signature := signFeishu(secret)
		payload["timestamp"] = timestamp
		payload["sign"] = signature
	}
	return jsonNotificationRequest(endpoint, payload, nil, true, false)
}

func (feishuNotificationAdapter) ValidateResponse(body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	if notificationResponseNumber(response, "code") != 0 && notificationResponseNumber(response, "errcode") != 0 && notificationResponseNumber(response, "StatusCode") != 0 {
		return notificationAPIFailure()
	}
	return nil
}

func signFeishu(secret string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(timestamp+"\n"+secret))
	return timestamp, base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
