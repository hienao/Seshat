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

type AdminApplicationLogHandler struct {
	service *service.ApplicationLogService
}

func NewAdminApplicationLogHandler(manager *logging.Manager) *AdminApplicationLogHandler {
	return &AdminApplicationLogHandler{service: service.NewApplicationLogService(manager)}
}

func (h *AdminApplicationLogHandler) List(c *gin.Context) {
	filter, err := queryApplicationLogFilter(c)
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

func (h *AdminApplicationLogHandler) Summary(c *gin.Context) {
	filter, err := queryApplicationLogFilter(c)
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

func (h *AdminApplicationLogHandler) Get(c *gin.Context) {
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

func (h *AdminApplicationLogHandler) Export(c *gin.Context) {
	filter, err := queryApplicationLogFilter(c)
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
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=seshat-application-logs-%s.%s", time.Now().Format("20060102150405"), extension))
	_, _ = h.service.Export(c.Writer, filter, format)
}

type clearApplicationLogsRequest struct {
	StartAt      string `json:"start_at"`
	EndAt        string `json:"end_at"`
	Level        string `json:"level"`
	Source       string `json:"source"`
	RequestID    string `json:"request_id"`
	Keyword      string `json:"keyword"`
	Confirmation string `json:"confirmation"`
}

func (h *AdminApplicationLogHandler) Clear(c *gin.Context) {
	var request clearApplicationLogsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if request.Confirmation != "CLEAR_APPLICATION_LOGS" {
		response.BadRequest(c, "请输入 CLEAR_APPLICATION_LOGS 确认清空日志")
		return
	}
	filter, err := applicationLogFilterFromValues(request.StartAt, request.EndAt, request.Level, request.Source, request.RequestID, request.Keyword, 0, 0)
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

func queryApplicationLogFilter(c *gin.Context) (service.ApplicationLogFilter, error) {
	limit, err := queryInt(c.Query("limit"))
	if err != nil {
		return service.ApplicationLogFilter{}, errors.New("limit 无效")
	}
	cursor, err := queryUint64(c.Query("cursor"))
	if err != nil {
		return service.ApplicationLogFilter{}, errors.New("cursor 无效")
	}
	return applicationLogFilterFromValues(c.Query("start_at"), c.Query("end_at"), c.Query("level"), c.Query("source"), c.Query("request_id"), c.Query("keyword"), cursor, limit)
}

func applicationLogFilterFromValues(startAt, endAt, level, source, requestID, keyword string, cursor uint64, limit int) (service.ApplicationLogFilter, error) {
	start, err := parseOptionalTime(startAt)
	if err != nil {
		return service.ApplicationLogFilter{}, errors.New("start_at 时间格式无效")
	}
	end, err := parseOptionalTime(endAt)
	if err != nil {
		return service.ApplicationLogFilter{}, errors.New("end_at 时间格式无效")
	}
	if start != nil && end != nil && !start.Before(*end) {
		return service.ApplicationLogFilter{}, errors.New("start_at 必须早于 end_at")
	}
	level = strings.ToUpper(strings.TrimSpace(level))
	if level != "" && level != "DEBUG" && level != "INFO" && level != "WARN" && level != "ERROR" {
		return service.ApplicationLogFilter{}, errors.New("level 无效")
	}
	return service.ApplicationLogFilter{StartAt: start, EndAt: end, Level: level, Source: source, RequestID: requestID, Keyword: keyword, Cursor: cursor, Limit: limit}, nil
}
