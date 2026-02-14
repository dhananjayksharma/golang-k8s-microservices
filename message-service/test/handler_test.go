package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"message-service/internal/domain"
	"message-service/internal/repository"
	"message-service/internal/service"
	thttp "message-service/internal/transport/http"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	createFn func(context.Context, domain.Order) (domain.Order, error)
	listFn   func(context.Context, string, int64) ([]domain.Order, error)
}

func (f fakeRepo) Create(ctx context.Context, o domain.Order) (domain.Order, error) {
	return f.createFn(ctx, o)
}
func (f fakeRepo) ListByUser(ctx context.Context, userID string, limit int64) ([]domain.Order, error) {
	return f.listFn(ctx, userID, limit)
}

var _ repository.OrderRepository = (*fakeRepo)(nil)

func TestCreateOrder_201(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := fakeRepo{
		createFn: func(ctx context.Context, o domain.Order) (domain.Order, error) {
			o.OrderID = "O-1"
			o.UserID = "U-1"
			return o, nil
		},
		listFn: func(ctx context.Context, userID string, limit int64) ([]domain.Order, error) {
			return nil, nil
		},
	}

	svc := service.NewOrderService(repo)
	h := thttp.NewHandler(svc)

	r := gin.New()
	thttp.RegisterRoutes(r, h)

	body := `{"order_id":"O-1","user_id":"U-1","items":["a"],"amount":99.5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), `"order_id":"O-1"`)
}

func TestListOrders_200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := fakeRepo{
		createFn: func(ctx context.Context, o domain.Order) (domain.Order, error) { return o, nil },
		listFn: func(ctx context.Context, userID string, limit int64) ([]domain.Order, error) {
			return []domain.Order{{OrderID: "O-1", UserID: userID}}, nil
		},
	}

	svc := service.NewOrderService(repo)
	h := thttp.NewHandler(svc)

	r := gin.New()
	thttp.RegisterRoutes(r, h)

	req := httptest.NewRequest(http.MethodGet, "/v1/orders?user_id=U-1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"orders"`)
}
