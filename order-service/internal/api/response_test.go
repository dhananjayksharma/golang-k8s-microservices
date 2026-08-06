package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessUsesSingleResponseContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("request_id", "request-123")

	Success(ctx, http.StatusOK, map[string]string{"status": "ready"})

	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Kind != ResponseKindStandard {
		t.Fatalf("unexpected kind: %s", response.Kind)
	}
	if response.Meta == nil || response.Meta.RequestID != "request-123" {
		t.Fatalf("unexpected meta: %+v", response.Meta)
	}
}

func TestFailureUsesSingleResponseContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Failure(ctx, http.StatusBadRequest, "INVALID_REQUEST", "bad request", nil)

	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Kind != ResponseKindError || response.Error == nil || response.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
