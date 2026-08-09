package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"seshat/internal/model"
)

type whatsAppNotificationAdapter struct{}

func (whatsAppNotificationAdapter) Type() string { return "whatsapp" }

func (whatsAppNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	cleanConfig, cleanCredentials, err := sanitizeNotificationChannelFields(config, credentials, []string{"api_version"}, []string{"access_token", "phone_number_id", "recipient"})
	if err != nil {
		return nil, nil, err
	}
	version, _ := cleanConfig["api_version"].(string)
	if version == "" {
		cleanConfig["api_version"] = "v25.0"
	} else if !validGraphAPIVersion(version) {
		return nil, nil, errors.New("WhatsApp Graph API 版本格式无效")
	}
	return cleanConfig, cleanCredentials, nil
}

func (whatsAppNotificationAdapter) Validate(_ map[string]interface{}, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "access_token") || !requiredNotificationString(credentials, "phone_number_id") || !requiredNotificationString(credentials, "recipient") {
		return errors.New("WhatsApp Access Token、Phone Number ID 和收件号码不能为空")
	}
	return nil
}

func (a whatsAppNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (whatsAppNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	config, credentials := notificationChannelConfiguration(channel)
	endpoint := "https://graph.facebook.com/" + fmt.Sprint(config["api_version"]) + "/" + url.PathEscape(fmt.Sprint(credentials["phone_number_id"])) + "/messages"
	headers := map[string]string{"Content-Type": "application/json", "Accept": "application/json", "Authorization": "Bearer " + fmt.Sprint(credentials["access_token"])}
	payload := map[string]interface{}{"messaging_product": "whatsapp", "recipient_type": "individual", "to": credentials["recipient"], "type": "text", "text": map[string]interface{}{"preview_url": false, "body": notificationPlainText(message)}}
	return jsonNotificationRequest(endpoint, payload, headers, true, true)
}

func (whatsAppNotificationAdapter) ValidateResponse(body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	messages, _ := response["messages"].([]interface{})
	if len(messages) == 0 || response["error"] != nil {
		return notificationAPIFailure()
	}
	return nil
}

func validGraphAPIVersion(value string) bool {
	if len(value) < 4 || value[0] != 'v' {
		return false
	}
	parts := strings.Split(value[1:], ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, part := range parts {
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
		if len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}
