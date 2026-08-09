package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"seshat/internal/model"
)

type dingTalkNotificationAdapter struct{}

func (dingTalkNotificationAdapter) Type() string { return "dingtalk" }

func (dingTalkNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, nil, []string{"webhook_url", "signing_secret"})
}

func (dingTalkNotificationAdapter) Validate(_ map[string]interface{}, credentials map[string]interface{}) error {
	if !requiredNotificationString(credentials, "webhook_url") {
		return errors.New("钉钉机器人 Webhook 地址不能为空")
	}
	return nil
}

func (a dingTalkNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (dingTalkNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	_, credentials := notificationChannelConfiguration(channel)
	endpoint, _ := credentials["webhook_url"].(string)
	if secret, _ := credentials["signing_secret"].(string); secret != "" {
		endpoint = signDingTalkURL(endpoint, secret)
	}
	payload := map[string]interface{}{"msgtype": "markdown", "markdown": map[string]interface{}{"title": singleLineTitle(message.Title), "text": notificationDingTalkMarkdown(message)}}
	return jsonNotificationRequest(endpoint, payload, nil, true, false)
}

func (dingTalkNotificationAdapter) ValidateResponse(_ int, body []byte) error {
	response, err := parseNotificationResponse(body)
	if err != nil {
		return err
	}
	if notificationResponseNumber(response, "code") != 0 && notificationResponseNumber(response, "errcode") != 0 && notificationResponseNumber(response, "StatusCode") != 0 {
		return notificationAPIFailure()
	}
	return nil
}

func signDingTalkURL(endpoint, secret string) string {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", signature)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func notificationDingTalkMarkdown(message outboundMessage) string {
	content := "### " + escapeDingTalkMarkdown(message.Title)
	if body := strings.TrimSpace(message.Body); body != "" {
		content += "\n\n" + escapeDingTalkMarkdown(body)
	}
	if message.DetailURL != "" {
		content += "\n\n[查看消息详情](" + message.DetailURL + ")"
	}
	return content
}

func singleLineTitle(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
}

func escapeDingTalkMarkdown(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "{", "\\{", "}", "\\}",
		"[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "#", "\\#", "+", "\\+",
		"-", "\\-", ".", "\\.", "!", "\\!", "|", "\\|", ">", "\\>",
	)
	return replacer.Replace(value)
}
