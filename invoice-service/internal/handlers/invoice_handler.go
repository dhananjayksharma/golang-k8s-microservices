package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dhananjayksharma/golang-k8s-microservices/invoice-service/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InvoiceHandler struct {
	DB *gorm.DB
}

func NewInvoiceHandler(db *gorm.DB) *InvoiceHandler {
	return &InvoiceHandler{DB: db}
}

// POST /invoices
func (h *InvoiceHandler) Create(c *gin.Context) {
	var req CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	o := models.Invoice{
		InvoiceNumber: req.InvoiceNumber,
		OrderId:       req.OrderId,
		TotalAmount:   req.TotalAmount,
		Status:        string(models.StatusNew),
	}

	if err := h.DB.Create(&o).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, o)
}

// GET /invoices/:id
func (h *InvoiceHandler) GetByID(c *gin.Context) {
	id := c.Query("id")

	var o models.Invoice
	if err := h.DB.First(&o, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, o)
}

// GET /invoices?invoice_number=&status=&region=&engine=&limit=&offset=
func (h *InvoiceHandler) List(c *gin.Context) {
	q := h.DB.Model(&models.Invoice{})

	if v := strings.TrimSpace(c.Query("invoice_number")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			q = q.Where("invoice_number = ?", id)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice_number"})
			return
		}
	}

	if v := strings.TrimSpace(c.Query("status")); v != "" {
		q = q.Where("status = ?", v)
	}

	limit := parseIntWithDefault(c.Query("limit"), 20)
	offset := parseIntWithDefault(c.Query("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	var items []models.Invoice
	if err := q.Order("invoice_number DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"limit":  limit,
		"offset": offset,
		"items":  items,
	})
}

// PATCH /invoices/:id
func (h *InvoiceHandler) Update(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{}

	if req.InvoiceNumber != nil {
		updates["invoice_number"] = *req.InvoiceNumber
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	res := h.DB.Model(&models.Invoice{}).Where("invoice_number = ?", id).Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	var o models.Invoice
	if err := h.DB.First(&o, "invoice_number = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

// DELETE /invoices/:id
func (h *InvoiceHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res := h.DB.Delete(&models.Invoice{}, "invoice_number = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

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

// GET /invoices?invoice_number=&status=&region=&engine=&limit=&offset=
func (h *InvoiceHandler) Listv2(c *gin.Context) {
	q := h.DB.Model(&models.Invoice{})

	if v := strings.TrimSpace(c.Query("invoice_number")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			q = q.Where("invoice_number = ?", id)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice_number"})
			return
		}
	}

	if v := strings.TrimSpace(c.Query("status")); v != "" {
		q = q.Where("status = ?", v)
	}

	limit := parseIntWithDefault(c.Query("limit"), 20)
	offset := parseIntWithDefault(c.Query("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	var items []models.Invoice
	if err := q.Order("invoice_number DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
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

// GET /invoices?invoice_number=&status=&region=&engine=&limit=&offset=
func (h *InvoiceHandler) Listv3(c *gin.Context) {
	q := h.DB.Model(&models.Invoice{})

	if v := strings.TrimSpace(c.Query("invoice_number")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			q = q.Where("invoice_number = ?", id)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice_number"})
			return
		}
	}

	if v := strings.TrimSpace(c.Query("status")); v != "" {
		q = q.Where("status = ?", v)
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

	var items []models.Invoice
	if err := q.Order("invoice_number DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"limit":  limit,
		"offset": offset,
		"items":  items,
		"count":  len(items),
		"tag":    "updated-version.3.0.0",
	})
}
