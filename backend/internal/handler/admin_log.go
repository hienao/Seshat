package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"seshat/internal/logging"
	"seshat/internal/service"
	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

type AdminLogHandler struct{ service *service.AdminLogService }

func NewAdminLogHandler(manager *logging.Manager) *AdminLogHandler {
	return &AdminLogHandler{service: service.NewAdminLogService(manager)}
}

func (h *AdminLogHandler) List(c *gin.Context) {
	filter, err := queryLogFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.List(filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AdminLogHandler) Summary(c *gin.Context) {
	filter, err := queryLogFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Summary(filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AdminLogHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "日志 ID 无效")
		return
	}
	result, err := h.service.Get(id)
	if err != nil {
		response.NotFound(c, "日志不存在")
		return
	}
	response.Success(c, result)
}

func (h *AdminLogHandler) Export(c *gin.Context) {
	filter, err := queryLogFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	format := strings.ToLower(c.DefaultQuery("format", "csv"))
	if format != "csv" && format != "jsonl" {
		response.BadRequest(c, "导出格式只支持 csv 或 jsonl")
		return
	}
	contentType := "text/csv; charset=utf-8"
	extension := "csv"
	if format == "jsonl" {
		contentType = "application/x-ndjson; charset=utf-8"
		extension = "jsonl"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=seshat-api-logs-%s.%s", time.Now().Format("20060102150405"), extension))
	if _, err := h.service.Export(c.Writer, filter, format); err != nil {
		return
	}
}

type clearLogsRequest struct {
	StartAt       string `json:"start_at"`
	EndAt         string `json:"end_at"`
	Method        string `json:"method"`
	Route         string `json:"route"`
	StatusGroup   string `json:"status_group"`
	StatusCode    int    `json:"status_code"`
	UserID        uint   `json:"user_id"`
	AppCode       string `json:"app_code"`
	IntegrationID uint   `json:"integration_id"`
	RequestID     string `json:"request_id"`
	Keyword       string `json:"keyword"`
	Confirmation  string `json:"confirmation"`
}

func (h *AdminLogHandler) Clear(c *gin.Context) {
	var request clearLogsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if request.Confirmation != "CLEAR_LOGS" {
		response.BadRequest(c, "请输入 CLEAR_LOGS 确认清空日志")
		return
	}
	filter, err := logFilterFromValues(request.StartAt, request.EndAt, request.Method, request.Route, request.StatusGroup, request.StatusCode, request.UserID, request.AppCode, request.IntegrationID, request.RequestID, request.Keyword, 0, 0)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	count, err := h.service.Clear(c.GetUint("user_id"), filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted_count": count})
}

func queryLogFilter(c *gin.Context) (service.ApiLogFilter, error) {
	limit, err := queryInt(c.Query("limit"))
	if err != nil {
		return service.ApiLogFilter{}, errors.New("limit 无效")
	}
	cursor, err := queryUint64(c.Query("cursor"))
	if err != nil {
		return service.ApiLogFilter{}, errors.New("cursor 无效")
	}
	statusCode, err := queryInt(c.Query("status_code"))
	if err != nil {
		return service.ApiLogFilter{}, errors.New("status_code 无效")
	}
	userID, err := queryUint64(c.Query("user_id"))
	if err != nil {
		return service.ApiLogFilter{}, errors.New("user_id 无效")
	}
	integrationID, err := queryUint64(c.Query("integration_id"))
	if err != nil {
		return service.ApiLogFilter{}, errors.New("integration_id 无效")
	}
	return logFilterFromValues(c.Query("start_at"), c.Query("end_at"), c.Query("method"), c.Query("route"), c.Query("status_group"), statusCode, uint(userID), c.Query("app_code"), uint(integrationID), c.Query("request_id"), c.Query("keyword"), cursor, limit)
}

func logFilterFromValues(startAt, endAt, method, route, statusGroup string, statusCode int, userID uint, appCode string, integrationID uint, requestID, keyword string, cursor uint64, limit int) (service.ApiLogFilter, error) {
	start, err := parseOptionalTime(startAt)
	if err != nil {
		return service.ApiLogFilter{}, errors.New("start_at 时间格式无效")
	}
	end, err := parseOptionalTime(endAt)
	if err != nil {
		return service.ApiLogFilter{}, errors.New("end_at 时间格式无效")
	}
	if start != nil && end != nil && !start.Before(*end) {
		return service.ApiLogFilter{}, errors.New("start_at 必须早于 end_at")
	}
	statusGroup = strings.ToLower(strings.TrimSpace(statusGroup))
	if statusGroup != "" && statusGroup != "2xx" && statusGroup != "3xx" && statusGroup != "4xx" && statusGroup != "5xx" {
		return service.ApiLogFilter{}, errors.New("status_group 无效")
	}
	return service.ApiLogFilter{StartAt: start, EndAt: end, Method: method, Route: route, StatusGroup: statusGroup, StatusCode: statusCode, UserID: userID, AppCode: appCode, IntegrationID: integrationID, RequestID: requestID, Keyword: keyword, Cursor: cursor, Limit: limit}, nil
}

func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return &parsed, nil
	}
	return nil, errors.New("时间格式无效")
}

func queryInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
func queryUint64(value string) (uint64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseUint(value, 10, 64)
}
