package service

import (
	"encoding/json"
	"strings"
	"testing"

	"seshat/internal/model"
	"seshat/internal/webhook"
)

func TestNotificationDocumentUsesNormalizedPresentation(t *testing.T) {
	presentation := &webhook.Presentation{
		Title:   "Jellyfin · 新增媒体 · 示例剧集",
		Summary: "示例剧集已加入媒体库",
		Facts: []map[string]string{
			{"label": "类型", "value": "Episode"},
			{"label": "", "value": "ignored"},
		},
		Links: []map[string]string{{"label": "打开服务", "url": "https://media.example.com/item/1"}},
		Data: map[string]interface{}{"media": map[string]interface{}{
			"display_name": "示例剧集",
			"overview":     "一段媒体简介",
			"image_url":    "/api/public/media-images/image-token",
		}},
	}
	message := outboundMessage{
		Title:         "fallback",
		Body:          "fallback summary",
		DetailURL:     "https://seshat.example.com/public/events/event-token",
		PublicBaseURL: "https://seshat.example.com/base",
		Presentation:  presentation,
	}
	document := buildNotificationDocument(message, true)
	if document.Title != presentation.Title || document.Summary != presentation.Summary || document.Overview != "一段媒体简介" {
		t.Fatalf("unexpected normalized document: %+v", document)
	}
	if document.ImageURL != "https://seshat.example.com/api/public/media-images/image-token" {
		t.Fatalf("relative public image was not resolved: %q", document.ImageURL)
	}
	if len(document.Facts) != 1 || len(document.Actions) != 2 {
		t.Fatalf("facts/actions were not normalized: %+v", document)
	}
}

func TestDingTalkContentNegotiationAndImagePolicy(t *testing.T) {
	presentation := &webhook.Presentation{
		Title:   "标题 *需要转义*",
		Summary: "摘要",
		Data: map[string]interface{}{"media": map[string]interface{}{
			"display_name": "图片",
			"image_url":    "https://images.example.com/poster.jpg",
		}},
	}
	message := outboundMessage{Title: "fallback", Presentation: presentation}

	markdownChannel := &model.NotificationChannel{Type: "dingtalk", Config: []byte(`{"message_format":"auto","include_image":true}`), SecretConfig: []byte(`{"token":"token"}`)}
	markdownRequest, err := buildTestNotificationRequest(markdownChannel, message)
	if err != nil {
		t.Fatal(err)
	}
	var markdownPayload map[string]interface{}
	_ = json.Unmarshal(markdownRequest.body, &markdownPayload)
	markdown, _ := markdownPayload["markdown"].(map[string]interface{})
	markdownText, _ := markdown["text"].(string)
	if markdownPayload["msgtype"] != "markdown" || !strings.Contains(markdownText, `![图片](https://images.example.com/poster.jpg)`) || !strings.Contains(markdownText, `\*需要转义\*`) {
		t.Fatalf("unexpected Markdown payload: %s", markdownRequest.body)
	}

	plainChannel := &model.NotificationChannel{Type: "dingtalk", Config: []byte(`{"message_format":"plain_text","include_image":false}`), SecretConfig: []byte(`{"token":"token"}`)}
	plainRequest, err := buildTestNotificationRequest(plainChannel, message)
	if err != nil {
		t.Fatal(err)
	}
	var plainPayload map[string]interface{}
	_ = json.Unmarshal(plainRequest.body, &plainPayload)
	if plainPayload["msgtype"] != "text" || plainPayload["markdown"] != nil {
		t.Fatalf("unexpected plain-text payload: %s", plainRequest.body)
	}
}

func TestNotificationImageRejectsUnsafeOrUnpublishedPaths(t *testing.T) {
	for _, rawURL := range []string{"javascript:alert(1)", "/private/image.jpg", "data:image/png;base64,AAAA", "//images.example.com/poster.jpg"} {
		if actual := resolveNotificationImageURL(rawURL, "https://seshat.example.com"); actual != "" {
			t.Errorf("unsafe image URL %q resolved to %q", rawURL, actual)
		}
	}
}

func TestChannelsWithoutRichContentCapabilityRemainPlainText(t *testing.T) {
	adapter, ok := defaultNotificationChannelRegistry.Get("webhook")
	if !ok {
		t.Fatal("webhook adapter missing")
	}
	channel := &model.NotificationChannel{Type: "webhook"}
	message, err := prepareNotificationMessage(adapter, channel, outboundMessage{Title: "标题", Body: "正文"})
	if err != nil {
		t.Fatal(err)
	}
	if message.Rendered == nil || message.Rendered.Format != notificationFormatPlainText || message.Body != "正文" {
		t.Fatalf("default channel rendering changed: %+v", message.Rendered)
	}
}

func TestDeliveryContentSnapshotIsStableAcrossChannelEdits(t *testing.T) {
	db := setupNotificationTestDB(t)
	adapter, _ := defaultNotificationChannelRegistry.Get("dingtalk")
	channel := &model.NotificationChannel{Type: "dingtalk", Config: []byte(`{"message_format":"markdown","include_image":true}`)}
	delivery := model.NotificationDelivery{ChannelName: "DingTalk", ChannelType: "dingtalk", Status: "sending"}
	if err := db.Create(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	first, err := messageForDelivery(adapter, channel, &delivery, outboundMessage{Title: "原始标题", Body: "原始正文"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Rendered == nil || first.Rendered.Format != notificationFormatMarkdown {
		t.Fatalf("first rendering did not select Markdown: %+v", first.Rendered)
	}
	channel.Config = []byte(`{"message_format":"plain_text","include_image":false}`)
	second, err := messageForDelivery(adapter, channel, &delivery, outboundMessage{Title: "已修改标题", Body: "已修改正文"})
	if err != nil {
		t.Fatal(err)
	}
	if second.Rendered == nil || second.Rendered.Format != notificationFormatMarkdown || second.Rendered.Title != "原始标题" || second.Rendered.Body != first.Rendered.Body {
		t.Fatalf("delivery snapshot changed during retry: first=%+v second=%+v", first.Rendered, second.Rendered)
	}
	var stored model.NotificationDelivery
	if err := db.First(&stored, delivery.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ContentVersion != notificationRendererVersion || stored.ContentFormat != string(notificationFormatMarkdown) || stored.ContentBody == "" {
		t.Fatalf("delivery snapshot was not persisted: %+v", stored)
	}
}
