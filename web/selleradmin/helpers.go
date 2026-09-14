package selleradmin

import (
	"net/url"
	"strings"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

func firstSellerReturn(returns []viewmodels.SellerAdminReturn) viewmodels.SellerAdminReturn {
	if len(returns) == 0 {
		return viewmodels.SellerAdminReturn{}
	}
	return returns[0]
}

func sellerNavClass(key, current string) string {
	if key == current {
		return "seller-nav-link active"
	}
	return "seller-nav-link"
}

func ariaCurrent(active bool) string {
	if active {
		return "page"
	}
	return ""
}

func sellerPath(active string) string {
	if active == "dashboard" || active == "" {
		return "/seller-admin"
	}
	return "/seller-admin/" + active
}

func sellerWorkspaceTabClass(active bool) string {
	if active {
		return "active"
	}
	return ""
}

func sellerTabActive(current, value string) bool {
	current = strings.ToLower(strings.TrimSpace(current))
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "all" {
		return current == "" || current == "all"
	}
	return current == value
}

func sellerTabClass(current, value string) string {
	if sellerTabActive(current, value) {
		return "active"
	}
	return ""
}

func sellerTabAriaCurrent(current, value string) string {
	if sellerTabActive(current, value) {
		return "page"
	}
	return ""
}

func sellerStatusClass(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "approved"), strings.Contains(value, "active"), strings.Contains(value, "paid"), strings.Contains(value, "delivered"), strings.Contains(value, "settled"), strings.Contains(value, "in stock"), strings.Contains(value, "completed"):
		return "success"
	case strings.Contains(value, "rejected"), strings.Contains(value, "out of stock"), strings.Contains(value, "cancel"):
		return "danger"
	case strings.Contains(value, "review"), strings.Contains(value, "processing"), strings.Contains(value, "shipped"), strings.Contains(value, "pending"), strings.Contains(value, "awaiting"), strings.Contains(value, "new"), strings.Contains(value, "ready"):
		return "warning"
	default:
		return "neutral"
	}
}

func sellerToneClass(tone string) string {
	if tone == "" {
		return "red"
	}
	return tone
}

func sellerNotice(prefix, value string) string {
	return url.QueryEscape(prefix + value)
}

func sellerAdjustmentClass(value string) string {
	if strings.HasPrefix(value, "-") {
		return "negative-value"
	}
	if strings.HasPrefix(value, "+") {
		return "positive-value"
	}
	return ""
}

func sellerStatCount(length int) string {
	switch {
	case length >= 10:
		return "10"
	case length >= 7:
		return "7"
	case length == 6:
		return "6"
	default:
		return "5"
	}
}

func sellerIntroImage(active string) string {
	if active == "dashboard" || active == "products" {
		return "/assets/images/customer/hero-studio.webp"
	}
	return "/assets/images/customer/saree-maroon.webp"
}

func sellerTrendClass(positive bool) string {
	if positive {
		return "positive"
	}
	return "negative"
}

func sellerTrendIcon(positive bool) string {
	if positive {
		return "arrow_upward"
	}
	return "arrow_downward"
}

func sellerStockClass(stock string) string {
	if stock == "0" {
		return "stock-out"
	}
	return "stock-ok"
}

func sellerStockLabel(stock string) string {
	if stock == "0" {
		return "out of stock"
	}
	return "in stock"
}

func sellerAvailability(value, code string) string {
	if strings.Contains(value, code) {
		return code
	}
	return "×"
}

func sellerOrderIcon(orderType string) string {
	if orderType == "Export" {
		return "public"
	}
	return "flag"
}

func sellerOrderCaret(index int) string {
	if index == 0 {
		return "expand_less"
	}
	return "chevron_right"
}

func sellerOrderRowClass(index int) string {
	if index == 0 {
		return "order-row selected"
	}
	return "order-row"
}

func sellerStatIcon(icon string) string {
	if icon == "" {
		return "analytics"
	}
	return icon
}
