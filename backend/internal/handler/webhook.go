package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	appLogging "seshat/internal/logging"
	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct{ service *service.WebhookService }

const webhookLogBodyLimit = 64 << 10

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{service: service.NewWebhookService()}
}

func (h *WebhookHandler) Catalog(c *gin.Context) { response.Success(c, h.service.Catalog()) }

func (h *WebhookHandler) ListIntegrations(c *gin.Context) {
	items, err := h.service.ListIntegrations(c.GetUint("user_id"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, items)
}

func (h *WebhookHandler) CreateIntegration(c *gin.Context) {
	var req service.CreateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	item, err := h.service.CreateIntegration(c.GetUint("user_id"), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *WebhookHandler) RotateSecret(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "接入 ID 无效")
		return
	}
	item, err := h.service.RotateSecret(c.GetUint("user_id"), uint(id))
	if err != nil {
		response.NotFound(c, "接入不存在")
		return
	}
	response.Success(c, item)
}

func (h *WebhookHandler) ListEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := h.service.ListEvents(c.GetUint("user_id"), c.Query("app_code"), c.Query("event_type"), limit, offset)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, items)
}

func (h *WebhookHandler) GetEvent(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "消息 ID 无效")
		return
	}
	item, err := h.service.GetEvent(c.GetUint("user_id"), uint(id))
	if err != nil {
		response.NotFound(c, "消息不存在")
		return
	}
	response.Success(c, item)
}

func (h *WebhookHandler) Receive(c *gin.Context) {
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	if safeHeaders, err := json.Marshal(redactWebhookHeaders(headers)); err == nil {
		c.Set("request_headers", string(safeHeaders))
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20+1))
	if err != nil {
		appLogging.Warn("webhook", "读取 Webhook 消息失败", appLogging.Fields{"request_id": c.GetString("request_id"), "error": err})
		response.ErrorWithCode(c, http.StatusBadRequest, "webhook_body_read_failed", "读取 Webhook 消息失败")
		return
	}
	c.Set("request_body", webhookBodyPreview(body))
	result, err := h.service.Ingest(c.Param("endpointKey"), headers, body, c.GetHeader("Content-Type"))
	setWebhookLogContext(c, result)
	if err != nil {
		var ingestError *service.WebhookIngestError
		if errors.As(err, &ingestError) {
			fields := webhookApplicationLogFields(c, result)
			fields["error_code"] = ingestError.Code
			fields["error"] = ingestError
			if ingestError.StatusCode >= http.StatusInternalServerError {
				appLogging.Error("webhook", "Webhook 消息处理失败", fields)
			} else {
				appLogging.Warn("webhook", "Webhook 消息被拒绝", fields)
			}
			c.Set("error_code", ingestError.Code)
			c.Set("error_message", ingestError.Error())
			c.JSON(ingestError.StatusCode, gin.H{"code": -1, "message": ingestError.Message})
			return
		}
		appLogging.Error("webhook", "Webhook 消息处理失败", appLogging.Fields{"request_id": c.GetString("request_id"), "error": err})
		response.ErrorWithCode(c, http.StatusInternalServerError, "webhook_ingest_failed", "处理 Webhook 消息失败")
		return
	}
	appLogging.Info("webhook", "Webhook 消息已接收", webhookApplicationLogFields(c, result))
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted"})
}

func webhookApplicationLogFields(c *gin.Context, result *service.WebhookIngestResult) appLogging.Fields {
	fields := appLogging.Fields{"request_id": c.GetString("request_id")}
	if result == nil {
		return fields
	}
	if result.AppCode != "" {
		fields["app_code"] = result.AppCode
	}
	if result.IntegrationID > 0 {
		fields["integration_id"] = result.IntegrationID
	}
	if result.EventID > 0 {
		fields["event_id"] = result.EventID
	}
	return fields
}

func redactWebhookHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "authorization") || strings.Contains(lowerKey, "cookie") || strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "signature") || strings.Contains(lowerKey, "api-key") || strings.Contains(lowerKey, "apikey") {
			result[key] = "[REDACTED]"
			continue
		}
		result[key] = value
	}
	return result
}

func webhookBodyPreview(body []byte) string {
	originalSize := len(body)
	if originalSize > webhookLogBodyLimit {
		body = body[:webhookLogBodyLimit]
	}
	var preview string
	if utf8.Valid(body) {
		preview = string(body)
	} else {
		preview = "base64:" + base64.StdEncoding.EncodeToString(body)
	}
	if originalSize > webhookLogBodyLimit {
		preview += "\n...[truncated, original bytes: " + strconv.Itoa(originalSize) + "]"
	}
	return preview
}

func setWebhookLogContext(c *gin.Context, result *service.WebhookIngestResult) {
	if result == nil {
		return
	}
	if result.AppCode != "" {
		c.Set("app_code", result.AppCode)
	}
	if result.IntegrationID > 0 {
		c.Set("integration_id", result.IntegrationID)
	}
	if result.EventID > 0 {
		c.Set("event_id", result.EventID)
	}
}
