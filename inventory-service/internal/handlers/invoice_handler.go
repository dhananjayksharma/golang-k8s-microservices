package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang-k8s-microservices/inventory-service/internal/api"
	"golang-k8s-microservices/inventory-service/internal/logger"
	"golang-k8s-microservices/inventory-service/internal/models"
	"golang-k8s-microservices/inventory-service/internal/utils/pdf"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	gomail "gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type InvoiceHandler struct {
	DB *gorm.DB
}

func NewInvoiceHandler(db *gorm.DB) *InvoiceHandler { return &InvoiceHandler{DB: db} }

func (h *InvoiceHandler) Create(c *gin.Context) {
	var req CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Failure(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", map[string]any{"cause": err.Error()})
		return
	}

	o := models.Order{
		CustomerID: req.CustomerID, CustomerEmail: req.CustomerEmail, DBName: req.DBName,
		DBEngine: req.DBEngine, DBVersion: req.DBVersion, StorageGB: req.StorageGB,
		Region: req.Region, PriceMonthly: req.PriceMonthly,
	}
	if err := h.DB.Create(&o).Error; err != nil {
		api.DomainError(c, err)
		return
	}
	api.Success(c, http.StatusCreated, o)
}

func (h *InvoiceHandler) GetByID(c *gin.Context) {
	id, ok := invoiceID(c)
	if !ok {
		return
	}

	o, err := h.findOrder(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Failure(c, http.StatusNotFound, "INVOICE_NOT_FOUND", "invoice not found", nil)
			return
		}
		api.DomainError(c, err)
		return
	}
	api.Success(c, http.StatusOK, o)
}

func (h *InvoiceHandler) List(c *gin.Context)   { h.list(c, false) }
func (h *InvoiceHandler) Listv2(c *gin.Context) { h.list(c, true) }

func (h *InvoiceHandler) list(c *gin.Context, includeVersion bool) {
	q := h.DB.Model(&models.Order{})
	if value := strings.TrimSpace(c.Query("customer_id")); value != "" {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			api.Failure(c, http.StatusBadRequest, "INVALID_CUSTOMER_ID", "customer_id must be an unsigned integer", nil)
			return
		}
		q = q.Where("customer_id = ?", id)
	}
	if value := strings.TrimSpace(c.Query("status")); value != "" {
		q = q.Where("order_status = ?", value)
	}
	if value := strings.TrimSpace(c.Query("region")); value != "" {
		q = q.Where("region = ?", value)
	}
	if value := strings.TrimSpace(c.Query("engine")); value != "" {
		q = q.Where("db_engine = ?", value)
	}

	limit := parseIntWithDefault(c.Query("limit"), 20)
	offset := parseIntWithDefault(c.Query("offset"), 0)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var items []models.Order
	if err := q.Order("order_id DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		api.DomainError(c, err)
		return
	}

	data := map[string]any{"items": items}
	if includeVersion {
		data["contract_version"] = "2.0.0"
	}
	api.SuccessWithMeta(c, http.StatusOK, data, api.Meta{Limit: limit, Offset: offset, Count: len(items)})
}

func (h *InvoiceHandler) Update(c *gin.Context) {
	id, ok := invoiceID(c)
	if !ok {
		return
	}

	var req UpdateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Failure(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", map[string]any{"cause": err.Error()})
		return
	}
	updates := invoiceUpdates(req)
	if len(updates) == 0 {
		api.Failure(c, http.StatusBadRequest, "NO_CHANGES", "no fields to update", nil)
		return
	}

	result := h.DB.Model(&models.Order{}).Where("order_id = ?", id).Updates(updates)
	if result.Error != nil {
		api.DomainError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		api.Failure(c, http.StatusNotFound, "INVOICE_NOT_FOUND", "invoice not found", nil)
		return
	}
	o, err := h.findOrder(id)
	if err != nil {
		api.DomainError(c, err)
		return
	}
	api.Success(c, http.StatusOK, o)
}

func (h *InvoiceHandler) Delete(c *gin.Context) {
	id, ok := invoiceID(c)
	if !ok {
		return
	}
	result := h.DB.Delete(&models.Order{}, "order_id = ?", id)
	if result.Error != nil {
		api.DomainError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		api.Failure(c, http.StatusNotFound, "INVOICE_NOT_FOUND", "invoice not found", nil)
		return
	}
	api.NoContent(c)
}

func (h *InvoiceHandler) Preview(c *gin.Context) {
	id, data, ok := h.invoiceData(c)
	if !ok {
		return
	}
	path, err := writeInvoiceFile(id, data)
	if err != nil {
		api.DomainError(c, err)
		return
	}
	filename := filepath.Base(path)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	c.Header("Cache-Control", "no-store")
	c.File(path)
}

func (h *InvoiceHandler) Download(c *gin.Context) {
	id, data, ok := h.invoiceData(c)
	if !ok {
		return
	}
	filename := fmt.Sprintf("inventory-%d.pdf", id)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := pdf.WriteInvoicePDF(c.Writer, data); err != nil {
		api.DomainError(c, err)
	}
}

func (h *InvoiceHandler) Generate(c *gin.Context) {
	_, data, ok := h.invoiceData(c)
	if !ok {
		return
	}
	api.Success(c, http.StatusOK, data)
}

func (h *InvoiceHandler) SendEmail(c *gin.Context) {
	id, data, ok := h.invoiceData(c)
	if !ok {
		return
	}
	path, err := writeInvoiceFile(id, data)
	if err != nil {
		api.DomainError(c, err)
		return
	}
	if err := sendmailInvoice(id, data.Invoice.CustomerEmail, path); err != nil {
		api.DomainError(c, err)
		return
	}
	api.Success(c, http.StatusOK, map[string]any{"message": "inventory email sent successfully", "file": path})
}

func (h *InvoiceHandler) Upload(c *gin.Context) {
	api.Failure(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "upload action is not implemented", nil)
}

func (h *InvoiceHandler) invoiceData(c *gin.Context) (uint64, pdf.InvoicePDFData, bool) {
	id, ok := invoiceID(c)
	if !ok {
		return 0, pdf.InvoicePDFData{}, false
	}
	o, err := h.findOrder(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Failure(c, http.StatusNotFound, "INVOICE_NOT_FOUND", "invoice not found", nil)
		} else {
			api.DomainError(c, err)
		}
		return 0, pdf.InvoicePDFData{}, false
	}
	return id, buildInvoiceData(id, o), true
}

func buildInvoiceData(id uint64, o models.Order) pdf.InvoicePDFData {
	invoice := pdf.Invoice{ID: fmt.Sprintf("%d", id), CustomerName: fmt.Sprintf("Customer-%d", o.CustomerID), CustomerEmail: o.CustomerEmail, CreatedAt: o.CreatedAt, Currency: "INR", TaxPercent: 18}
	items := []pdf.InvoiceItem{{Name: fmt.Sprintf("DB: %s (%s %s) %s", o.DBName, o.DBEngine, o.DBVersion, o.Region), Qty: 1, UnitPrice: o.PriceMonthly}}
	tax := (o.PriceMonthly * invoice.TaxPercent) / 100
	return pdf.InvoicePDFData{
		CompanyName: "Payment Service Pvt Ltd", CompanyTax: "GSTIN: XX1234XXXX",
		CompanyAddr: "Bengaluru, Karnataka, India", CompanyHelp: "support@company.com | +91-XXXXXXXXXX",
		Invoice: invoice, Items: items,
		Totals: pdf.Totals{SubTotal: o.PriceMonthly, TaxAmount: tax, GrandTotal: o.PriceMonthly + tax},
	}
}

func writeInvoiceFile(id uint64, data pdf.InvoicePDFData) (string, error) {
	dir := getenv("INVOICE_OUTPUT_DIR", filepath.Join(os.TempDir(), "inventory-data"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("inventory-%d.pdf", id))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := pdf.WriteInvoicePDF(file, data); err != nil {
		return "", err
	}
	return path, nil
}

func (h *InvoiceHandler) findOrder(id uint64) (models.Order, error) {
	var order models.Order
	err := h.DB.First(&order, "order_id = ?", id).Error
	return order, err
}

func invoiceID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		api.Failure(c, http.StatusBadRequest, "INVALID_ID", "id must be a positive integer", nil)
		return 0, false
	}
	return id, true
}

func parseIntWithDefault(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func invoiceUpdates(req UpdateInvoiceRequest) map[string]any {
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
	return updates
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func sendmailInvoice(orderID uint64, toEmail, pdfPath string) error {
	smtpPort, err := strconv.Atoi(getenv("SMTP_PORT", "1025"))
	if err != nil {
		return fmt.Errorf("invalid SMTP_PORT: %w", err)
	}
	if _, err := os.Stat(pdfPath); err != nil {
		return fmt.Errorf("inventory pdf not found: %w", err)
	}

	message := gomail.NewMessage()
	message.SetHeader("From", getenv("MAIL_FROM", "billing@local.test"))
	message.SetHeader("To", toEmail)
	message.SetHeader("Subject", fmt.Sprintf("Invoice for Order #%d", orderID))
	message.SetBody("text/plain", "Hi,\n\nPlease find your inventory attached.\n\nThanks,\nBilling Team\n")
	message.Attach(pdfPath)

	if err := gomail.NewDialer(getenv("SMTP_HOST", "localhost"), smtpPort, "", "").DialAndSend(message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	logger.Log.Info("inventory email sent", zap.Uint64("order_id", orderID), zap.String("email", toEmail))
	return nil
}
