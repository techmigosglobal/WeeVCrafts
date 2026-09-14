package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/wecratfs/commerce/internal/identity"
)

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	if h.searchService == nil {
		h.renderError(w, http.StatusNotFound, "Search unavailable", "Search has not been configured in this runtime.")
		return
	}
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	sort := r.URL.Query().Get("sort")
	page := 1
	if parsed, parseErr := strconv.Atoi(r.URL.Query().Get("page")); parseErr == nil && parsed > 0 && parsed <= 1000 {
		page = parsed
	}
	const pageSize = 24
	offset := (page - 1) * pageSize
	products, total, facets, err := h.searchService.SearchWithOptionsAndFacets(r.Context(), query, category, sort, pageSize, offset)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Search unavailable", "We could not search the PostgreSQL catalogue right now.")
		return
	}
	data := pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Search · " + identity.Name, SearchQuery: query, SearchCategory: category, SearchSort: sort, SearchPage: page, SearchHasNext: offset+len(products) < total, SearchHasPrevious: page > 1, Products: products, SearchFacets: facets, Notice: formatSearchNotice(total)}
	data.SearchNextURL = searchURL(query, category, sort, page+1)
	data.SearchPreviousURL = searchURL(query, category, sort, page-1)
	h.decorateSession(r, &data)
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.render(w, http.StatusOK, "search", data)
}

func (h *Handler) searchSuggestions(w http.ResponseWriter, r *http.Request) {
	if h.searchService == nil {
		h.render(w, http.StatusNotFound, "search-suggestions", pageData{})
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := pageData{}
	if len([]rune(query)) >= 2 {
		data.SearchQuery = query
		products, _, _, err := h.searchService.SearchWithOptionsAndFacets(r.Context(), query, "", "", 6, 0)
		if err != nil {
			h.render(w, http.StatusServiceUnavailable, "search-suggestions", pageData{SearchQuery: query, Notice: "Suggestions are temporarily unavailable."})
			return
		}
		data.Products = products
	}
	h.render(w, http.StatusOK, "search-suggestions", data)
}

func searchURL(query, category, sort string, page int) string {
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if category != "" {
		values.Set("category", category)
	}
	if sort != "" {
		values.Set("sort", sort)
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/search?" + encoded
	}
	return "/search"
}

func formatSearchNotice(total int) string {
	if total == 0 {
		return "No published products matched that search."
	}
	return "Showing " + strconv.Itoa(total) + " published catalogue matches."
}
