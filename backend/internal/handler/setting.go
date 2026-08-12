package handler

import (
	"errors"

	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

// SettingHandler 设置处理器
type SettingHandler struct {
	settingService       *service.SettingService
	mediaMetadataService *service.MediaMetadataService
}

// NewSettingHandler 创建设置处理器
func NewSettingHandler(retentionUpdaters ...service.APILogRetentionUpdater) *SettingHandler {
	return &SettingHandler{
		settingService:       service.NewSettingService(retentionUpdaters...),
		mediaMetadataService: service.NewMediaMetadataService(),
	}
}

// TestTMDBConnection 测试当前输入的 TMDB API 密钥和可选 HTTP 代理，不保存配置。
// @Summary 测试 TMDB 连接
// @Description 使用当前输入的 API 密钥和代理配置请求 TMDB（需要管理员权限）
// @Tags 设置
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.TestTMDBConnectionRequest true "TMDB 测试配置"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/settings/system/test-tmdb [post]
func (h *SettingHandler) TestTMDBConnection(c *gin.Context) {
	var req service.TestTMDBConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写 TMDB API 密钥")
		return
	}
	if err := h.mediaMetadataService.TestConnection(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	message := "TMDB API 密钥连接测试成功"
	if req.UseProxy {
		message = "HTTP 代理与 TMDB API 密钥连接测试成功"
	}
	response.Success(c, gin.H{"message": message})
}

// TestHTTPProxy 测试当前输入的 HTTP 代理，不保存配置。
// @Summary 测试 HTTP 代理
// @Description 使用当前输入的代理地址访问固定 HTTPS 探测地址（需要管理员权限）
// @Tags 设置
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.TestHTTPProxyRequest true "HTTP 代理测试配置"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/settings/system/test-http-proxy [post]
func (h *SettingHandler) TestHTTPProxy(c *gin.Context) {
	var req service.TestHTTPProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写 HTTP 代理地址")
		return
	}
	if err := h.mediaMetadataService.TestHTTPProxy(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "HTTP 代理连接测试成功"})
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
		if errors.Is(err, service.ErrInvalidAPILogRetentionDays) || errors.Is(err, service.ErrInvalidHTTPProxyURL) || errors.Is(err, service.ErrConflictingHTTPProxyUpdate) || errors.Is(err, service.ErrConflictingTMDBAPIKeyUpdate) || errors.Is(err, service.ErrInvalidTMDBAPIKey) || errors.Is(err, service.ErrInvalidPublicBaseURL) {
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
