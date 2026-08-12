package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seshat/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestUpdateHandlerReturnsDevelopmentBuildWithoutRemoteAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUpdateHandler()
	router := gin.New()
	router.GET("/api/version", handler.GetVersion)
	router.GET("/api/admin/updates", handler.CheckUpdates)

	versionRecorder := httptest.NewRecorder()
	router.ServeHTTP(versionRecorder, httptest.NewRequest(http.MethodGet, "/api/version", nil))
	if versionRecorder.Code != http.StatusOK {
		t.Fatalf("version status = %d", versionRecorder.Code)
	}
	var versionPayload response.Response
	if err := json.Unmarshal(versionRecorder.Body.Bytes(), &versionPayload); err != nil || versionPayload.Code != 0 {
		t.Fatalf("unexpected version response: %s", versionRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, httptest.NewRequest(http.MethodGet, "/api/admin/updates", nil))
	if updateRecorder.Code != http.StatusOK || !json.Valid(updateRecorder.Body.Bytes()) {
		t.Fatalf("unexpected update response: %d %s", updateRecorder.Code, updateRecorder.Body.String())
	}
}
