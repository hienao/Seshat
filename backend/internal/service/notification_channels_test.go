package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"seshat/internal/model"
)

func buildTestNotificationRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	adapter, ok := defaultNotificationChannelRegistry.Get(channel.Type)
	if !ok {
		return nil, errors.New("adapter not found")
	}
	httpAdapter, ok := adapter.(notificationHTTPChannelAdapter)
	if !ok {
		return nil, errors.New("adapter is not HTTP based")
	}
	message, err := prepareNotificationMessage(adapter, channel, message)
	if err != nil {
		return nil, err
	}
	return httpAdapter.BuildRequest(channel, message)
}

func validateTestNotificationResponse(channelType string, body []byte) error {
	adapter, ok := defaultNotificationChannelRegistry.Get(channelType)
	if !ok {
		return errors.New("adapter not found")
	}
	httpAdapter, ok := adapter.(notificationHTTPChannelAdapter)
	if !ok {
		return errors.New("adapter is not HTTP based")
	}
	return httpAdapter.ValidateResponse(http.StatusOK, body)
}

func TestAppriseRequiresSentResponseStatus(t *testing.T) {
	adapter, ok := defaultNotificationChannelRegistry.Get("apprise")
	if !ok {
		t.Fatal("Apprise adapter not registered")
	}
	httpAdapter := adapter.(notificationHTTPChannelAdapter)
	if err := httpAdapter.ValidateResponse(http.StatusOK, nil); err != nil {
		t.Fatalf("Apprise HTTP 200 should succeed: %v", err)
	}
	err := httpAdapter.ValidateResponse(http.StatusNoContent, nil)
	var sendErr *deliveryError
	if err == nil || !errors.As(err, &sendErr) || sendErr.retryable || sendErr.statusCode != http.StatusNoContent {
		t.Fatalf("Apprise HTTP 204 must be a permanent delivery failure: %v", err)
	}
}

func TestAppriseHTTP204FailsThroughTheSharedSender(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	channel := model.NotificationChannel{Type: "apprise", Config: []byte(`{"base_url":"` + server.URL + `"}`), SecretConfig: []byte(`{"config_id":"missing"}`)}
	err := sendChannel(t.Context(), &channel, outboundMessage{Title: "测试", Body: "通知"}, true, "")
	var sendErr *deliveryError
	if err == nil || !errors.As(err, &sendErr) || sendErr.statusCode != http.StatusNoContent {
		t.Fatalf("shared sender accepted Apprise HTTP 204: %v", err)
	}
}

func TestBarkSenderChecksBusinessResponse(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		if request.URL.Path != "/push" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = writer.Write([]byte(`{"code":200,"message":"success"}`))
			return
		}
		_, _ = writer.Write([]byte(`{"code":400,"message":"invalid device key"}`))
	}))
	defer server.Close()

	channel := model.NotificationChannel{Type: "bark", Config: []byte(`{"base_url":"` + server.URL + `"}`), SecretConfig: []byte(`{"device_key":"device-key"}`)}
	message := outboundMessage{Title: "测试", Body: "Bark 通知"}
	if err := sendChannel(t.Context(), &channel, message, true, ""); err != nil {
		t.Fatal(err)
	}
	err := sendChannel(t.Context(), &channel, message, true, "")
	var sendErr *deliveryError
	if err == nil || !errors.As(err, &sendErr) || sendErr.retryable {
		t.Fatalf("expected permanent Bark business error, got %v", err)
	}
}

func TestAdditionalNotificationChannelsPersistAndReturnCredentials(t *testing.T) {
	setupNotificationTestDB(t)
	service := NewNotificationService()
	tests := []struct {
		channelType string
		config      map[string]interface{}
		credentials map[string]interface{}
		secret      string
	}{
		{"email", map[string]interface{}{"smtp_host": "smtp.example.com", "smtp_port": 587, "encryption": "starttls", "from": "notice@example.com", "to": "one@example.com,two@example.com"}, map[string]interface{}{"username": "mailer", "password": "email-secret"}, "email-secret"},
		{"serverchan", map[string]interface{}{}, map[string]interface{}{"send_key": "SCT-server-secret"}, "SCT-server-secret"},
		{"bark", map[string]interface{}{"base_url": "https://api.day.app", "group": "Seshat"}, map[string]interface{}{"device_key": "bark-secret"}, "bark-secret"},
		{"dingtalk", map[string]interface{}{}, map[string]interface{}{"secret": "SEC-ding", "token": "ding-secret", "targets": "13800138000"}, "ding-secret"},
		{"feishu", map[string]interface{}{}, map[string]interface{}{"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/feishu-secret", "signing_secret": "SEC-feishu"}, "feishu-secret"},
		{"whatsapp", map[string]interface{}{"api_version": "v25.0"}, map[string]interface{}{"access_token": "whatsapp-secret", "phone_number_id": "123456", "recipient": "8613800000000"}, "whatsapp-secret"},
		{"wxpusher", map[string]interface{}{"uids": "UID_one", "topic_ids": "42"}, map[string]interface{}{"app_token": "AT-wxpusher-secret"}, "AT-wxpusher-secret"},
	}

	for _, test := range tests {
		t.Run(test.channelType, func(t *testing.T) {
			response, err := service.CreateChannel(7, &NotificationChannelRequest{
				Name:        "channel-" + test.channelType,
				Type:        test.channelType,
				Config:      test.config,
				Credentials: test.credentials,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !response.HasCredentials || response.Type != test.channelType {
				t.Fatalf("unexpected response: %+v", response)
			}
			encoded, _ := json.Marshal(response.Credentials)
			if !strings.Contains(string(encoded), test.secret) {
				t.Fatalf("response did not return %s credentials", test.channelType)
			}
		})
	}
}

func TestBuildAdditionalNotificationRequests(t *testing.T) {
	message := outboundMessage{
		Title:           strings.Repeat("新", 40),
		Body:            "收到一条业务消息",
		AppCode:         "jellyfin",
		IntegrationID:   2,
		IntegrationName: "家庭媒体库",
		EventID:         3,
		EventType:       "media_added",
		ReceivedAt:      time.Now(),
	}

	serverChan, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "serverchan", Config: []byte(`{}`), SecretConfig: []byte(`{"send_key":"SCT123"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	if serverChan.endpoint != "https://sctapi.ftqq.com/SCT123.send" || serverChan.headers["Content-Type"] != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected ServerChan request: %+v", serverChan)
	}
	serverChanForm, _ := url.ParseQuery(string(serverChan.body))
	if len([]rune(serverChanForm.Get("title"))) != 32 || serverChanForm.Get("desp") != message.Body {
		t.Fatalf("unexpected ServerChan payload: %s", serverChan.body)
	}

	bark, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "bark", Config: []byte(`{"base_url":"https://api.day.app","group":"Seshat","sound":"bell"}`), SecretConfig: []byte(`{"device_key":"device-key"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	var barkPayload map[string]interface{}
	_ = json.Unmarshal(bark.body, &barkPayload)
	if bark.endpoint != "https://api.day.app/push" || barkPayload["device_key"] != "device-key" || barkPayload["group"] != "Seshat" {
		t.Fatalf("unexpected Bark request: %s %s", bark.endpoint, bark.body)
	}

	dingTalk, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "dingtalk", Config: []byte(`{}`), SecretConfig: []byte(`{"secret":"SEC-test","token":"token","targets":"13800138000, 13900139000"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	dingTalkURL, _ := url.Parse(dingTalk.endpoint)
	var dingTalkPayload map[string]interface{}
	_ = json.Unmarshal(dingTalk.body, &dingTalkPayload)
	dingTalkAt, _ := dingTalkPayload["at"].(map[string]interface{})
	dingTalkTargets, _ := dingTalkAt["atMobiles"].([]interface{})
	if dingTalkURL.Scheme != "https" || dingTalkURL.Host != "oapi.dingtalk.com" || dingTalkURL.Path != "/robot/send" || dingTalkURL.Query().Get("access_token") != "token" || dingTalkURL.Query().Get("timestamp") == "" || dingTalkURL.Query().Get("sign") == "" || dingTalkPayload["msgtype"] != "markdown" || len(dingTalkTargets) != 2 {
		t.Fatalf("unexpected DingTalk request: %s %s", dingTalk.endpoint, dingTalk.body)
	}
	legacyDingTalk, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "dingtalk", Config: []byte(`{}`), SecretConfig: []byte(`{"webhook_url":"https://oapi.dingtalk.com/robot/send?access_token=legacy-token","signing_secret":"legacy-secret"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	legacyDingTalkURL, _ := url.Parse(legacyDingTalk.endpoint)
	if legacyDingTalkURL.Query().Get("access_token") != "legacy-token" || legacyDingTalkURL.Query().Get("timestamp") == "" || legacyDingTalkURL.Query().Get("sign") == "" {
		t.Fatalf("legacy DingTalk credentials were not migrated at send time: %s", legacyDingTalk.endpoint)
	}

	feishu, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "feishu", Config: []byte(`{}`), SecretConfig: []byte(`{"webhook_url":"https://open.feishu.cn/open-apis/bot/v2/hook/token","signing_secret":"secret"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	var feishuPayload map[string]interface{}
	_ = json.Unmarshal(feishu.body, &feishuPayload)
	if feishuPayload["timestamp"] == "" || feishuPayload["sign"] == "" || feishuPayload["msg_type"] != "text" {
		t.Fatalf("unexpected Feishu payload: %s", feishu.body)
	}

	whatsApp, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "whatsapp", Config: []byte(`{"api_version":"v25.0"}`), SecretConfig: []byte(`{"access_token":"access-token","phone_number_id":"1234","recipient":"8613800000000"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	if whatsApp.endpoint != "https://graph.facebook.com/v25.0/1234/messages" || whatsApp.headers["Authorization"] != "Bearer access-token" || !strings.Contains(string(whatsApp.body), `"messaging_product":"whatsapp"`) {
		t.Fatalf("unexpected WhatsApp request: %s %s", whatsApp.endpoint, whatsApp.body)
	}

	wxPusher, err := buildTestNotificationRequest(&model.NotificationChannel{Type: "wxpusher", Config: []byte(`{"uids":"UID_one,UID_two","topic_ids":"42,43"}`), SecretConfig: []byte(`{"app_token":"AT_test"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	var wxPusherPayload map[string]interface{}
	_ = json.Unmarshal(wxPusher.body, &wxPusherPayload)
	if wxPusher.endpoint != "https://wxpusher.zjiecode.com/api/send/message" || len(wxPusherPayload["topicIds"].([]interface{})) != 2 || len([]rune(wxPusherPayload["summary"].(string))) != 40 {
		t.Fatalf("unexpected WxPusher request: %s %s", wxPusher.endpoint, wxPusher.body)
	}
}

func TestNotificationAPIResponseValidation(t *testing.T) {
	tests := []struct {
		channelType string
		body        string
		wantError   bool
	}{
		{"telegram", `{"ok":true,"result":{}}`, false},
		{"telegram", `{"ok":false,"description":"bad request"}`, true},
		{"serverchan", `{"code":0}`, false},
		{"dingtalk", `{"errcode":0}`, false},
		{"feishu", `{"StatusCode":0}`, false},
		{"bark", `{"code":200}`, false},
		{"whatsapp", `{"messages":[{"id":"message-id"}]}`, false},
		{"wxpusher", `{"code":1000,"msg":"处理成功"}`, false},
		{"wxpusher", `{"code":1001,"msg":"失败"}`, true},
		{"bark", `not-json`, true},
	}
	for _, test := range tests {
		t.Run(test.channelType+test.body, func(t *testing.T) {
			err := validateTestNotificationResponse(test.channelType, []byte(test.body))
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestNotificationChannelRegistryKeepsAdaptersIndependent(t *testing.T) {
	expected := []string{"apprise", "bark", "dingtalk", "email", "feishu", "serverchan", "telegram", "webhook", "whatsapp", "wxpusher"}
	actual := defaultNotificationChannelRegistry.Types()
	if strings.Join(actual, ",") != strings.Join(expected, ",") {
		t.Fatalf("registered adapters = %v, want %v", actual, expected)
	}
	telegram, _ := defaultNotificationChannelRegistry.Get("telegram")
	if _, _, err := telegram.Sanitize(map[string]interface{}{"base_url": "https://example.com"}, map[string]interface{}{}); err == nil {
		t.Fatal("Telegram adapter accepted an Apprise-only field")
	}
	bark, _ := defaultNotificationChannelRegistry.Get("bark")
	if _, _, err := bark.Sanitize(map[string]interface{}{}, map[string]interface{}{"bot_token": "secret"}); err == nil {
		t.Fatal("Bark adapter accepted a Telegram-only credential")
	}
}

func TestDingTalkAdapterOwnsTokenSecretAndTargets(t *testing.T) {
	adapter, ok := defaultNotificationChannelRegistry.Get("dingtalk")
	if !ok {
		t.Fatal("DingTalk adapter not registered")
	}
	config, credentials, err := adapter.Sanitize(
		map[string]interface{}{"message_format": "markdown", "include_image": true},
		map[string]interface{}{"secret": "SEC-test", "token": "token", "targets": "13800138000,13900139000"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(config) != 2 || config["message_format"] != "markdown" || config["include_image"] != true || len(credentials) != 3 || credentials["secret"] != "SEC-test" || credentials["token"] != "token" || credentials["targets"] != "13800138000,13900139000" {
		t.Fatalf("unexpected DingTalk sanitized configuration: config=%v credentials=%v", config, credentials)
	}
	if err := adapter.Validate(config, credentials); err != nil {
		t.Fatal(err)
	}
	if _, _, err := adapter.Sanitize(map[string]interface{}{}, map[string]interface{}{"webhook_url": dingTalkRobotEndpoint + "?access_token=token"}); err == nil {
		t.Fatal("DingTalk adapter accepted the legacy webhook_url field for a new configuration")
	}
	if err := adapter.Validate(map[string]interface{}{}, map[string]interface{}{"token": "token", "targets": "invalid"}); err == nil {
		t.Fatal("DingTalk adapter accepted an invalid target")
	}
	if err := adapter.Validate(map[string]interface{}{}, map[string]interface{}{"token": "", "targets": ""}); err == nil {
		t.Fatal("DingTalk adapter accepted an empty token")
	}
	if err := adapter.Validate(map[string]interface{}{"message_format": "html"}, map[string]interface{}{"token": "token"}); err == nil {
		t.Fatal("DingTalk adapter accepted an unsupported message format")
	}
}

func TestEveryNotificationChannelAdapterOwnsItsConfiguration(t *testing.T) {
	tests := []struct {
		channelType string
		config      map[string]interface{}
		credentials map[string]interface{}
	}{
		{"webhook", map[string]interface{}{}, map[string]interface{}{"url": "https://example.com/notify"}},
		{"telegram", map[string]interface{}{"chat_id": "42"}, map[string]interface{}{"bot_token": "token"}},
		{"apprise", map[string]interface{}{"base_url": "https://apprise.example.com"}, map[string]interface{}{"config_id": "main"}},
		{"email", map[string]interface{}{"smtp_host": "smtp.example.com", "smtp_port": 587, "from": "notice@example.com", "to": "user@example.com"}, map[string]interface{}{}},
		{"serverchan", map[string]interface{}{}, map[string]interface{}{"send_key": "SCT123"}},
		{"bark", map[string]interface{}{"base_url": "https://api.day.app"}, map[string]interface{}{"device_key": "device"}},
		{"dingtalk", map[string]interface{}{"message_format": "auto", "include_image": true}, map[string]interface{}{"secret": "secret", "token": "token", "targets": "13800138000"}},
		{"feishu", map[string]interface{}{}, map[string]interface{}{"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/token"}},
		{"whatsapp", map[string]interface{}{"api_version": "v25.0"}, map[string]interface{}{"access_token": "token", "phone_number_id": "123", "recipient": "8613800000000"}},
		{"wxpusher", map[string]interface{}{"uids": "UID_one"}, map[string]interface{}{"app_token": "AT_token"}},
	}
	for _, test := range tests {
		t.Run(test.channelType, func(t *testing.T) {
			adapter, ok := defaultNotificationChannelRegistry.Get(test.channelType)
			if !ok {
				t.Fatal("adapter not registered")
			}
			config, credentials, err := adapter.Sanitize(test.config, test.credentials)
			if err != nil {
				t.Fatal(err)
			}
			if err := adapter.Validate(config, credentials); err != nil {
				t.Fatal(err)
			}
			foreignConfig := cloneNotificationTestValues(test.config)
			foreignConfig["foreign_channel_field"] = true
			if _, _, err := adapter.Sanitize(foreignConfig, test.credentials); err == nil {
				t.Fatal("adapter accepted a foreign config field")
			}
			foreignCredentials := cloneNotificationTestValues(test.credentials)
			foreignCredentials["foreign_channel_secret"] = "secret"
			if _, _, err := adapter.Sanitize(test.config, foreignCredentials); err == nil {
				t.Fatal("adapter accepted a foreign credential field")
			}
		})
	}
}

func cloneNotificationTestValues(values map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(values)+1)
	for key, value := range values {
		result[key] = value
	}
	return result
}

func TestNotificationChannelInputValidationEdges(t *testing.T) {
	for _, value := range []string{"v25.0", "v1.2"} {
		if !validGraphAPIVersion(value) {
			t.Fatalf("valid Graph API version rejected: %s", value)
		}
	}
	for _, value := range []string{"25.0", "v.1", "v1.", "v1.2.3", "v01.2"} {
		if validGraphAPIVersion(value) {
			t.Fatalf("invalid Graph API version accepted: %s", value)
		}
	}
	if _, err := notificationSMTPPort(map[string]interface{}{"smtp_port": 587.5}); err == nil {
		t.Fatal("fractional SMTP port should be rejected")
	}
	endpoint, err := serverChanEndpoint("sctp123tsecret")
	if err != nil || endpoint != "https://123.push.ft07.com/send/sctp123tsecret.send" {
		t.Fatalf("unexpected ServerChan3 endpoint: %q, %v", endpoint, err)
	}
}
