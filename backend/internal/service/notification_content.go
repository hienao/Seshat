package service

import (
	"errors"
	"net/url"
	"strings"

	"seshat/internal/model"
)

type notificationFormat string

const (
	notificationFormatPlainText notificationFormat = "plain_text"
	notificationFormatMarkdown  notificationFormat = "markdown"
	notificationRendererVersion                    = 1
)

type notificationFact struct {
	Label string
	Value string
}

type notificationAction struct {
	Label string
	URL   string
}

type notificationDocument struct {
	Version  int
	Title    string
	Summary  string
	Severity string
	ImageURL string
	ImageAlt string
	Overview string
	Facts    []notificationFact
	Actions  []notificationAction
}

type renderedNotification struct {
	Format  notificationFormat
	Profile string
	Version int
	Title   string
	Body    string
}

type notificationContentPolicy struct {
	Formats         []notificationFormat
	DefaultFormat   notificationFormat
	PreferredFormat notificationFormat
	Profile         string
	IncludeImage    bool
}

// notificationContentPolicyProvider is optional. Adapters that do not expose
// richer formats continue to receive the shared plain-text rendering.
type notificationContentPolicyProvider interface {
	ContentPolicy(map[string]interface{}) notificationContentPolicy
}

type notificationContentRenderer interface {
	Format() notificationFormat
	Render(notificationDocument, notificationContentPolicy) renderedNotification
}

var notificationContentRenderers = map[notificationFormat]notificationContentRenderer{
	notificationFormatPlainText: plainTextNotificationRenderer{},
	notificationFormatMarkdown:  markdownNotificationRenderer{},
}

func prepareNotificationMessage(adapter notificationChannelAdapter, channel *model.NotificationChannel, message outboundMessage) (outboundMessage, error) {
	if message.Rendered != nil {
		message.Title = message.Rendered.Title
		message.Body = message.Rendered.Body
		return message, nil
	}
	config, _ := notificationChannelConfiguration(channel)
	policy := defaultNotificationContentPolicy()
	if provider, ok := adapter.(notificationContentPolicyProvider); ok {
		policy = normalizeNotificationContentPolicy(provider.ContentPolicy(config))
	}
	format := negotiateNotificationFormat(policy)
	renderer, ok := notificationContentRenderers[format]
	if !ok {
		return message, &deliveryError{message: "推送渠道请求了不支持的消息格式"}
	}
	document := buildNotificationDocument(message, policy.IncludeImage)
	rendered := renderer.Render(document, policy)
	message.Rendered = &rendered
	message.Title = rendered.Title
	message.Body = rendered.Body
	return message, nil
}

func defaultNotificationContentPolicy() notificationContentPolicy {
	return notificationContentPolicy{
		Formats:       []notificationFormat{notificationFormatPlainText},
		DefaultFormat: notificationFormatPlainText,
	}
}

func normalizeNotificationContentPolicy(policy notificationContentPolicy) notificationContentPolicy {
	if len(policy.Formats) == 0 {
		policy.Formats = []notificationFormat{notificationFormatPlainText}
	}
	if !notificationFormatSupported(policy.Formats, policy.DefaultFormat) {
		policy.DefaultFormat = policy.Formats[0]
	}
	return policy
}

func negotiateNotificationFormat(policy notificationContentPolicy) notificationFormat {
	if notificationFormatSupported(policy.Formats, policy.PreferredFormat) {
		return policy.PreferredFormat
	}
	if notificationFormatSupported(policy.Formats, policy.DefaultFormat) {
		return policy.DefaultFormat
	}
	return notificationFormatPlainText
}

func notificationFormatSupported(formats []notificationFormat, target notificationFormat) bool {
	if target == "" {
		return false
	}
	for _, format := range formats {
		if format == target {
			return true
		}
	}
	return false
}

func buildNotificationDocument(message outboundMessage, includeImage bool) notificationDocument {
	document := notificationDocument{
		Version:  notificationRendererVersion,
		Title:    strings.TrimSpace(message.Title),
		Summary:  strings.TrimSpace(message.Body),
		Severity: strings.TrimSpace(message.Severity),
	}
	if presentation := message.Presentation; presentation != nil {
		if title := strings.TrimSpace(presentation.Title); title != "" {
			document.Title = title
		}
		if summary := strings.TrimSpace(presentation.Summary); summary != "" {
			document.Summary = summary
		}
		for _, fact := range presentation.Facts {
			label := strings.TrimSpace(fact["label"])
			if label == "" {
				label = strings.TrimSpace(fact["label_key"])
			}
			value := strings.TrimSpace(fact["value"])
			if label != "" && value != "" {
				document.Facts = append(document.Facts, notificationFact{Label: label, Value: value})
			}
		}
		for _, link := range presentation.Links {
			appendNotificationAction(&document, link["label"], link["url"])
		}
		if media, ok := presentation.Data["media"].(map[string]interface{}); ok {
			document.Overview = notificationMapString(media, "overview")
			document.ImageAlt = notificationMapString(media, "display_name")
			if document.ImageAlt == "" {
				document.ImageAlt = document.Title
			}
			if includeImage {
				document.ImageURL = resolveNotificationImageURL(notificationMapString(media, "image_url"), message.PublicBaseURL)
			}
		}
	}
	if document.Overview == document.Summary {
		document.Overview = ""
	}
	appendNotificationAction(&document, "消息详情", message.DetailURL)
	return document
}

func notificationMapString(values map[string]interface{}, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func appendNotificationAction(document *notificationDocument, label, rawURL string) {
	label = strings.TrimSpace(label)
	rawURL = strings.TrimSpace(rawURL)
	if label == "" || !validNotificationActionURL(rawURL) {
		return
	}
	for _, action := range document.Actions {
		if action.URL == rawURL {
			return
		}
	}
	document.Actions = append(document.Actions, notificationAction{Label: label, URL: rawURL})
}

func validNotificationActionURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && parsed.Hostname() != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func resolveNotificationImageURL(rawURL, publicBaseURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		if parsed.Hostname() != "" && (parsed.Scheme == "http" || parsed.Scheme == "https") {
			return parsed.String()
		}
		return ""
	}
	if !strings.HasPrefix(parsed.Path, "/api/public/media-images/") {
		return ""
	}
	base, err := url.Parse(strings.TrimSpace(publicBaseURL))
	if err != nil || base.Hostname() == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return ""
	}
	base.Path, base.RawPath, base.RawQuery, base.Fragment = "/", "", "", ""
	return base.ResolveReference(parsed).String()
}

type plainTextNotificationRenderer struct{}

func (plainTextNotificationRenderer) Format() notificationFormat { return notificationFormatPlainText }

func (plainTextNotificationRenderer) Render(document notificationDocument, policy notificationContentPolicy) renderedNotification {
	sections := make([]string, 0, 4)
	if document.Summary != "" {
		sections = append(sections, document.Summary)
	}
	if document.Overview != "" {
		sections = append(sections, document.Overview)
	}
	if len(document.Facts) > 0 {
		facts := make([]string, 0, len(document.Facts))
		for _, fact := range document.Facts {
			facts = append(facts, fact.Label+": "+fact.Value)
		}
		sections = append(sections, strings.Join(facts, "\n"))
	}
	if len(document.Actions) > 0 {
		actions := make([]string, 0, len(document.Actions))
		for _, action := range document.Actions {
			actions = append(actions, action.Label+"："+action.URL)
		}
		sections = append(sections, strings.Join(actions, "\n"))
	}
	return renderedNotification{Format: notificationFormatPlainText, Profile: policy.Profile, Version: notificationRendererVersion, Title: document.Title, Body: strings.Join(sections, "\n\n")}
}

type markdownNotificationRenderer struct{}

func (markdownNotificationRenderer) Format() notificationFormat { return notificationFormatMarkdown }

func (markdownNotificationRenderer) Render(document notificationDocument, policy notificationContentPolicy) renderedNotification {
	sections := []string{"### " + escapeNotificationMarkdown(document.Title)}
	if document.Summary != "" {
		sections = append(sections, "> "+strings.ReplaceAll(escapeNotificationMarkdown(document.Summary), "\n", "\n> "))
	}
	if document.ImageURL != "" {
		sections = append(sections, "!["+escapeNotificationMarkdown(document.ImageAlt)+"]("+document.ImageURL+")")
	}
	if document.Overview != "" {
		sections = append(sections, escapeNotificationMarkdown(document.Overview))
	}
	if len(document.Facts) > 0 {
		facts := make([]string, 0, len(document.Facts))
		for _, fact := range document.Facts {
			facts = append(facts, "- **"+escapeNotificationMarkdown(fact.Label)+"：** "+escapeNotificationMarkdown(fact.Value))
		}
		sections = append(sections, strings.Join(facts, "\n"))
	}
	if len(document.Actions) > 0 {
		actions := make([]string, 0, len(document.Actions))
		for _, action := range document.Actions {
			actions = append(actions, "["+escapeNotificationMarkdown(action.Label)+"]("+action.URL+")")
		}
		sections = append(sections, strings.Join(actions, " · "))
	}
	return renderedNotification{Format: notificationFormatMarkdown, Profile: policy.Profile, Version: notificationRendererVersion, Title: document.Title, Body: strings.Join(sections, "\n\n")}
}

func escapeNotificationMarkdown(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "{", "\\{", "}", "\\}",
		"[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "#", "\\#", "+", "\\+",
		"-", "\\-", ".", "\\.", "!", "\\!", "|", "\\|", ">", "\\>",
	)
	return replacer.Replace(value)
}

var errNotificationSnapshotUnavailable = errors.New("推送内容快照不可用")
