package service

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"seshat/internal/model"
)

type serverChanNotificationAdapter struct{}

func (serverChanNotificationAdapter) Type() string { return "serverchan" }

func (serverChanNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, nil, []string{"send_key"})
}

func (serverChanNotificationAdapter) Validate(_ map[string]interface{}, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "send_key") {
		return errors.New("Server酱 SendKey 不能为空")
	}
	return nil
}

func (a serverChanNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (serverChanNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	_, credentials := notificationChannelConfiguration(channel)
	sendKey, _ := credentials["send_key"].(string)
	endpoint, err := serverChanEndpoint(sendKey)
	if err != nil {
		return nil, &deliveryError{message: err.Error()}
	}
	form := url.Values{"title": []string{truncateRunes(message.Title, 32)}, "desp": []string{notificationBody(message)}}
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"}
	return notificationRequest(endpoint, []byte(form.Encode()), headers, true, true), nil
}

func (serverChanNotificationAdapter) ValidateResponse(_ int, body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	if notificationResponseNumber(response, "code") != 0 && notificationResponseNumber(response, "errcode") != 0 && notificationResponseNumber(response, "StatusCode") != 0 {
		return notificationAPIFailure()
	}
	return nil
}

func serverChanEndpoint(sendKey string) (string, error) {
	if strings.HasPrefix(sendKey, "sctp") {
		remainder := strings.TrimPrefix(sendKey, "sctp")
		separator := strings.IndexByte(remainder, 't')
		if separator <= 0 {
			return "", errors.New("Server酱³ SendKey 格式无效")
		}
		uid := remainder[:separator]
		if _, err := strconv.ParseUint(uid, 10, 64); err != nil {
			return "", errors.New("Server酱³ SendKey 格式无效")
		}
		return "https://" + uid + ".push.ft07.com/send/" + url.PathEscape(sendKey) + ".send", nil
	}
	return "https://sctapi.ftqq.com/" + url.PathEscape(sendKey) + ".send", nil
}
