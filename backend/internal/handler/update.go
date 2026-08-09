package handler

import (
	"seshat/internal/logging"
	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

type UpdateHandler struct {
	updateService *service.UpdateService
}

func NewUpdateHandler(updateServices ...*service.UpdateService) *UpdateHandler {
	updateService := service.NewUpdateService()
	if len(updateServices) > 0 && updateServices[0] != nil {
		updateService = updateServices[0]
	}
	return &UpdateHandler{updateService: updateService}
}

// GetVersion 返回当前运行镜像内嵌的构建版本。
// @Summary 获取当前版本
// @Tags 更新
// @Produce json
// @Success 200 {object} response.Response{data=buildinfo.Info}
// @Router /api/version [get]
func (h *UpdateHandler) GetVersion(c *gin.Context) {
	response.Success(c, h.updateService.CurrentVersion())
}

// CheckUpdates 检查当前发布渠道中的更新。
// @Summary 检查更新
// @Description Beta 只检查 Beta，Release/Hotfix 只检查正式版本（需要管理员权限）
// @Tags 更新
// @Security BearerAuth
// @Produce json
// @Param refresh query bool false "忽略缓存并重新检查"
// @Success 200 {object} response.Response{data=service.UpdateStatusResponse}
// @Failure 502 {object} response.Response
// @Router /api/admin/updates [get]
func (h *UpdateHandler) CheckUpdates(c *gin.Context) {
	status, err := h.updateService.Check(c.Request.Context(), c.Query("refresh") == "true")
	if err != nil {
		logging.Warn("update", "更新检查失败", logging.Fields{"error": err, "channel": h.updateService.CurrentVersion().Channel})
		response.ErrorWithCode(c, 502, "update_check_failed", "无法获取更新信息")
		return
	}
	response.Success(c, status)
}
