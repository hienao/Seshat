package handler

import (
	"io"
	"net/http"
	"strconv"

	"basegoapp/internal/service"
	"basegoapp/pkg/response"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct{ service *service.WebhookService }

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
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "读取 Webhook 消息失败"})
		return
	}
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	if err := h.service.Ingest(c.Param("endpointKey"), headers, body, c.GetHeader("Content-Type")); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted"})
}
