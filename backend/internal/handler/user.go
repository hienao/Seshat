package handler

import (
	"seshat/config"
	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	authService *service.AuthService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{
		authService: service.NewAuthService(cfg),
	}
}

// GetProfile 获取当前用户信息
// @Summary 获取用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 用户
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response{data=service.UserResponse}
// @Failure 401 {object} response.Response
// @Router /api/user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, user)
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的密码
// @Tags 用户
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.ChangePasswordRequest true "密码信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/user/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req service.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetUint("user_id")

	if err := h.authService.ChangePassword(userID, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, nil)
}
