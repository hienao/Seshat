package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const DefaultEventType = "__default__"

type EventTypeDefinition struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	RenderMode string `json:"render_mode"`
}

type AppDefinition struct {
	Code             string                `json:"code"`
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	AuthMode         string                `json:"auth_mode"`
	DefaultEventType string                `json:"default_event_type"`
	EventTypes       []EventTypeDefinition `json:"event_types"`
}

type IncomingRequest struct {
	Headers map[string]string
	Body    []byte
}

type Presentation struct {
	SchemaVersion int                    `json:"schema_version"`
	Title         string                 `json:"title"`
	Summary       string                 `json:"summary,omitempty"`
	Severity      string                 `json:"severity"`
	Tags          []string               `json:"tags,omitempty"`
	Facts         []map[string]string    `json:"facts,omitempty"`
	Links         []map[string]string    `json:"links,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
}

type Provider interface {
	Code() string
	Definition() AppDefinition
	Verify(secret string, request IncomingRequest) bool
	DetectType(request IncomingRequest) string
	MapType(sourceEventType string) string
	ExternalEventID(request IncomingRequest) string
	Normalize(eventType string, request IncomingRequest) Presentation
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	r := &Registry{providers: make(map[string]Provider)}
	r.Register(NewGenericProvider("generic", "通用 Webhook", "支持 X-Webhook-Event 和 JSON event_type/type 约定的消息"))
	r.Register(NewGitHubProvider())
	r.Register(NewJellyfinProvider())
	r.Register(NewEmbyProvider())
	return r
}

func (r *Registry) Register(provider Provider) { r.providers[provider.Code()] = provider }

func (r *Registry) Get(code string) (Provider, bool) {
	provider, ok := r.providers[code]
	return provider, ok
}

func (r *Registry) Definitions() []AppDefinition {
	result := make([]AppDefinition, 0, len(r.providers))
	for _, provider := range r.providers {
		result = append(result, provider.Definition())
	}
	return result
}

func (r *Registry) CreateDefinition(providerCode, providerName, description string) AppDefinition {
	return AppDefinition{
		Code: providerCode, Name: providerName, Description: description,
		AuthMode:         "secret_header",
		DefaultEventType: DefaultEventType,
		EventTypes:       []EventTypeDefinition{{Code: DefaultEventType, Name: "其他消息", RenderMode: "raw"}},
	}
}

type genericProvider struct{ definition AppDefinition }

func NewGenericProvider(code, name, description string) Provider {
	return &genericProvider{definition: AppDefinition{
		Code: code, Name: name, Description: description, DefaultEventType: DefaultEventType,
		AuthMode:   "secret_header",
		EventTypes: []EventTypeDefinition{{Code: DefaultEventType, Name: "其他消息", RenderMode: "raw"}},
	}}
}

func (p *genericProvider) Code() string              { return p.definition.Code }
func (p *genericProvider) Definition() AppDefinition { return p.definition }
func (p *genericProvider) Verify(secret string, request IncomingRequest) bool {
	return verifyRequest(secret, request)
}
func (p *genericProvider) DetectType(request IncomingRequest) string {
	if value := header(request, "X-Webhook-Event"); value != "" {
		return value
	}
	var payload map[string]interface{}
	if json.Unmarshal(request.Body, &payload) == nil {
		for _, key := range []string{"event_type", "event", "type"} {
			if value, ok := payload[key].(string); ok && value != "" {
				return value
			}
		}
	}
	return "unknown"
}
func (p *genericProvider) MapType(sourceEventType string) string { return sourceEventType }
func (p *genericProvider) ExternalEventID(request IncomingRequest) string {
	for _, key := range []string{"X-Webhook-ID", "X-Event-ID", "Idempotency-Key"} {
		if value := header(request, key); value != "" {
			return value
		}
	}
	return ""
}
func (p *genericProvider) Normalize(eventType string, request IncomingRequest) Presentation {
	title := fmt.Sprintf("%s Webhook", p.definition.Name)
	sourceEventType := p.DetectType(request)
	if sourceEventType != "" && sourceEventType != "unknown" {
		title += " · " + sourceEventType
	}
	return Presentation{SchemaVersion: 1, Title: title, Summary: "收到一条 Webhook 消息", Severity: "info", Data: map[string]interface{}{"category": "raw", "event_type": sourceEventType, "raw_preview": bodyPreview(request.Body, 600)}}
}

type githubProvider struct{ definition AppDefinition }

func NewGitHubProvider() Provider {
	return &githubProvider{definition: AppDefinition{
		Code: "github", Name: "GitHub", Description: "GitHub Webhooks", DefaultEventType: DefaultEventType,
		AuthMode: "github_signature",
		EventTypes: []EventTypeDefinition{
			{Code: DefaultEventType, Name: "其他消息", RenderMode: "raw"},
			{Code: "push", Name: "代码推送", RenderMode: "custom"},
			{Code: "pull_request", Name: "Pull Request", RenderMode: "custom"},
			{Code: "issues", Name: "Issue", RenderMode: "custom"},
		},
	}}
}
func (p *githubProvider) Code() string              { return p.definition.Code }
func (p *githubProvider) Definition() AppDefinition { return p.definition }
func (p *githubProvider) Verify(secret string, request IncomingRequest) bool {
	signature := header(request, "X-Hub-Signature-256")
	if signature == "" {
		return verifyRequest(secret, request)
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	actual := hmac.New(sha256.New, []byte(secret))
	_, _ = actual.Write(request.Body)
	expected, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	return err == nil && hmac.Equal(actual.Sum(nil), expected)
}
func (p *githubProvider) DetectType(request IncomingRequest) string {
	if value := header(request, "X-GitHub-Event"); value != "" {
		return value
	}
	return "unknown"
}
func (p *githubProvider) MapType(sourceEventType string) string { return sourceEventType }
func (p *githubProvider) ExternalEventID(request IncomingRequest) string {
	return header(request, "X-GitHub-Delivery")
}
func (p *githubProvider) Normalize(eventType string, request IncomingRequest) Presentation {
	var payload map[string]interface{}
	_ = json.Unmarshal(request.Body, &payload)
	repository := ""
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		repository, _ = repo["full_name"].(string)
	}
	sourceEventType := p.DetectType(request)
	title := "GitHub · " + sourceEventType
	if repository != "" {
		title += " · " + repository
	}
	data := map[string]interface{}{"repository": repository}
	if eventType == DefaultEventType {
		data["category"] = "raw"
		data["raw_preview"] = bodyPreview(request.Body, 600)
	}
	return Presentation{SchemaVersion: 1, Title: title, Summary: "收到 GitHub Webhook 消息", Severity: "info", Data: data}
}

func firstString(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func verifyRequest(secret string, request IncomingRequest) bool {
	if secret == "" {
		return false
	}
	if candidate := header(request, "X-Webhook-Secret"); candidate != "" && hmac.Equal([]byte(candidate), []byte(secret)) {
		return true
	}
	signature := header(request, "X-Webhook-Signature")
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	actual := hmac.New(sha256.New, []byte(secret))
	_, _ = actual.Write(request.Body)
	expected, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	return err == nil && hmac.Equal(actual.Sum(nil), expected)
}

func header(request IncomingRequest, name string) string {
	for key, value := range request.Headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func bodyPreview(body []byte, limit int) string {
	value := []rune(strings.TrimSpace(string(body)))
	if limit > 0 && len(value) > limit {
		return string(value[:limit]) + "…"
	}
	return string(value)
}
