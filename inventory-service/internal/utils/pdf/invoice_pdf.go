package pdf

// utils/pdf/invoice_pdf.go

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ====== Generic structs (you can map your DB model -> this) ======

type Invoice struct {
	ID            string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	BillingAddr   string

	CreatedAt time.Time
	Currency  string

	TaxPercent     float64
	DiscountAmount float64
	Notes          string
}

type InvoiceItem struct {
	Name      string
	Qty       int64
	UnitPrice float64
}

type Totals struct {
	SubTotal   float64
	TaxAmount  float64
	Discount   float64
	GrandTotal float64
}

type InvoicePDFData struct {
	Invoice Invoice
	Items   []InvoiceItem
	Totals  Totals

	// Company details shown on PDF header
	CompanyName string
	CompanyTax  string
	CompanyAddr string
	CompanyHelp string // Email/phone line
}

// WriteInvoicePDF generates inventory PDF and writes to w (HTTP response, file, buffer).
func WriteInvoicePDF(w io.Writer, data InvoicePDFData) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 11)

	drawHeader(pdf, data)
	drawMeta(pdf, data)
	drawItemsTable(pdf, data)
	drawTotals(pdf, data)
	drawNotes(pdf, data)

	return pdf.Output(w)
}

// ====== PDF layout helpers ======

func drawHeader(pdf *gofpdf.Fpdf, data InvoicePDFData) {
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 8, "INVOICE", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 6, dashIfEmpty(data.CompanyName, "Your Company Name"), "", 1, "L", false, 0, "")
	if data.CompanyTax != "" {
		pdf.CellFormat(0, 6, data.CompanyTax, "", 1, "L", false, 0, "")
	}
	if data.CompanyAddr != "" {
		pdf.CellFormat(0, 6, data.CompanyAddr, "", 1, "L", false, 0, "")
	}
	if data.CompanyHelp != "" {
		pdf.CellFormat(0, 6, data.CompanyHelp, "", 1, "L", false, 0, "")
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(6)
}

func drawMeta(pdf *gofpdf.Fpdf, data InvoicePDFData) {
	inv := data.Invoice

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(95, 7, "Billed To", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 7, "Invoice Details", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)

	leftX := pdf.GetX()
	leftY := pdf.GetY()

	billed := fmt.Sprintf("%s\n%s\n%s\n%s",
		dash(inv.CustomerName),
		dash(inv.CustomerEmail),
		dash(inv.CustomerPhone),
		dash(inv.BillingAddr),
	)
	pdf.MultiCell(95, 5.5, billed, "1", "L", false)

	pdf.SetXY(leftX+95, leftY)

	details := []struct {
		k string
		v string
	}{
		{"Invoice ID", dash(inv.ID)},
		{"Invoice Date", inv.CreatedAt.Format("02 Jan 2006")},
		{"Currency", dash(inv.Currency)},
		{"Tax %", fmt.Sprintf("%.2f", inv.TaxPercent)},
	}

	boxH := float64(len(details))*6 + 2
	startX := pdf.GetX()
	startY := pdf.GetY()
	pdf.Rect(startX, startY, 95, boxH, "D")

	pdf.SetXY(startX+3, startY+2)
	for _, row := range details {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(30, 6, row.k+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, row.v, "", 1, "L", false, 0, "")
		pdf.SetX(startX + 3)
	}

	pdf.SetXY(leftX, leftY+maxf(pdf.GetY()-leftY, boxH)+6)
}

func drawItemsTable(pdf *gofpdf.Fpdf, data InvoicePDFData) {
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "Items", "", 1, "L", false, 0, "")

	wName := 95.0
	wQty := 20.0
	wUnit := 35.0
	wAmt := 35.0

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(wName, 7, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(wQty, 7, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wUnit, 7, "Unit Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(wAmt, 7, "Amount", "1", 1, "R", true, 0, "")

	pdf.SetFont("Helvetica", "", 10)

	if len(data.Items) == 0 {
		pdf.CellFormat(0, 7, "No items found.", "1", 1, "L", false, 0, "")
		pdf.Ln(6)
		return
	}

	for _, it := range data.Items {
		amount := float64(it.Qty) * it.UnitPrice
		pdf.CellFormat(wName, 7, dash(it.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(wQty, 7, fmt.Sprintf("%d", it.Qty), "1", 0, "C", false, 0, "")
		pdf.CellFormat(wUnit, 7, money(data.Invoice.Currency, it.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(wAmt, 7, money(data.Invoice.Currency, amount), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(6)
}

func drawTotals(pdf *gofpdf.Fpdf, data InvoicePDFData) {
	t := data.Totals
	cur := data.Invoice.Currency

	boxW := 80.0
	x := 210.0 - 12.0 - boxW // A4 width - right margin - box width
	y := pdf.GetY()

	pdf.SetXY(x, y)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(boxW, 7, "Totals", "1", 1, "L", false, 0, "")

	rows := []struct {
		k string
		v string
	}{
		{"Subtotal", money(cur, t.SubTotal)},
		{"Tax", money(cur, t.TaxAmount)},
		{"Discount", money(cur, t.Discount)},
		{"Grand Total", money(cur, t.GrandTotal)},
	}

	for i, r := range rows {
		if i == len(rows)-1 {
			pdf.SetFont("Helvetica", "B", 11)
		} else {
			pdf.SetFont("Helvetica", "", 10)
		}
		pdf.SetXY(x, pdf.GetY())
		pdf.CellFormat(boxW*0.5, 7, r.k, "1", 0, "L", false, 0, "")
		pdf.CellFormat(boxW*0.5, 7, r.v, "1", 1, "R", false, 0, "")
	}

	pdf.Ln(6)
}

func drawNotes(pdf *gofpdf.Fpdf, data InvoicePDFData) {
	if data.Invoice.Notes == "" {
		return
	}
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "Notes", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5.5, data.Invoice.Notes, "1", "L", false)
	pdf.Ln(2)
}

// ====== formatting helpers ======

func money(currency string, v float64) string {
	return fmt.Sprintf("%s %.2f", dash(currency), round2(v))
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func dashIfEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
