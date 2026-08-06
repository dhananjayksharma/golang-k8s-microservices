package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang-k8s-microservices/inventory-service/internal/api"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newDryRunHandler(t *testing.T) *InvoiceHandler {
	t.Helper()
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "user:pass@tcp(localhost:3306)/invoice_test?parseTime=true",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("failed to create dry-run gorm db: %v", err)
	}
	return NewInvoiceHandler(gdb)
}

func performListRequest(t *testing.T, h *InvoiceHandler, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/orders", h.List)
	req := httptest.NewRequest(http.MethodGet, "/orders"+rawQuery, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) api.Response {
	t.Helper()
	var response api.Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	return response
}

func TestInvoiceHandlerList_InvalidCustomerID(t *testing.T) {
	rr := performListRequest(t, newDryRunHandler(t), "?customer_id=abc")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	response := decodeResponse(t, rr)
	if response.Kind != api.ResponseKindError {
		t.Fatalf("expected error response, got %s", response.Kind)
	}
	if response.Error == nil || response.Error.Code != "INVALID_CUSTOMER_ID" {
		t.Fatalf("expected INVALID_CUSTOMER_ID, got %+v", response.Error)
	}
}

func TestInvoiceHandlerList_DefaultPagination(t *testing.T) {
	rr := performListRequest(t, newDryRunHandler(t), "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	response := decodeResponse(t, rr)
	if response.Kind != api.ResponseKindStandard {
		t.Fatalf("expected standard response, got %s", response.Kind)
	}
	if response.Meta == nil || response.Meta.Limit != 20 || response.Meta.Offset != 0 {
		t.Fatalf("unexpected metadata: %+v", response.Meta)
	}
}

func TestInvoiceHandlerList_LimitCappedAt100(t *testing.T) {
	rr := performListRequest(t, newDryRunHandler(t), "?customer_id=42&limit=250&offset=7&region=ap-south-1&engine=mysql&status=CREATED")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	response := decodeResponse(t, rr)
	if response.Meta == nil || response.Meta.Limit != 100 || response.Meta.Offset != 7 {
		t.Fatalf("unexpected metadata: %+v", response.Meta)
	}
}
