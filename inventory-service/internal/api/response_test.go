package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessWithMetaUsesSingleContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("request_id", "request-123")

	SuccessWithMeta(ctx, http.StatusOK, []string{"one"}, Meta{Limit: 20, Count: 1})

	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Kind != ResponseKindStandard || response.Meta == nil || response.Meta.RequestID != "request-123" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
