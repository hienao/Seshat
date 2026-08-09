package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"seshat/internal/model"
)

type notificationSendOptions struct {
	allowPrivate bool
	proxyURL     string
}

type notificationRequestSpec struct {
	endpoint          string
	body              []byte
	headers           map[string]string
	lockedHost        string
	requirePublicHost bool
}

type notificationChannelAdapter interface {
	Type() string
	Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error)
	Validate(config, credentials map[string]interface{}) error
	Send(context.Context, *model.NotificationChannel, outboundMessage, notificationSendOptions) error
}

type notificationHTTPChannelAdapter interface {
	notificationChannelAdapter
	BuildRequest(*model.NotificationChannel, outboundMessage) (*notificationRequestSpec, error)
	ValidateResponse(int, []byte) error
}

type notificationChannelRegistry struct {
	adapters map[string]notificationChannelAdapter
}

func newNotificationChannelRegistry() *notificationChannelRegistry {
	registry := &notificationChannelRegistry{adapters: make(map[string]notificationChannelAdapter)}
	registry.Register(webhookNotificationAdapter{})
	registry.Register(telegramNotificationAdapter{})
	registry.Register(appriseNotificationAdapter{})
	registry.Register(emailNotificationAdapter{})
	registry.Register(serverChanNotificationAdapter{})
	registry.Register(barkNotificationAdapter{})
	registry.Register(dingTalkNotificationAdapter{})
	registry.Register(feishuNotificationAdapter{})
	registry.Register(whatsAppNotificationAdapter{})
	registry.Register(wxPusherNotificationAdapter{})
	return registry
}

func (r *notificationChannelRegistry) Register(adapter notificationChannelAdapter) {
	if adapter == nil || strings.TrimSpace(adapter.Type()) == "" {
		panic("notification channel adapter type is required")
	}
	channelType := strings.ToLower(strings.TrimSpace(adapter.Type()))
	if _, exists := r.adapters[channelType]; exists {
		panic("duplicate notification channel adapter: " + channelType)
	}
	r.adapters[channelType] = adapter
}

func (r *notificationChannelRegistry) Get(channelType string) (notificationChannelAdapter, bool) {
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(channelType))]
	return adapter, ok
}

func (r *notificationChannelRegistry) Types() []string {
	result := make([]string, 0, len(r.adapters))
	for channelType := range r.adapters {
		result = append(result, channelType)
	}
	sort.Strings(result)
	return result
}

var defaultNotificationChannelRegistry = newNotificationChannelRegistry()

func sanitizeNotificationChannelFields(config, credentials map[string]interface{}, publicKeys, secretKeys []string) (map[string]interface{}, map[string]interface{}, error) {
	allowedPublic := make(map[string]bool, len(publicKeys)+1)
	allowedPublic["use_proxy"] = true
	for _, key := range publicKeys {
		allowedPublic[key] = true
	}
	allowedSecret := make(map[string]bool, len(secretKeys))
	for _, key := range secretKeys {
		allowedSecret[key] = true
	}
	cleanPublic := make(map[string]interface{}, len(config))
	for key, value := range config {
		if !allowedPublic[key] {
			return nil, nil, errors.New("包含不支持的渠道配置项: " + key)
		}
		cleanPublic[key] = value
	}
	if useProxy, ok := cleanPublic["use_proxy"]; ok {
		if _, valid := useProxy.(bool); !valid {
			return nil, nil, errors.New("渠道代理开关必须是布尔值")
		}
	}
	cleanSecret := make(map[string]interface{}, len(credentials))
	for key, value := range credentials {
		if !allowedSecret[key] {
			return nil, nil, errors.New("包含不支持的渠道凭据项: " + key)
		}
		cleanSecret[key] = value
	}
	return cleanPublic, cleanSecret, nil
}

func notificationChannelConfiguration(channel *model.NotificationChannel) (map[string]interface{}, map[string]interface{}) {
	config, secret := map[string]interface{}{}, map[string]interface{}{}
	_ = json.Unmarshal(channel.Config, &config)
	_ = json.Unmarshal(channel.SecretConfig, &secret)
	return config, secret
}

func requiredNotificationString(values map[string]interface{}, key string) bool {
	value, ok := values[key].(string)
	return ok && strings.TrimSpace(value) != ""
}

func jsonNotificationRequest(endpoint string, payload interface{}, headers map[string]string, lockHost, requirePublicHost bool) (*notificationRequestSpec, error) {
	if headers == nil {
		headers = map[string]string{"Content-Type": "application/json", "Accept": "application/json"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &deliveryError{message: "无法生成推送内容"}
	}
	return notificationRequest(endpoint, body, headers, lockHost, requirePublicHost), nil
}

func notificationRequest(endpoint string, body []byte, headers map[string]string, lockHost, requirePublicHost bool) *notificationRequestSpec {
	spec := &notificationRequestSpec{endpoint: endpoint, body: body, headers: headers, requirePublicHost: requirePublicHost}
	if lockHost {
		if parsed, err := url.Parse(endpoint); err == nil {
			spec.lockedHost = parsed.Hostname()
		}
	}
	return spec
}

func sendHTTPNotificationAdapter(ctx context.Context, adapter notificationHTTPChannelAdapter, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	spec, err := adapter.BuildRequest(channel, message)
	if err != nil {
		return err
	}
	return sendNotificationHTTPRequest(ctx, spec, options, adapter.ValidateResponse)
}

func notificationPlainText(message outboundMessage) string {
	body := notificationBody(message)
	if body == "" {
		return message.Title
	}
	return message.Title + "\n\n" + body
}

func notificationBody(message outboundMessage) string {
	body := strings.TrimSpace(message.Body)
	if message.DetailURL == "" {
		return body
	}
	if body == "" {
		return "消息详情：" + message.DetailURL
	}
	return body + "\n\n消息详情：" + message.DetailURL
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func parseNotificationResponse(body []byte) (map[string]interface{}, error) {
	var response map[string]interface{}
	if len(body) == 0 || json.Unmarshal(body, &response) != nil {
		return nil, &deliveryError{message: "推送渠道返回了无法识别的响应"}
	}
	return response, nil
}

func notificationResponseNumber(response map[string]interface{}, key string) int64 {
	switch value := response[key].(type) {
	case float64:
		return int64(value)
	case string:
		var result int64
		_, _ = fmt.Sscan(value, &result)
		return result
	default:
		return -1
	}
}

func notificationAPIFailure() error {
	return &deliveryError{message: "推送渠道 API 返回失败状态"}
}
