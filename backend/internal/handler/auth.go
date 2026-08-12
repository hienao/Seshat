package handler

import (
	"seshat/config"
	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService    *service.AuthService
	settingService *service.SettingService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService:    service.NewAuthService(cfg),
		settingService: service.NewSettingService(),
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账户
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "注册信息"
// @Success 200 {object} response.Response{data=service.UserResponse}
// @Failure 400 {object} response.Response
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	// 检查是否允许注册
	if !h.settingService.IsRegistrationAllowed() {
		response.BadRequest(c, "系统当前不允许注册")
		return
	}

	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, user)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取 JWT Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "登录信息"
// @Success 200 {object} response.Response{data=service.TokenResponse}
// @Failure 400 {object} response.Response
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	token, err := h.authService.Login(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, token)
}

// SetupAdmin 设置正式管理员凭据
// @Summary 设置正式管理员凭据
// @Description 使用一次性 admin/admin 登录后设置正式管理员用户名和密码
// @Tags 认证
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.SetupAdminRequest true "管理员凭据"
// @Success 200 {object} response.Response{data=service.TokenResponse}
// @Failure 400 {object} response.Response
// @Router /api/auth/setup-admin [post]
func (h *AuthHandler) SetupAdmin(c *gin.Context) {
	var req service.SetupAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	token, err := h.authService.SetupAdmin(c.GetUint("user_id"), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, token)
}

// Logout 用户退出
// @Summary 用户退出
// @Description 使当前用户已签发的 Bearer Token 立即失效
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.authService.Logout(c.GetUint("user_id")); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// GetAuthService 获取认证服务（用于初始化引导管理员）
func (h *AuthHandler) GetAuthService() *service.AuthService {
	return h.authService
}
