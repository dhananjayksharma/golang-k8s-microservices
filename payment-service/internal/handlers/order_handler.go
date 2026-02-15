package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go-gin-mysql-k8s/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	DB *gorm.DB
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{DB: db}
}

// POST /orders
func (h *OrderHandler) Create(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	o := models.Order{
		CustomerID:    req.CustomerID,
		CustomerEmail: req.CustomerEmail,
		DBName:        req.DBName,
		DBEngine:      req.DBEngine,
		DBVersion:     req.DBVersion,
		StorageGB:     req.StorageGB,
		Region:        req.Region,
		PriceMonthly:  req.PriceMonthly,
		OrderStatus:   models.StatusCreated,
	}

	if err := h.DB.Create(&o).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, o)
}

// GET /orders/:id
func (h *OrderHandler) GetByID(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var o models.Order
	if err := h.DB.First(&o, "order_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, o)
}

// GET /orders?customer_id=&status=&region=&engine=&limit=&offset=
func (h *OrderHandler) List(c *gin.Context) {
	q := h.DB.Model(&models.Order{})

	if v := strings.TrimSpace(c.Query("customer_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			q = q.Where("customer_id = ?", id)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
	}

	if v := strings.TrimSpace(c.Query("status")); v != "" {
		q = q.Where("order_status = ?", v)
	}
	if v := strings.TrimSpace(c.Query("region")); v != "" {
		q = q.Where("region = ?", v)
	}
	if v := strings.TrimSpace(c.Query("engine")); v != "" {
		q = q.Where("db_engine = ?", v)
	}

	limit := parseIntWithDefault(c.Query("limit"), 20)
	offset := parseIntWithDefault(c.Query("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	var items []models.Order
	if err := q.Order("order_id DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"limit":  limit,
		"offset": offset,
		"items":  items,
	})
}

// PATCH /orders/:id
func (h *OrderHandler) Update(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{}

	if req.CustomerEmail != nil {
		updates["customer_email"] = *req.CustomerEmail
	}
	if req.DBName != nil {
		updates["db_name"] = *req.DBName
	}
	if req.DBEngine != nil {
		updates["db_engine"] = *req.DBEngine
	}
	if req.DBVersion != nil {
		updates["db_version"] = *req.DBVersion
	}
	if req.StorageGB != nil {
		updates["storage_gb"] = *req.StorageGB
	}
	if req.Region != nil {
		updates["region"] = *req.Region
	}
	if req.OrderStatus != nil {
		updates["order_status"] = *req.OrderStatus
	}
	if req.PriceMonthly != nil {
		updates["price_monthly"] = *req.PriceMonthly
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	res := h.DB.Model(&models.Order{}).Where("order_id = ?", id).Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	// return updated row
	var o models.Order
	if err := h.DB.First(&o, "order_id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

// DELETE /orders/:id
func (h *OrderHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res := h.DB.Delete(&models.Order{}, "order_id = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// helpers
func parseUint64Param(c *gin.Context, name string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(c.Param(name)), 10, 64)
}

func parseIntWithDefault(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// GET /orders?customer_id=&status=&region=&engine=&limit=&offset=
func (h *OrderHandler) Listv2(c *gin.Context) {
	q := h.DB.Model(&models.Order{})

	if v := strings.TrimSpace(c.Query("customer_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			q = q.Where("customer_id = ?", id)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
	}

	if v := strings.TrimSpace(c.Query("status")); v != "" {
		q = q.Where("order_status = ?", v)
	}
	if v := strings.TrimSpace(c.Query("region")); v != "" {
		q = q.Where("region = ?", v)
	}
	if v := strings.TrimSpace(c.Query("engine")); v != "" {
		q = q.Where("db_engine = ?", v)
	}

	limit := parseIntWithDefault(c.Query("limit"), 20)
	offset := parseIntWithDefault(c.Query("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	var items []models.Order
	if err := q.Order("order_id DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"limit":  limit,
		"offset": offset,
		"items":  items,
		"count":  len(items),
		"tag":    "updated-version.2.2.0",
	})
}
