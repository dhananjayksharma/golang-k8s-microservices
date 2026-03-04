package domain

// Keep pricing logic centralized.
// For MVP: recompute totals from items + applied promotions.
// Later: integrate pricing engine / tax rules / shipping rules.

type PricingSummary struct {
	SubtotalPaise   int64
	TaxPaise        int64
	ShippingPaise   int64
	DiscountPaise   int64
	GrandTotalPaise int64
}
