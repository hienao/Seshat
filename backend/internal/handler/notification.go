package handler

import (
	"errors"
	"net/http"
	"strconv"

	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct{ service *service.NotificationService }

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{service: service.NewNotificationService()}
}

func (h *NotificationHandler) ListChannels(c *gin.Context) {
	items, err := h.service.ListChannels(c.GetUint("user_id"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, items)
}

func (h *NotificationHandler) CreateChannel(c *gin.Context) {
	var request service.NotificationChannelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	item, err := h.service.CreateChannel(c.GetUint("user_id"), &request)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *NotificationHandler) UpdateChannel(c *gin.Context) {
	id, ok := notificationID(c)
	if !ok {
		return
	}
	var request service.NotificationChannelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	item, err := h.service.UpdateChannel(c.GetUint("user_id"), id, &request)
	if err != nil {
		if errors.Is(err, service.ErrNotificationChannelNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *NotificationHandler) DeleteChannel(c *gin.Context) {
	id, ok := notificationID(c)
	if !ok {
		return
	}
	err := h.service.DeleteChannel(c.GetUint("user_id"), id)
	if errors.Is(err, service.ErrNotificationChannelNotFound) {
		response.NotFound(c, err.Error())
		return
	}
	if errors.Is(err, service.ErrNotificationChannelInUse) {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *NotificationHandler) TestChannel(c *gin.Context) {
	id, ok := notificationID(c)
	if !ok {
		return
	}
	item, err := h.service.TestChannel(c.GetUint("user_id"), id)
	if errors.Is(err, service.ErrNotificationChannelNotFound) {
		response.NotFound(c, err.Error())
		return
	}
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *NotificationHandler) ListDeliveries(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.service.ListDeliveries(c.GetUint("user_id"), c.Query("status"), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, items)
}

func (h *NotificationHandler) RetryDelivery(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID 无效")
		return
	}
	if err := h.service.RetryDelivery(c.GetUint("user_id"), id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *NotificationHandler) GetEventStatus(c *gin.Context) {
	id, ok := notificationID(c)
	if !ok {
		return
	}
	item, err := h.service.GetEventNotificationStatus(c.GetUint("user_id"), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *NotificationHandler) GetIntegrationSettings(c *gin.Context) {
	id, ok := notificationID(c)
	if !ok {
		return
	}
	item, err := h.service.GetIntegrationSettings(c.GetUint("user_id"), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *NotificationHandler) UpdateIntegrationSettings(c *gin.Context) {
	id, ok := notificationID(c)
	if !ok {
		return
	}
	var request service.UpdateIntegrationNotificationSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	item, err := h.service.UpdateIntegrationSettings(c.GetUint("user_id"), id, &request)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func notificationID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID 无效")
		return 0, false
	}
	return uint(id), true
}
