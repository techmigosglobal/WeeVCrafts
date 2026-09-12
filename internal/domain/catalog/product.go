package catalog

type Product struct {
	ID             int64    `json:"id"`
	VariantID      int64    `json:"variant_id"`
	Slug           string   `json:"slug"`
	Brand          string   `json:"brand"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	CategorySlug   string   `json:"category_slug"`
	CategoryName   string   `json:"category_name"`
	PriceCents     int64    `json:"price_cents"`
	CompareAtCents int64    `json:"compare_at_cents"`
	HasCompareAt   bool     `json:"has_compare_at"`
	Rating         float64  `json:"rating"`
	ReviewCount    int      `json:"review_count"`
	DeliveryLabel  string   `json:"delivery_label"`
	ImageURL       string   `json:"image_url"`
	Swatches       []string `json:"swatches"`
	Available      int      `json:"available"`
}
