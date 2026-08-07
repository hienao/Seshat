package service

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"seshat/internal/logging"
	"seshat/internal/model"
	"seshat/pkg/database"

	"gorm.io/gorm"
)

type ApplicationLogFilter struct {
	StartAt   *time.Time
	EndAt     *time.Time
	Level     string
	Source    string
	RequestID string
	Keyword   string
	Cursor    uint64
	Limit     int
}

type ApplicationLogListResponse struct {
	Items      []model.ApplicationLog `json:"items"`
	Total      int64                  `json:"total"`
	NextCursor uint64                 `json:"next_cursor,omitempty"`
	HasMore    bool                   `json:"has_more"`
}

type ApplicationLogSummary struct {
	Total      int64  `json:"total"`
	DebugCount int64  `json:"debug_count"`
	InfoCount  int64  `json:"info_count"`
	WarnCount  int64  `json:"warn_count"`
	ErrorCount int64  `json:"error_count"`
	Dropped    uint64 `json:"dropped"`
}

type ApplicationLogService struct{ manager *logging.Manager }

func NewApplicationLogService(manager *logging.Manager) *ApplicationLogService {
	return &ApplicationLogService{manager: manager}
}

func (s *ApplicationLogService) List(filter ApplicationLogFilter) (*ApplicationLogListResponse, error) {
	if !s.enabled() {
		return &ApplicationLogListResponse{Items: []model.ApplicationLog{}}, nil
	}
	filter = normalizeApplicationLogFilter(filter)
	base := s.query(filter)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []model.ApplicationLog
	if err := base.Omit("fields").Order("id DESC").Limit(filter.Limit + 1).Find(&items).Error; err != nil {
		return nil, err
	}
	result := &ApplicationLogListResponse{Total: total, HasMore: len(items) > filter.Limit}
	if result.HasMore {
		items = items[:filter.Limit]
	}
	result.Items = items
	if result.HasMore && len(items) > 0 {
		result.NextCursor = items[len(items)-1].ID
	}
	return result, nil
}

func (s *ApplicationLogService) Summary(filter ApplicationLogFilter) (*ApplicationLogSummary, error) {
	if !s.enabled() {
		return &ApplicationLogSummary{}, nil
	}
	filter = normalizeApplicationLogFilter(filter)
	var result ApplicationLogSummary
	row := s.query(filter).Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN level = 'DEBUG' THEN 1 ELSE 0 END), 0) AS debug_count, COALESCE(SUM(CASE WHEN level = 'INFO' THEN 1 ELSE 0 END), 0) AS info_count, COALESCE(SUM(CASE WHEN level = 'WARN' THEN 1 ELSE 0 END), 0) AS warn_count, COALESCE(SUM(CASE WHEN level = 'ERROR' THEN 1 ELSE 0 END), 0) AS error_count").Scan(&result)
	if row.Error != nil {
		return nil, row.Error
	}
	result.Dropped = s.manager.ApplicationDropped()
	return &result, nil
}

func (s *ApplicationLogService) Get(id uint64) (*model.ApplicationLog, error) {
	if !s.enabled() {
		return nil, errors.New("业务日志未启用")
	}
	var item model.ApplicationLog
	if err := s.manager.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ApplicationLogService) Clear(actorID uint, filter ApplicationLogFilter) (int64, error) {
	if !s.enabled() {
		return 0, errors.New("业务日志未启用")
	}
	filter = normalizeApplicationLogFilter(filter)
	result := s.query(filter).Where("1 = 1").Delete(&model.ApplicationLog{})
	if result.Error != nil {
		return 0, result.Error
	}
	detail, _ := json.Marshal(filter)
	audit := &model.AdminAuditLog{ActorID: actorID, Action: "clear_application_logs", Target: "application_logs", Detail: string(detail), OccurredAt: time.Now()}
	if err := database.GetDB().Create(audit).Error; err != nil {
		return result.RowsAffected, err
	}
	return result.RowsAffected, nil
}

func (s *ApplicationLogService) Export(writer io.Writer, filter ApplicationLogFilter, format string) (int, error) {
	if !s.enabled() {
		return 0, errors.New("业务日志未启用")
	}
	filter = normalizeApplicationLogFilter(filter)
	if format != "csv" && format != "jsonl" {
		return 0, errors.New("导出格式只支持 csv 或 jsonl")
	}
	maxRows := s.manager.ExportLimit()
	var csvWriter *csv.Writer
	if format == "csv" {
		csvWriter = csv.NewWriter(writer)
		if err := csvWriter.Write([]string{"id", "occurred_at", "level", "source", "message", "request_id", "user_id", "app_code", "integration_id", "event_id", "fields"}); err != nil {
			return 0, err
		}
	}
	count := 0
	var cursor uint64
	for count < maxRows {
		pageFilter := filter
		pageFilter.Cursor = cursor
		pageFilter.Limit = 1000
		var rows []model.ApplicationLog
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
				if err := csvWriter.Write([]string{strconv.FormatUint(row.ID, 10), row.OccurredAt.Format(time.RFC3339Nano), row.Level, row.Source, row.Message, row.RequestID, strconv.FormatUint(uint64(row.UserID), 10), row.AppCode, strconv.FormatUint(uint64(row.IntegrationID), 10), strconv.FormatUint(uint64(row.EventID), 10), row.Fields}); err != nil {
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

func (s *ApplicationLogService) enabled() bool {
	return s != nil && s.manager != nil && s.manager.Enabled()
}

func (s *ApplicationLogService) query(filter ApplicationLogFilter) *gorm.DB {
	query := s.manager.DB.Model(&model.ApplicationLog{})
	if filter.StartAt != nil {
		query = query.Where("occurred_at >= ?", *filter.StartAt)
	}
	if filter.EndAt != nil {
		query = query.Where("occurred_at < ?", *filter.EndAt)
	}
	if filter.Level != "" {
		query = query.Where("level = ?", filter.Level)
	}
	if filter.Source != "" {
		query = query.Where("source LIKE ?", "%"+escapeLike(filter.Source)+"%")
	}
	if filter.RequestID != "" {
		query = query.Where("request_id = ?", filter.RequestID)
	}
	if filter.Keyword != "" {
		keyword := "%" + escapeLike(filter.Keyword) + "%"
		query = query.Where("message LIKE ? OR fields LIKE ?", keyword, keyword)
	}
	if filter.Cursor > 0 {
		query = query.Where("id < ?", filter.Cursor)
	}
	return query
}

func normalizeApplicationLogFilter(filter ApplicationLogFilter) ApplicationLogFilter {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	filter.Level = strings.ToUpper(strings.TrimSpace(filter.Level))
	filter.Source = strings.TrimSpace(filter.Source)
	filter.RequestID = strings.TrimSpace(filter.RequestID)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return filter
}
