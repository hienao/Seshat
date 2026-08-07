package handler

import (
	"errors"

	"basegoapp/internal/service"
	"basegoapp/pkg/response"

	"github.com/gin-gonic/gin"
)

// SettingHandler 设置处理器
type SettingHandler struct {
	settingService *service.SettingService
}

// NewSettingHandler 创建设置处理器
func NewSettingHandler(retentionUpdaters ...service.APILogRetentionUpdater) *SettingHandler {
	return &SettingHandler{
		settingService: service.NewSettingService(retentionUpdaters...),
	}
}

// GetRegistrationStatus 获取注册状态
// @Summary 获取注册状态
// @Description 检查系统是否允许注册（公开接口）
// @Tags 设置
// @Produce json
// @Success 200 {object} response.Response{data=map[string]bool}
// @Router /api/settings/registration-status [get]
func (h *SettingHandler) GetRegistrationStatus(c *gin.Context) {
	allowed := h.settingService.IsRegistrationAllowed()
	response.Success(c, gin.H{"allowed": allowed})
}

// GetSystemSettings 获取系统设置
// @Summary 获取系统设置
// @Description 获取所有系统设置（需要管理员权限）
// @Tags 设置
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response{data=service.SystemSettingsResponse}
// @Failure 403 {object} response.Response
// @Router /api/settings/system [get]
func (h *SettingHandler) GetSystemSettings(c *gin.Context) {
	settings, err := h.settingService.GetSystemSettings()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, settings)
}

// UpdateSystemSettings 更新系统设置
// @Summary 更新系统设置
// @Description 更新系统设置（需要管理员权限）
// @Tags 设置
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.UpdateSystemSettingsRequest true "系统设置"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/settings/system [put]
func (h *SettingHandler) UpdateSystemSettings(c *gin.Context) {
	var req service.UpdateSystemSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.UpdateSystemSettings(&req); err != nil {
		if errors.Is(err, service.ErrInvalidAPILogRetentionDays) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetSettingService 获取设置服务
func (h *SettingHandler) GetSettingService() *service.SettingService {
	return h.settingService
}
