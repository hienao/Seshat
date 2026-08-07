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

func TestAdditionalNotificationChannelsPersistWithoutExposingCredentials(t *testing.T) {
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
		{"dingtalk", map[string]interface{}{}, map[string]interface{}{"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=ding-secret", "signing_secret": "SEC-ding"}, "ding-secret"},
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
			encoded, _ := json.Marshal(response)
			if strings.Contains(string(encoded), test.secret) {
				t.Fatalf("response exposed %s credentials", test.channelType)
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

	serverChan, err := buildNotificationHTTPRequest(&model.NotificationChannel{Type: "serverchan", Config: []byte(`{}`), SecretConfig: []byte(`{"send_key":"SCT123"}`)}, message)
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

	bark, err := buildNotificationHTTPRequest(&model.NotificationChannel{Type: "bark", Config: []byte(`{"base_url":"https://api.day.app","group":"Seshat","sound":"bell"}`), SecretConfig: []byte(`{"device_key":"device-key"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	var barkPayload map[string]interface{}
	_ = json.Unmarshal(bark.body, &barkPayload)
	if bark.endpoint != "https://api.day.app/push" || barkPayload["device_key"] != "device-key" || barkPayload["group"] != "Seshat" {
		t.Fatalf("unexpected Bark request: %s %s", bark.endpoint, bark.body)
	}

	dingTalk, err := buildNotificationHTTPRequest(&model.NotificationChannel{Type: "dingtalk", Config: []byte(`{}`), SecretConfig: []byte(`{"webhook_url":"https://oapi.dingtalk.com/robot/send?access_token=token","signing_secret":"SEC-test"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	dingTalkURL, _ := url.Parse(dingTalk.endpoint)
	if dingTalkURL.Query().Get("timestamp") == "" || dingTalkURL.Query().Get("sign") == "" || !strings.Contains(string(dingTalk.body), `"msgtype":"markdown"`) {
		t.Fatalf("unexpected DingTalk request: %s %s", dingTalk.endpoint, dingTalk.body)
	}

	feishu, err := buildNotificationHTTPRequest(&model.NotificationChannel{Type: "feishu", Config: []byte(`{}`), SecretConfig: []byte(`{"webhook_url":"https://open.feishu.cn/open-apis/bot/v2/hook/token","signing_secret":"secret"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	var feishuPayload map[string]interface{}
	_ = json.Unmarshal(feishu.body, &feishuPayload)
	if feishuPayload["timestamp"] == "" || feishuPayload["sign"] == "" || feishuPayload["msg_type"] != "text" {
		t.Fatalf("unexpected Feishu payload: %s", feishu.body)
	}

	whatsApp, err := buildNotificationHTTPRequest(&model.NotificationChannel{Type: "whatsapp", Config: []byte(`{"api_version":"v25.0"}`), SecretConfig: []byte(`{"access_token":"access-token","phone_number_id":"1234","recipient":"8613800000000"}`)}, message)
	if err != nil {
		t.Fatal(err)
	}
	if whatsApp.endpoint != "https://graph.facebook.com/v25.0/1234/messages" || whatsApp.headers["Authorization"] != "Bearer access-token" || !strings.Contains(string(whatsApp.body), `"messaging_product":"whatsapp"`) {
		t.Fatalf("unexpected WhatsApp request: %s %s", whatsApp.endpoint, whatsApp.body)
	}

	wxPusher, err := buildNotificationHTTPRequest(&model.NotificationChannel{Type: "wxpusher", Config: []byte(`{"uids":"UID_one,UID_two","topic_ids":"42,43"}`), SecretConfig: []byte(`{"app_token":"AT_test"}`)}, message)
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
			err := validateNotificationAPIResponse(test.channelType, []byte(test.body))
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
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
