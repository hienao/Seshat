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

func (h *WebhookHandler) GetIntegrationSecret(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "接入 ID 无效")
		return
	}
	item, err := h.service.GetIntegrationSecret(c.GetUint("user_id"), uint(id))
	if err != nil {
		response.NotFound(c, "接入不存在")
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, item)
}

// GetIntegrationMediaSettings 获取 Jellyfin/Emby 实例的媒体 API 配置。
// @Summary 获取实例媒体 API 配置
// @Tags Webhook
// @Security BearerAuth
// @Produce json
// @Param id path int true "接入实例 ID"
// @Success 200 {object} response.Response{data=service.IntegrationMediaSettingsResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/webhooks/integrations/{id}/media-settings [get]
func (h *WebhookHandler) GetIntegrationMediaSettings(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "接入 ID 无效")
		return
	}
	item, err := h.service.GetIntegrationMediaSettings(c.GetUint("user_id"), uint(id))
	if err != nil {
		if errors.Is(err, service.ErrMediaAPIUnsupported) {
			response.BadRequest(c, err.Error())
			return
		}
		response.NotFound(c, "接入不存在")
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, item)
}

// UpdateIntegrationMediaSettings 保存 Jellyfin/Emby 实例的媒体 API 配置。
// @Summary 保存实例媒体 API 配置
// @Tags Webhook
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "接入实例 ID"
// @Param request body service.UpdateIntegrationMediaSettingsRequest true "媒体 API 配置"
// @Success 200 {object} response.Response{data=service.IntegrationMediaSettingsResponse}
// @Failure 400 {object} response.Response
// @Router /api/webhooks/integrations/{id}/media-settings [put]
func (h *WebhookHandler) UpdateIntegrationMediaSettings(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "接入 ID 无效")
		return
	}
	var req service.UpdateIntegrationMediaSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "服务器地址和 API Key 均不能为空")
		return
	}
	item, err := h.service.UpdateIntegrationMediaSettings(c.GetUint("user_id"), uint(id), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, item)
}

// TestIntegrationMediaSettings 使用当前输入测试 Jellyfin/Emby 媒体 API，不保存配置。
// @Summary 测试实例媒体 API
// @Tags Webhook
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "接入实例 ID"
// @Param request body service.UpdateIntegrationMediaSettingsRequest true "媒体 API 配置"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Router /api/webhooks/integrations/{id}/media-settings/test [post]
func (h *WebhookHandler) TestIntegrationMediaSettings(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "接入 ID 无效")
		return
	}
	var req service.UpdateIntegrationMediaSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "服务器地址和 API Key 均不能为空")
		return
	}
	if err := h.service.TestIntegrationMediaSettings(c.Request.Context(), c.GetUint("user_id"), uint(id), &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "连接成功"})
}

func (h *WebhookHandler) ListEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	var integrationID uint64
	var err error
	if value := c.Query("integration_id"); value != "" {
		integrationID, err = strconv.ParseUint(value, 10, 64)
		if err != nil || integrationID == 0 {
			response.BadRequest(c, "App 接入 ID 无效")
			return
		}
	}
	items, err := h.service.ListEvents(c.GetUint("user_id"), service.EventListFilter{AppCode: c.Query("app_code"), IntegrationID: uint(integrationID), EventType: c.Query("event_type"), Limit: limit, Offset: offset})
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

// GetPublicEvent 获取公开消息的标准化展示
// @Summary 获取公开消息详情
// @Description 通过不可猜测的访问标识获取标准化展示，不返回原始消息或内部信息
// @Tags Webhook
// @Produce json
// @Param token path string true "公开访问标识"
// @Success 200 {object} response.Response{data=service.PublicEventResponse}
// @Failure 404 {object} response.Response
// @Router /api/public/events/{token} [get]
func (h *WebhookHandler) GetPublicEvent(c *gin.Context) {
	item, err := h.service.GetPublicEvent(c.Param("token"))
	if err != nil {
		if errors.Is(err, service.ErrPublicEventNotFound) {
			response.NotFound(c, "消息不存在")
			return
		}
		response.InternalError(c, "读取消息失败")
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	response.Success(c, item)
}

// GetPublicMediaImage 通过随机标识读取缓存后的媒体服务器图片。
// @Summary 获取缓存媒体图片
// @Tags Webhook
// @Produce image/jpeg
// @Param token path string true "图片访问标识"
// @Success 200 {file} binary
// @Failure 404 {object} response.Response
// @Router /api/public/media-images/{token} [get]
func (h *WebhookHandler) GetPublicMediaImage(c *gin.Context) {
	item, err := h.service.GetCachedMediaImage(c.Param("token"))
	if err != nil {
		response.NotFound(c, "媒体图片不存在或已过期")
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, item.ImageType, item.ImageData)
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
