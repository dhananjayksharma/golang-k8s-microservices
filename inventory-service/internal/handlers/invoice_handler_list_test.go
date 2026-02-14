package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newDryRunHandler(t *testing.T) *InvoiceHandler {
	t.Helper()

	gdb, err := gorm.Open(
		mysql.New(mysql.Config{
			DSN:                       "user:pass@tcp(localhost:3306)/invoice_test?parseTime=true",
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			DryRun:               true,
			DisableAutomaticPing: true,
		},
	)
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

func TestInvoiceHandlerList_InvalidCustomerID(t *testing.T) {
	h := newDryRunHandler(t)
	rr := performListRequest(t, h, "?customer_id=abc")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["error"] != "invalid customer_id" {
		t.Fatalf("expected error 'invalid customer_id', got %v", body["error"])
	}
}

func TestInvoiceHandlerList_DefaultPagination(t *testing.T) {
	h := newDryRunHandler(t)
	rr := performListRequest(t, h, "")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if int(body["limit"].(float64)) != 20 {
		t.Fatalf("expected default limit 20, got %v", body["limit"])
	}
	if int(body["offset"].(float64)) != 0 {
		t.Fatalf("expected default offset 0, got %v", body["offset"])
	}
}

func TestInvoiceHandlerList_LimitCappedAt100(t *testing.T) {
	h := newDryRunHandler(t)
	rr := performListRequest(t, h, "?customer_id=42&limit=250&offset=7&region=ap-south-1&engine=mysql&status=CREATED")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if int(body["limit"].(float64)) != 100 {
		t.Fatalf("expected capped limit 100, got %v", body["limit"])
	}
	if int(body["offset"].(float64)) != 7 {
		t.Fatalf("expected offset 7, got %v", body["offset"])
	}
}
