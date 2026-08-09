package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"seshat/internal/model"
)

type wxPusherNotificationAdapter struct{}

func (wxPusherNotificationAdapter) Type() string { return "wxpusher" }

func (wxPusherNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, []string{"uids", "topic_ids"}, []string{"app_token"})
}

func (wxPusherNotificationAdapter) Validate(config, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "app_token") || (!requiredNotificationString(config, "uids") && !requiredNotificationString(config, "topic_ids")) {
		return errors.New("WxPusher AppToken 和至少一个 UID 或 Topic ID 不能为空")
	}
	return nil
}

func (a wxPusherNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (wxPusherNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	config, credentials := notificationChannelConfiguration(channel)
	topicIDs, err := commaSeparatedInts(fmt.Sprint(config["topic_ids"]))
	if err != nil {
		return nil, &deliveryError{message: "WxPusher Topic ID 配置无效"}
	}
	payload := map[string]interface{}{"appToken": credentials["app_token"], "content": notificationPlainText(message), "summary": truncateRunes(message.Title, 100), "contentType": 1, "uids": commaSeparatedStrings(fmt.Sprint(config["uids"])), "topicIds": topicIDs}
	return jsonNotificationRequest("https://wxpusher.zjiecode.com/api/send/message", payload, nil, true, true)
}

func (wxPusherNotificationAdapter) ValidateResponse(body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	if notificationResponseNumber(response, "code") != 1000 {
		return notificationAPIFailure()
	}
	return nil
}

func commaSeparatedStrings(value string) []string {
	parts := strings.FieldsFunc(value, func(character rune) bool { return character == ',' || character == ';' || character == '\n' })
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func commaSeparatedInts(value string) ([]int64, error) {
	stringsList := commaSeparatedStrings(value)
	result := make([]int64, 0, len(stringsList))
	for _, item := range stringsList {
		number, err := strconv.ParseInt(item, 10, 64)
		if err != nil || number <= 0 {
			return nil, errors.New("invalid integer list")
		}
		result = append(result, number)
	}
	return result, nil
}
