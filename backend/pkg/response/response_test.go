package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestErrorIncludesStableErrorCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	ErrorWithCode(context, http.StatusBadRequest, "invalid_input", "legacy localized message")

	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error_code"] != "invalid_input" || body["message"] != "legacy localized message" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func TestGenericErrorUsesHTTPCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	NotFound(context, "missing")

	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error_code"] != "http.404" {
		t.Fatalf("error_code = %#v", body["error_code"])
	}
}
