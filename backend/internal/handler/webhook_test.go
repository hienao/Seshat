package handler

import (
	"strings"
	"testing"
)

func TestRedactWebhookHeaders(t *testing.T) {
	headers := redactWebhookHeaders(map[string]string{
		"Content-Type":        "application/json",
		"X-GitHub-Event":      "push",
		"X-Hub-Signature-256": "sha256=secret",
		"Authorization":       "Bearer secret",
		"X-Webhook-Secret":    "secret",
	})
	if headers["Content-Type"] != "application/json" || headers["X-GitHub-Event"] != "push" {
		t.Fatalf("safe headers were changed: %+v", headers)
	}
	for _, key := range []string{"X-Hub-Signature-256", "Authorization", "X-Webhook-Secret"} {
		if headers[key] != "[REDACTED]" {
			t.Fatalf("header %s was not redacted", key)
		}
	}
}

func TestWebhookBodyPreviewIsLimited(t *testing.T) {
	body := []byte(strings.Repeat("a", webhookLogBodyLimit+100))
	preview := webhookBodyPreview(body)
	if !strings.Contains(preview, "...[truncated, original bytes:") {
		t.Fatal("large webhook body was not marked as truncated")
	}
	if len(preview) >= len(body) {
		t.Fatal("large webhook body was not reduced")
	}
}
