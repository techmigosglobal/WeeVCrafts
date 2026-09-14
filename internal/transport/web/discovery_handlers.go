package web

import (
	"net/http"
	"sort"
	"strings"
	"unicode"

	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/identity"
)

// BrandSummary is derived from approved catalogue records. There is no
// runtime fixture or placeholder brand directory in the live storefront.
type BrandSummary struct {
	Name         string
	Slug         string
	ProductCount int
}

func (h *Handler) deals(w http.ResponseWriter, r *http.Request) {
	products, err := h.catalog.List(r.Context(), "", 100)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Deals unavailable", "We could not load current catalogue pricing.")
		return
	}
	deals := make([]domain.Product, 0, len(products))
	for _, product := range products {
		if product.HasCompareAt && product.CompareAtCents > product.PriceCents {
			deals = append(deals, product)
		}
	}
	data := pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Deals · " + identity.Name, Category: "deals", Products: deals, Notice: "Only catalogue records with a real compare-at price are shown."}
	h.decorateSession(r, &data)
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.render(w, http.StatusOK, "deals", data)
}

func (h *Handler) brands(w http.ResponseWriter, r *http.Request) {
	products, err := h.catalog.List(r.Context(), "", 100)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Brands unavailable", "We could not load the published maker directory.")
		return
	}
	counts := make(map[string]int)
	for _, product := range products {
		name := strings.TrimSpace(product.Brand)
		if name != "" {
			counts[name]++
		}
	}
	summaries := make([]BrandSummary, 0, len(counts))
	for name, count := range counts {
		summaries = append(summaries, BrandSummary{Name: name, Slug: brandSlug(name), ProductCount: count})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
	data := pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Makers · " + identity.Name, Category: "brands", Brands: summaries}
	h.decorateSession(r, &data)
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.render(w, http.StatusOK, "brands", data)
}

func (h *Handler) brand(w http.ResponseWriter, r *http.Request, slug string) {
	if strings.TrimSpace(slug) == "" || strings.Contains(slug, "/") {
		h.notFound(w, r)
		return
	}
	products, err := h.catalog.List(r.Context(), "", 100)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Brand unavailable", "We could not load this maker's published catalogue.")
		return
	}
	var name string
	filtered := make([]domain.Product, 0)
	for _, product := range products {
		if brandSlug(product.Brand) == slug {
			if name == "" {
				name = product.Brand
			}
			filtered = append(filtered, product)
		}
	}
	if name == "" {
		h.notFound(w, r)
		return
	}
	data := pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: name + " · " + identity.Name, Category: "brands", BrandName: name, BrandSlug: slug, Products: filtered}
	h.decorateSession(r, &data)
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.render(w, http.StatusOK, "brand", data)
}

func brandSlug(value string) string {
	var builder strings.Builder
	lastHyphen := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
			lastHyphen = false
			continue
		}
		if builder.Len() > 0 && !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
