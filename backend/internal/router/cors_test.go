package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareAllowsAnyOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("Origin", "https://example.test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("allow origin = %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentialed CORS must remain disabled")
	}
}

func TestCORSMiddlewareAllowsBearerPreflightWithoutCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware())
	router.POST("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/test", nil)
	request.Header.Set("Origin", "https://example.test")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("allow origin = %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers")), "authorization") {
		t.Fatal("Authorization header is missing from CORS preflight response")
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentialed CORS must remain disabled")
	}
}
