package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"seshat/internal/model"
)

type Fields map[string]interface{}

var defaultManager atomic.Pointer[Manager]

func SetDefaultManager(manager *Manager) { defaultManager.Store(manager) }

func Debug(source, message string, fields Fields) {
	writeApplicationLog("DEBUG", source, message, fields)
}
func Info(source, message string, fields Fields) {
	writeApplicationLog("INFO", source, message, fields)
}
func Warn(source, message string, fields Fields) {
	writeApplicationLog("WARN", source, message, fields)
}
func Error(source, message string, fields Fields) {
	writeApplicationLog("ERROR", source, message, fields)
}

func writeApplicationLog(level, source, message string, fields Fields) {
	level = normalizeLevel(level)
	source = truncate(strings.TrimSpace(source), 255)
	message = truncate(strings.TrimSpace(message), 4000)
	safeFields := sanitizeFields(fields)
	encodedFields, err := json.Marshal(safeFields)
	if err != nil {
		encodedFields = []byte(`{"logging_error":"failed to encode structured fields"}`)
	}
	log.Printf("[%s] [%s] %s %s", level, source, message, string(encodedFields))

	manager := defaultManager.Load()
	if manager == nil || !manager.Enabled() {
		return
	}
	manager.SubmitApplication(model.ApplicationLog{
		OccurredAt:    time.Now(),
		Level:         level,
		Source:        source,
		Message:       message,
		Fields:        string(encodedFields),
		RequestID:     fieldString(safeFields, "request_id"),
		UserID:        fieldUint(safeFields, "user_id"),
		AppCode:       fieldString(safeFields, "app_code"),
		IntegrationID: fieldUint(safeFields, "integration_id"),
		EventID:       fieldUint(safeFields, "event_id"),
	})
}

func normalizeLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return "DEBUG"
	case "WARN", "WARNING":
		return "WARN"
	case "ERROR":
		return "ERROR"
	default:
		return "INFO"
	}
}

func sanitizeFields(fields Fields) Fields {
	result := make(Fields, len(fields))
	for key, value := range fields {
		if isSensitiveField(key) {
			result[key] = "[REDACTED]"
			continue
		}
		result[key] = sanitizeFieldValue(value)
	}
	return result
}

func sanitizeFieldValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case error:
		return typed.Error()
	case Fields:
		return sanitizeFields(typed)
	case map[string]interface{}:
		return sanitizeFields(Fields(typed))
	case map[string]string:
		result := make(Fields, len(typed))
		for key, nestedValue := range typed {
			if isSensitiveField(key) {
				result[key] = "[REDACTED]"
			} else {
				result[key] = nestedValue
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(typed))
		for index, item := range typed {
			result[index] = sanitizeFieldValue(item)
		}
		return result
	default:
		return value
	}
}

func isSensitiveField(key string) bool {
	key = strings.ToLower(key)
	for _, sensitive := range []string{"authorization", "cookie", "token", "secret", "password", "signature", "api-key", "apikey"} {
		if strings.Contains(key, sensitive) {
			return true
		}
	}
	return false
}

func fieldString(fields Fields, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}
	return truncate(fmt.Sprint(value), 255)
}

func fieldUint(fields Fields, key string) uint {
	switch value := fields[key].(type) {
	case uint:
		return value
	case uint64:
		return uint(value)
	case int:
		if value > 0 {
			return uint(value)
		}
	case int64:
		if value > 0 {
			return uint(value)
		}
	case float64:
		if value > 0 {
			return uint(value)
		}
	}
	return 0
}
