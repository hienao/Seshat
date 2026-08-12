package service

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"seshat/internal/logging"
	"seshat/internal/model"
	"seshat/pkg/database"
)

type ApiLogFilter struct {
	StartAt       *time.Time
	EndAt         *time.Time
	Method        string
	Route         string
	StatusGroup   string
	StatusCode    int
	UserID        uint
	AppCode       string
	IntegrationID uint
	RequestID     string
	Keyword       string
	Cursor        uint64
	Limit         int
}

type ApiLogListResponse struct {
	Items      []model.ApiRequestLog `json:"items"`
	Total      int64                 `json:"total"`
	NextCursor uint64                `json:"next_cursor,omitempty"`
	HasMore    bool                  `json:"has_more"`
}

type ApiLogSummary struct {
	Total        int64   `json:"total"`
	SuccessCount int64   `json:"success_count"`
	ClientErrors int64   `json:"client_errors"`
	ServerErrors int64   `json:"server_errors"`
	AverageMs    float64 `json:"average_ms"`
	Dropped      uint64  `json:"dropped"`
}

type AdminLogService struct{ manager *logging.Manager }

func NewAdminLogService(manager *logging.Manager) *AdminLogService {
	return &AdminLogService{manager: manager}
}

func (s *AdminLogService) List(filter ApiLogFilter) (*ApiLogListResponse, error) {
	if !s.enabled() {
		return &ApiLogListResponse{Items: []model.ApiRequestLog{}}, nil
	}
	filter = normalizeFilter(filter)
	base := s.query(filter)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []model.ApiRequestLog
	// 请求头和正文只在详情/导出时读取，避免日志列表一次返回大量内容。
	if err := base.Omit("request_headers", "request_body").Order("id DESC").Limit(filter.Limit + 1).Find(&items).Error; err != nil {
		return nil, err
	}
	result := &ApiLogListResponse{Total: total, HasMore: len(items) > filter.Limit}
	if result.HasMore {
		items = items[:filter.Limit]
	}
	result.Items = items
	if result.HasMore && len(items) > 0 {
		result.NextCursor = items[len(items)-1].ID
	}
	return result, nil
}

func (s *AdminLogService) Summary(filter ApiLogFilter) (*ApiLogSummary, error) {
	if !s.enabled() {
		return &ApiLogSummary{}, nil
	}
	filter = normalizeFilter(filter)
	var result ApiLogSummary
	row := s.query(filter).Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN status_code < 400 THEN 1 ELSE 0 END), 0) AS success_count, COALESCE(SUM(CASE WHEN status_code >= 400 AND status_code < 500 THEN 1 ELSE 0 END), 0) AS client_errors, COALESCE(SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END), 0) AS server_errors, COALESCE(AVG(latency_ms), 0) AS average_ms").Scan(&result)
	if row.Error != nil {
		return nil, row.Error
	}
	result.Dropped = s.manager.Dropped()
	return &result, nil
}

func (s *AdminLogService) Get(id uint64) (*model.ApiRequestLog, error) {
	if !s.enabled() {
		return nil, errors.New("接口日志未启用")
	}
	var item model.ApiRequestLog
	if err := s.manager.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *AdminLogService) Clear(actorID uint, filter ApiLogFilter) (int64, error) {
	if !s.enabled() {
		return 0, errors.New("接口日志未启用")
	}
	filter = normalizeFilter(filter)
	// 清空接口已经经过管理员权限和 CLEAR_LOGS 确认；显式条件同时满足 GORM 的全表删除保护。
	result := s.query(filter).Where("1 = 1").Delete(&model.ApiRequestLog{})
	if result.Error != nil {
		return 0, result.Error
	}
	detail, _ := json.Marshal(filter)
	audit := &model.AdminAuditLog{ActorID: actorID, Action: "clear_api_logs", Target: "api_request_logs", Detail: string(detail), OccurredAt: time.Now()}
	if err := database.GetDB().Create(audit).Error; err != nil {
		return result.RowsAffected, err
	}
	return result.RowsAffected, nil
}

func (s *AdminLogService) Export(writer io.Writer, filter ApiLogFilter, format string) (int, error) {
	if !s.enabled() {
		return 0, errors.New("接口日志未启用")
	}
	filter = normalizeFilter(filter)
	maxRows := s.manager.ExportLimit()
	if format != "csv" && format != "jsonl" {
		return 0, errors.New("导出格式只支持 csv 或 jsonl")
	}
	var csvWriter *csv.Writer
	if format == "csv" {
		csvWriter = csv.NewWriter(writer)
		if err := csvWriter.Write([]string{"id", "occurred_at", "request_id", "method", "route", "status_code", "latency_ms", "request_bytes", "response_bytes", "client_ip", "user_id", "username", "app_code", "integration_id", "event_id", "error_code", "error_message", "user_agent", "request_headers", "request_body"}); err != nil {
			return 0, err
		}
	}
	count := 0
	var cursor uint64
	for count < maxRows {
		pageFilter := filter
		pageFilter.Cursor = cursor
		pageFilter.Limit = 1000
		var rows []model.ApiRequestLog
		if err := s.query(pageFilter).Order("id DESC").Limit(pageFilter.Limit).Find(&rows).Error; err != nil {
			return count, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			if count >= maxRows {
				break
			}
			if format == "csv" {
				if err := csvWriter.Write([]string{strconv.FormatUint(row.ID, 10), row.OccurredAt.Format(time.RFC3339Nano), row.RequestID, row.Method, row.Route, strconv.Itoa(row.StatusCode), strconv.FormatInt(row.LatencyMs, 10), strconv.FormatInt(row.RequestBytes, 10), strconv.FormatInt(row.ResponseBytes, 10), row.ClientIP, strconv.FormatUint(uint64(row.UserID), 10), row.Username, row.AppCode, strconv.FormatUint(uint64(row.IntegrationID), 10), strconv.FormatUint(uint64(row.EventID), 10), row.ErrorCode, row.ErrorMessage, row.UserAgent, row.RequestHeaders, row.RequestBody}); err != nil {
					return count, err
				}
			} else if err := json.NewEncoder(writer).Encode(row); err != nil {
				return count, err
			}
			count++
		}
		cursor = rows[len(rows)-1].ID
		if len(rows) < pageFilter.Limit {
			break
		}
	}
	if csvWriter != nil {
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return count, err
		}
	}
	return count, nil
}

func (s *AdminLogService) enabled() bool { return s != nil && s.manager != nil && s.manager.Enabled() }

func (s *AdminLogService) query(filter ApiLogFilter) *gorm.DB {
	query := s.manager.DB.Model(&model.ApiRequestLog{})
	if filter.StartAt != nil {
		query = query.Where("occurred_at >= ?", *filter.StartAt)
	}
	if filter.EndAt != nil {
		query = query.Where("occurred_at < ?", *filter.EndAt)
	}
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	if filter.Route != "" {
		query = query.Where("route LIKE ?", "%"+escapeLike(filter.Route)+"%")
	}
	if filter.StatusCode > 0 {
		query = query.Where("status_code = ?", filter.StatusCode)
	}
	switch filter.StatusGroup {
	case "2xx":
		query = query.Where("status_code >= 200 AND status_code < 300")
	case "3xx":
		query = query.Where("status_code >= 300 AND status_code < 400")
	case "4xx":
		query = query.Where("status_code >= 400 AND status_code < 500")
	case "5xx":
		query = query.Where("status_code >= 500")
	}
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.AppCode != "" {
		query = query.Where("app_code = ?", filter.AppCode)
	}
	if filter.IntegrationID > 0 {
		query = query.Where("integration_id = ?", filter.IntegrationID)
	}
	if filter.RequestID != "" {
		query = query.Where("request_id = ?", filter.RequestID)
	}
	if filter.Keyword != "" {
		keyword := "%" + escapeLike(filter.Keyword) + "%"
		query = query.Where("error_message LIKE ? OR request_headers LIKE ? OR request_body LIKE ?", keyword, keyword, keyword)
	}
	if filter.Cursor > 0 {
		query = query.Where("id < ?", filter.Cursor)
	}
	return query
}

func normalizeFilter(filter ApiLogFilter) ApiLogFilter {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	filter.Method = strings.ToUpper(strings.TrimSpace(filter.Method))
	filter.StatusGroup = strings.ToLower(strings.TrimSpace(filter.StatusGroup))
	filter.Route = strings.TrimSpace(filter.Route)
	filter.RequestID = strings.TrimSpace(filter.RequestID)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return filter
}
func escapeLike(value string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value)
}
