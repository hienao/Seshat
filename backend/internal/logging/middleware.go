package logging

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"seshat/internal/model"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if len(requestID) == 0 || len(requestID) > 100 || strings.ContainsAny(requestID, "\r\n") {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func AccessLogMiddleware(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if manager == nil || !manager.Enabled() || shouldSkip(c.Request.URL.Path) {
			return
		}
		requestID, _ := c.Get(requestIDKey)
		entry := model.ApiRequestLog{
			RequestID:      stringValue(requestID),
			OccurredAt:     start,
			Method:         c.Request.Method,
			Route:          routeName(c),
			StatusCode:     c.Writer.Status(),
			LatencyMs:      time.Since(start).Microseconds() / 1000,
			RequestBytes:   requestBytes(c.Request),
			ResponseBytes:  responseBytes(c),
			RequestHeaders: stringContextValue(c, "request_headers"),
			RequestBody:    stringContextValue(c, "request_body"),
			ClientIP:       c.ClientIP(),
			UserAgent:      truncate(c.GetHeader("User-Agent"), 500),
			UserID:         uintValue(c, "user_id"),
			Username:       stringContextValue(c, "username"),
			AppCode:        stringContextValue(c, "app_code"),
			IntegrationID:  uintValue(c, "integration_id"),
			EventID:        uintValue(c, "event_id"),
			ErrorCode:      stringContextValue(c, "error_code"),
			ErrorMessage:   truncate(stringContextValue(c, "error_message"), 1000),
		}
		if entry.StatusCode >= http.StatusInternalServerError && entry.ErrorMessage == "" {
			entry.ErrorMessage = http.StatusText(entry.StatusCode)
		}
		manager.Submit(entry)
	}
}

func shouldSkip(path string) bool {
	return strings.HasPrefix(path, "/api/admin/logs") || strings.HasPrefix(path, "/api/admin/application-logs") || strings.HasPrefix(path, "/swagger")
}

func routeName(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	if c.Writer.Status() == http.StatusNotFound {
		return "404"
	}
	return truncate(c.Request.URL.Path, 255)
}

func requestBytes(request *http.Request) int64 {
	if request.ContentLength < 0 {
		return 0
	}
	return request.ContentLength
}

func responseBytes(c *gin.Context) int64 {
	if c.Writer.Size() < 0 {
		return 0
	}
	return int64(c.Writer.Size())
}

func newRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "req-" + time.Now().Format("20060102150405.000000000")
	}
	return "req-" + base64.RawURLEncoding.EncodeToString(buffer)
}

func stringValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
func stringContextValue(c *gin.Context, key string) string {
	value, _ := c.Get(key)
	return stringValue(value)
}
func uintValue(c *gin.Context, key string) uint {
	value, _ := c.Get(key)
	if number, ok := value.(uint); ok {
		return number
	}
	return 0
}
func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
