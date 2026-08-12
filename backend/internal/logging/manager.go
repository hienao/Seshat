package logging

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"seshat/config"
	"seshat/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type queuedLog struct {
	api         *model.ApiRequestLog
	application *model.ApplicationLog
}

// Manager 负责接口日志和业务日志的异步写入、查询共用数据库连接和保留期清理。
type Manager struct {
	DB                 *gorm.DB
	enabled            bool
	queue              chan queuedLog
	done               chan struct{}
	retentionChanged   chan struct{}
	wg                 sync.WaitGroup
	apiDropped         atomic.Uint64
	applicationDropped atomic.Uint64
	retentionDays      atomic.Int64
	config             *config.Config
}

func NewManager(cfg *config.Config) (*Manager, error) {
	manager := &Manager{
		enabled:          cfg.APILogEnabled,
		config:           cfg,
		retentionChanged: make(chan struct{}, 1),
	}
	if !cfg.APILogEnabled {
		return manager, nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.APILogPath), 0755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(cfg.APILogPath), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)})
	if err != nil {
		return nil, err
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, err
	}
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.ApiRequestLog{}, &model.ApplicationLog{}); err != nil {
		return nil, err
	}
	manager.DB = db
	queueSize := cfg.APILogQueueSize
	if queueSize < 100 {
		queueSize = 100
	}
	if cfg.APILogBatchSize < 1 {
		cfg.APILogBatchSize = 100
	}
	manager.queue = make(chan queuedLog, queueSize)
	manager.done = make(chan struct{})
	manager.wg.Add(2)
	go manager.writeLoop()
	go manager.retentionLoop()
	return manager, nil
}

func (m *Manager) Enabled() bool { return m != nil && m.enabled && m.DB != nil }

func (m *Manager) Submit(entry model.ApiRequestLog) {
	if !m.Enabled() {
		return
	}
	select {
	case m.queue <- queuedLog{api: &entry}:
	default:
		m.apiDropped.Add(1)
	}
}

func (m *Manager) SubmitApplication(entry model.ApplicationLog) {
	if !m.Enabled() {
		return
	}
	select {
	case m.queue <- queuedLog{application: &entry}:
	default:
		m.applicationDropped.Add(1)
	}
}

func (m *Manager) Dropped() uint64 {
	if m == nil {
		return 0
	}
	return m.apiDropped.Load()
}

func (m *Manager) ApplicationDropped() uint64 {
	if m == nil {
		return 0
	}
	return m.applicationDropped.Load()
}

func (m *Manager) ExportLimit() int {
	if m == nil || m.config == nil || m.config.APILogExportLimit <= 0 {
		return 100000
	}
	return m.config.APILogExportLimit
}

// SetRetentionDays 更新日志保留天数，并立即触发一次过期日志清理。
func (m *Manager) SetRetentionDays(days int) {
	if m == nil {
		return
	}
	m.retentionDays.Store(int64(days))
	select {
	case m.retentionChanged <- struct{}{}:
	default:
	}
}

func (m *Manager) writeLoop() {
	defer m.wg.Done()
	interval := time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.flush(m.config.APILogBatchSize)
		case <-m.done:
			m.flush(0)
			return
		}
	}
}

func (m *Manager) flush(max int) {
	if !m.Enabled() {
		return
	}
	apiEntries := make([]model.ApiRequestLog, 0, m.config.APILogBatchSize)
	applicationEntries := make([]model.ApplicationLog, 0, m.config.APILogBatchSize)
	count := 0
	for max == 0 || count < max {
		select {
		case entry := <-m.queue:
			if entry.api != nil {
				apiEntries = append(apiEntries, *entry.api)
			}
			if entry.application != nil {
				applicationEntries = append(applicationEntries, *entry.application)
			}
			count++
		default:
			m.persist(apiEntries, applicationEntries)
			return
		}
	}
	m.persist(apiEntries, applicationEntries)
}

func (m *Manager) persist(apiEntries []model.ApiRequestLog, applicationEntries []model.ApplicationLog) {
	if len(apiEntries) > 0 {
		if err := m.DB.CreateInBatches(&apiEntries, len(apiEntries)).Error; err != nil {
			log.Printf("failed to persist API request logs: %v", err)
		}
	}
	if len(applicationEntries) > 0 {
		if err := m.DB.CreateInBatches(&applicationEntries, len(applicationEntries)).Error; err != nil {
			log.Printf("failed to persist application logs: %v", err)
		}
	}
}

func (m *Manager) retentionLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-m.retentionChanged:
			m.cleanup()
		case <-ticker.C:
			m.cleanup()
		case <-m.done:
			return
		}
	}
}

func (m *Manager) cleanup() {
	days := int(m.retentionDays.Load())
	if !m.Enabled() || days <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	if err := m.DB.Where("occurred_at < ?", cutoff).Delete(&model.ApiRequestLog{}).Error; err != nil {
		log.Printf("failed to clean API request logs: %v", err)
	}
	if err := m.DB.Where("occurred_at < ?", cutoff).Delete(&model.ApplicationLog{}).Error; err != nil {
		log.Printf("failed to clean application logs: %v", err)
	}
}

func (m *Manager) Close() {
	if m == nil || !m.Enabled() {
		return
	}
	close(m.done)
	m.wg.Wait()
	if sqlDB, err := m.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
