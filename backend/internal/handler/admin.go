package handler

import (
	"seshat/internal/service"
	"seshat/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	authService *service.AuthService
}

// NewAdminHandler 创建管理员处理器
func NewAdminHandler(authService *service.AuthService) *AdminHandler {
	return &AdminHandler{
		authService: authService,
	}
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取所有用户列表（需要管理员权限）
// @Tags 管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response{data=[]service.UserResponse}
// @Failure 403 {object} response.Response
// @Router /api/admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.authService.ListUsers()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, users)
}

// SetUserRoleRequest 设置用户角色请求
type SetUserRoleRequest struct {
	IsAdmin bool `json:"is_admin"`
}

// SetUserRole 设置用户角色
// @Summary 设置用户角色
// @Description 将用户提升或降级管理员权限（需要管理员权限）
// @Tags 管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body SetUserRoleRequest true "角色信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/admin/users/{id}/role [put]
func (h *AdminHandler) SetUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req SetUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 防止管理员降级自己
	currentUserID := c.GetUint("user_id")
	if uint(userID) == currentUserID && !req.IsAdmin {
		response.BadRequest(c, "不能降级自己的管理员权限")
		return
	}

	if err := h.authService.SetUserRole(uint(userID), req.IsAdmin); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, nil)
}
