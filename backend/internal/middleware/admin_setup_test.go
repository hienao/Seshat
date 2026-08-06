package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"basegoapp/config"
	"github.com/gin-gonic/gin"
)

func TestAdminSetupCompleteBlocksBootstrapAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("requires_admin_setup", true)
		c.Next()
	}, AdminSetupComplete())
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestJWTAuthDoesNotAcceptCookieToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuth(&config.Config{JWTSecret: "test-secret"}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: "auth_token", Value: "legacy-cookie-token"})
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
