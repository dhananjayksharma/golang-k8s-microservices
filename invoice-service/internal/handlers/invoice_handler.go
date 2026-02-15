package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang-k8s-microservices/invoice-service/internal/logger"
	"golang-k8s-microservices/invoice-service/internal/models"

	"golang-k8s-microservices/invoice-service/internal/utils/pdf"

	"path/filepath"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	gomail "gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type InvoiceHandler struct {
	DB *gorm.DB
}

func NewInvoiceHandler(db *gorm.DB) *InvoiceHandler {
	return &InvoiceHandler{DB: db}
}

// POST /orders
func (h *InvoiceHandler) Create(c *gin.Context) {
	var req CreateInvoiceRequest
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
		//OrderStatus:   models.StatusCreated,
	}

	if err := h.DB.Create(&o).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, o)
}

// GET /orders/:id
func (h *InvoiceHandler) GetByID(c *gin.Context) {
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

func (h *InvoiceHandler) getOrderByID(id uint64) (models.Order, error) {
	var o models.Order
	err := h.DB.First(&o, "order_id = ?", id).Error
	return o, err
}

// GET /orders?customer_id=&status=&region=&engine=&limit=&offset=
func (h *InvoiceHandler) List(c *gin.Context) {
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
	// if req.OrderStatus != nil {
	// 	updates["status"] = *req.OrderStatus
	// }
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
func (h *InvoiceHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
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
func (h *InvoiceHandler) Listv2(c *gin.Context) {
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

func (h *InvoiceHandler) InvoiceActions(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(c.Query("action")))
	if action == "" {
		action = "preview"
	}

	// Fetch order
	var o models.Order
	if err := h.DB.First(&o, "order_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Map to PDF data
	inv := pdf.Invoice{
		ID:            fmt.Sprintf("%d", id),
		CustomerName:  fmt.Sprintf("Customer-%d", o.CustomerID),
		CustomerEmail: o.CustomerEmail,
		CreatedAt:     o.CreatedAt,
		Currency:      "INR",
		TaxPercent:    18,
	}

	items := []pdf.InvoiceItem{
		{
			Name:      fmt.Sprintf("DB: %s (%s %s) %s", o.DBName, o.DBEngine, o.DBVersion, o.Region),
			Qty:       1,
			UnitPrice: o.PriceMonthly,
		},
	}

	sub := o.PriceMonthly
	tax := (sub * inv.TaxPercent) / 100

	totals := pdf.Totals{
		SubTotal:   sub,
		TaxAmount:  tax,
		Discount:   0,
		GrandTotal: sub + tax,
	}

	data := pdf.InvoicePDFData{
		CompanyName: "Payment Service Pvt Ltd",
		CompanyTax:  "GSTIN: XX1234XXXX",
		CompanyAddr: "Bengaluru, Karnataka, India",
		CompanyHelp: "support@company.com | +91-XXXXXXXXXX",
		Invoice:     inv,
		Items:       items,
		Totals:      totals,
	}

	//filename := fmt.Sprintf("invoice-%d.pdf", id)

	switch action {

	case "preview":

		// Local directory
		localDir := "/Users/dkgosql/tmp/invoice-data/"
		//localDir := os.Getenv("INVOICE_PDF_DIR")

		// Ensure directory exists
		if err := os.MkdirAll(localDir, 0o755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		filename := fmt.Sprintf("invoice-%d.pdf", id)
		fullPath := filepath.Join(localDir, filename)
		fmt.Println(fullPath)
		// Create file locally
		file, err := os.Create(fullPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Write PDF into file
		if err := pdf.WriteInvoicePDF(file, data); err != nil {
			file.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		file.Close()

		// Now preview from saved file
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		c.Header("Cache-Control", "no-store")

		c.File(fullPath)
		return

	case "download":
		// Force download
		logger.Log.Info("invoice download",
			zap.Uint64("order_id", id),
		)

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", `attachment; filename="invoice-3.pdf"`)

		pdf.WriteInvoicePDF(c.Writer, data)
		// c.Header("Content-Type", "application/pdf")
		// c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		// c.Header("Cache-Control", "no-store")

		// if err := pdf.WriteInvoicePDF(c.Writer, data); err != nil {
		// 	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// 	return
		// }

	case "generate":
		// Return JSON
		c.JSON(http.StatusOK, gin.H{
			"invoice": data.Invoice,
			"items":   data.Items,
			"totals":  data.Totals,
		})
	case "sendemail":

		localDir := "/Users/dkgosql/tmp/invoice-data/"
		if err := os.MkdirAll(localDir, 0o755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		filename := fmt.Sprintf("invoice-%d.pdf", id)
		fullPath := filepath.Join(localDir, filename)

		// Create file
		file, err := os.Create(fullPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := pdf.WriteInvoicePDF(file, data); err != nil {
			file.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		file.Close()

		// Send email using generated file
		if err := sendmailInvoice(id, o.CustomerEmail, fullPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "invoice email sent successfully",
			"file":    fullPath,
		})
		return

	case "upload":
		c.JSON(http.StatusNotImplemented, gin.H{
			"message": "upload action not implemented yet",
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid action",
			"allowed": []string{"preview", "download", "generate", "upload"},
		})
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func sendmailInvoice(orderID uint64, toEmail string, pdfPath string) error {

	smtpHost := getenv("SMTP_HOST", "localhost")
	smtpPortStr := getenv("SMTP_PORT", "1025")
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	from := getenv("MAIL_FROM", "billing@local.test")

	if _, err := os.Stat(pdfPath); err != nil {
		return fmt.Errorf("invoice pdf not found: %w", err)
	}

	subject := fmt.Sprintf("Invoice for Order #%d", orderID)

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject)

	m.SetBody("text/plain",
		"Hi,\n\nPlease find your invoice attached.\n\nThanks,\nBilling Team\n",
	)

	m.Attach(pdfPath)

	d := gomail.NewDialer(smtpHost, smtpPort, "", "")

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Log.Info("invoice email sent",
		zap.Uint64("order_id", orderID),
		zap.String("email", toEmail),
	)

	return nil
}
