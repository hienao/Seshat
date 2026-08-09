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

const dingTalkRobotEndpoint = "https://oapi.dingtalk.com/robot/send"

func (dingTalkNotificationAdapter) Type() string { return "dingtalk" }

func (dingTalkNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	return sanitizeNotificationChannelFields(config, credentials, []string{"message_format", "include_image"}, []string{"secret", "token", "targets"})
}

func (dingTalkNotificationAdapter) Validate(config map[string]interface{}, credentials map[string]interface{}) error {
	if configured, ok := config["message_format"]; ok {
		format, valid := configured.(string)
		if !valid || (format != "" && format != "auto" && format != string(notificationFormatMarkdown) && format != string(notificationFormatPlainText)) {
			return errors.New("钉钉消息格式无效")
		}
	}
	if includeImage, ok := config["include_image"]; ok {
		if _, valid := includeImage.(bool); !valid {
			return errors.New("钉钉图片开关必须是布尔值")
		}
	}
	if !requiredNotificationString(credentials, "token") {
		return errors.New("钉钉机器人 Token 不能为空")
	}
	if _, ok := credentials["secret"].(string); !ok && credentials["secret"] != nil {
		return errors.New("钉钉机器人 Secret 必须是字符串")
	}
	targets, ok := credentials["targets"].(string)
	if !ok && credentials["targets"] != nil {
		return errors.New("钉钉机器人 Targets 必须是字符串")
	}
	if _, err := parseDingTalkTargets(targets); err != nil {
		return err
	}
	return nil
}

func (dingTalkNotificationAdapter) ContentPolicy(config map[string]interface{}) notificationContentPolicy {
	preferred, _ := config["message_format"].(string)
	if preferred == "auto" {
		preferred = ""
	}
	includeImage := true
	if configured, ok := config["include_image"].(bool); ok {
		includeImage = configured
	}
	return notificationContentPolicy{
		Formats:         []notificationFormat{notificationFormatMarkdown, notificationFormatPlainText},
		DefaultFormat:   notificationFormatMarkdown,
		PreferredFormat: notificationFormat(preferred),
		Profile:         "dingtalk_robot",
		IncludeImage:    includeImage,
	}
}

func (a dingTalkNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	return sendHTTPNotificationAdapter(ctx, a, channel, message, options)
}

func (dingTalkNotificationAdapter) BuildRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	_, credentials := notificationChannelConfiguration(channel)
	token, _ := credentials["token"].(string)
	secret, _ := credentials["secret"].(string)
	targetValue, _ := credentials["targets"].(string)
	if strings.TrimSpace(token) == "" {
		token = dingTalkLegacyToken(credentials)
	}
	if strings.TrimSpace(secret) == "" {
		secret, _ = credentials["signing_secret"].(string)
	}
	if strings.TrimSpace(token) == "" {
		return nil, &deliveryError{message: "钉钉机器人 Token 不能为空"}
	}
	targets, err := parseDingTalkTargets(targetValue)
	if err != nil {
		return nil, &deliveryError{message: err.Error()}
	}
	endpoint := dingTalkRobotEndpoint + "?access_token=" + url.QueryEscape(strings.TrimSpace(token))
	if secret = strings.TrimSpace(secret); secret != "" {
		endpoint = signDingTalkURL(endpoint, secret)
	}
	if message.Rendered == nil {
		return nil, &deliveryError{message: "钉钉推送内容尚未渲染"}
	}
	payload := map[string]interface{}{"at": map[string]interface{}{"atMobiles": targets, "isAtAll": false}}
	switch message.Rendered.Format {
	case notificationFormatMarkdown:
		payload["msgtype"] = "markdown"
		payload["markdown"] = map[string]interface{}{"title": singleLineTitle(message.Title), "text": message.Body}
	case notificationFormatPlainText:
		payload["msgtype"] = "text"
		payload["text"] = map[string]interface{}{"content": notificationPlainText(message)}
	default:
		return nil, &deliveryError{message: "钉钉不支持当前消息格式"}
	}
	return jsonNotificationRequest(endpoint, payload, nil, true, true)
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

func dingTalkLegacyToken(credentials map[string]interface{}) string {
	legacyURL, _ := credentials["webhook_url"].(string)
	parsed, err := url.Parse(strings.TrimSpace(legacyURL))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Query().Get("access_token"))
}

func parseDingTalkTargets(value string) ([]string, error) {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' })
	targets := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		target := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, part)
		if len(target) < 11 || len(target) > 14 {
			return nil, errors.New("钉钉机器人 Targets 必须是 11 到 14 位手机号，多个号码使用逗号分隔")
		}
		if !seen[target] {
			targets = append(targets, target)
			seen[target] = true
		}
	}
	return targets, nil
}

func singleLineTitle(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
}
