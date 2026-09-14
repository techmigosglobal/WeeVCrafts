package pages

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

func breadcrumbHref(item string) string {
	switch strings.ToLower(strings.TrimSpace(item)) {
	case "home":
		return "/"
	case "categories":
		return "/categories"
	case "brands":
		return "/brands"
	case "deals":
		return "/deals"
	case "search results":
		return "/products"
	case "sarees":
		return "/products?category=sarees"
	case "arts":
		return "/products?category=arts"
	case "crafts":
		return "/products?category=crafts"
	case "home & living":
		return "/products?category=home-living"
	case "fashion":
		return "/products?category=fashion"
	case "gifts":
		return "/products?category=gifts"
	case "my account":
		return "/account"
	case "account":
		return "/account"
	case "my orders":
		return "/orders"
	case "your cart":
		return "/cart"
	case "my wishlist":
		return "/wishlist"
	case "returns and refunds":
		return "/returns"
	case "checkout":
		return "/checkout"
	case "track order", "track your order":
		return "/tracking"
	case "payments", "payment methods":
		return "/payments"
	case "order details":
		return "/orders/WC2504267819"
	case "chanderi silk cotton saree":
		return "/products/chanderi-royal"
	default:
		return "/products"
	}
}

func breadcrumbCurrent(index, total int) string {
	if index == total-1 {
		return "page"
	}
	return ""
}

func paginationClass(page, current int) string {
	if page == current {
		return "active"
	}
	return ""
}

func paginationCurrent(page, current int) string {
	if page == current {
		return "page"
	}
	return ""
}

func pageLabel(page int) string { return strconv.Itoa(page) }

func orderPath(number string) string {
	return "/orders/" + strings.TrimPrefix(strings.TrimSpace(number), "#")
}

func orderSubset(orders []viewmodels.Order, status string) []viewmodels.Order {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return orders
	}
	filtered := make([]viewmodels.Order, 0, len(orders))
	for _, order := range orders {
		orderStatus := strings.ToLower(strings.TrimSpace(order.Status))
		matches := (status == "active" && orderStatus != "delivered" && orderStatus != "cancelled") ||
			(status == "delivered" && orderStatus == "delivered") ||
			(status == "cancelled" && orderStatus == "cancelled")
		if matches {
			filtered = append(filtered, order)
		}
	}
	return filtered
}

func orderCount(orders []viewmodels.Order, status string) int {
	return len(orderSubset(orders, status))
}

func orderSeller(order viewmodels.Order) string {
	item, ok := firstCartItem(order.Items)
	if !ok || strings.TrimSpace(item.Product.Seller) == "" {
		return "Independent maker"
	}
	return item.Product.Seller
}

func orderDeliveryCopy(order viewmodels.Order) string {
	switch strings.ToLower(strings.TrimSpace(order.Status)) {
	case "delivered":
		return "Delivered"
	case "cancelled":
		return "Order cancelled"
	default:
		return "Delivery status will update as the maker ships"
	}
}

func orderTabClass(filters url.Values, status string) string {
	if strings.EqualFold(filters.Get("status"), status) || (status == "" && filters.Get("status") == "") {
		return "active"
	}
	return ""
}

func orderTabCurrent(filters url.Values, status string) string {
	if orderTabClass(filters, status) == "active" {
		return "page"
	}
	return ""
}

func tabSelected(active bool) string {
	if active {
		return "true"
	}
	return "false"
}

func productSlice(products []viewmodels.Product, start, end int) []viewmodels.Product {
	if start < 0 {
		start = 0
	}
	if start > len(products) {
		start = len(products)
	}
	if end < start {
		end = start
	}
	if end > len(products) {
		end = len(products)
	}
	return products[start:end]
}

func firstCartItem(items []viewmodels.CartItem) (viewmodels.CartItem, bool) {
	if len(items) == 0 {
		return viewmodels.CartItem{}, false
	}
	return items[0], true
}

func firstCartItemValue(items []viewmodels.CartItem) viewmodels.CartItem {
	item, _ := firstCartItem(items)
	return item
}

func orderSlice(orders []viewmodels.Order, start int) []viewmodels.Order {
	if start < 0 {
		start = 0
	}
	if start > len(orders) {
		start = len(orders)
	}
	return orders[start:]
}

func firstOrder(orders []viewmodels.Order) viewmodels.Order {
	if len(orders) == 0 {
		return viewmodels.Order{}
	}
	return orders[0]
}

func orderStatusTitle(order viewmodels.Order) string {
	switch strings.ToLower(strings.TrimSpace(order.Status)) {
	case "delivered":
		return "Your order has arrived safely."
	case "cancelled":
		return "This order was cancelled."
	default:
		return "Your order is being prepared with care."
	}
}

func orderStatusCopy(order viewmodels.Order) string {
	switch strings.ToLower(strings.TrimSpace(order.Status)) {
	case "delivered":
		return "We hope it brings something beautiful into your everyday."
	case "cancelled":
		return "If you need help with this cancellation, our customer care team is here."
	default:
		return "We will notify you as soon as it ships."
	}
}

func firstReturn(returns []viewmodels.Return) viewmodels.Return {
	if len(returns) == 0 {
		return viewmodels.Return{}
	}
	return returns[0]
}

func returnAt(returns []viewmodels.Return, index int) viewmodels.Return {
	if index < 0 || index >= len(returns) {
		return viewmodels.Return{}
	}
	return returns[index]
}

func returnCount(returns []viewmodels.Return, tab string) int {
	count := 0
	for _, item := range returns {
		status := strings.ToLower(strings.TrimSpace(item.Status))
		switch tab {
		case "refunds":
			if strings.Contains(status, "refund") {
				count++
			}
		case "closed":
			if strings.Contains(status, "closed") {
				count++
			}
		default:
			if !strings.Contains(status, "refund") && !strings.Contains(status, "closed") {
				count++
			}
		}
	}
	return count
}

func returnSubset(returns []viewmodels.Return, tab string) []viewmodels.Return {
	tab = strings.ToLower(strings.TrimSpace(tab))
	filtered := make([]viewmodels.Return, 0, len(returns))
	for _, item := range returns {
		status := strings.ToLower(strings.TrimSpace(item.Status))
		matches := false
		switch tab {
		case "refunds":
			matches = strings.Contains(status, "refund")
		case "closed":
			matches = strings.Contains(status, "closed")
		default:
			matches = !strings.Contains(status, "refund") && !strings.Contains(status, "closed")
		}
		if matches {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func returnActionLabel(status string) string {
	if strings.Contains(strings.ToLower(strings.TrimSpace(status)), "refund") {
		return "View refund details"
	}
	return "View request"
}

func returnTabClass(filters url.Values, tab string) string {
	if strings.EqualFold(filters.Get("tab"), tab) || (tab == "" && filters.Get("tab") == "") {
		return "active"
	}
	return ""
}

func returnTabCurrent(filters url.Values, tab string) string {
	if returnTabClass(filters, tab) == "active" {
		return "page"
	}
	return ""
}

func productAt(products []viewmodels.Product, index int) viewmodels.Product {
	if index < 0 || index >= len(products) {
		return viewmodels.Product{}
	}
	return products[index]
}

func brandProducts(products []viewmodels.Product, brand string) []viewmodels.Product {
	result := make([]viewmodels.Product, 0)
	for _, product := range products {
		if strings.EqualFold(product.Seller, brand) {
			result = append(result, product)
		}
	}
	return result
}

func paymentStateLabel(state string) string {
	switch state {
	case "success":
		return "Success"
	case "failed":
		return "Failed"
	default:
		return "Pending"
	}
}

func paymentStateTitle(state string) string {
	switch state {
	case "success":
		return "Payment successful"
	case "failed":
		return "Payment failed"
	default:
		return "Payment pending"
	}
}

func paymentStateDescription(state string) string {
	switch state {
	case "success":
		return "Your payment was captured and matched to the order."
	case "failed":
		return "The payment did not complete. Your order was not charged twice."
	default:
		return "The payment is waiting for confirmation from the payment network."
	}
}

func paymentStateHeading(state string) string {
	switch state {
	case "success":
		return "Payment captured"
	case "failed":
		return "Action needed"
	default:
		return "Reconciliation in progress"
	}
}

func paymentStateCopy(state string) string {
	switch state {
	case "success":
		return "The payment gateway confirmed the transaction and WeeVCrafts has recorded it against your order."
	case "failed":
		return "You can retry with another method or contact support with the reference below. No duplicate order will be created."
	default:
		return "We are checking the gateway callback. If the status does not change, support can reconcile the reference manually."
	}
}

func paymentStateReference(state string) string {
	switch state {
	case "success":
		return "#PAY25603421"
	case "failed":
		return "#PAY25601432"
	default:
		return "#PAY25602912"
	}
}

func paymentStateNextStep(state string) string {
	switch state {
	case "success":
		return "No action required"
	case "failed":
		return "Retry or contact support"
	default:
		return "Wait for gateway confirmation"
	}
}

func paymentStateIcon(state string) string {
	switch state {
	case "success":
		return "check_circle"
	case "failed":
		return "error"
	default:
		return "hourglass_top"
	}
}

func paymentActivityClass(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed":
		return "danger"
	case "pending":
		return "review"
	default:
		return "success"
	}
}

func listingTitle(p viewmodels.CustomerPage) string {
	if p.Query != "" {
		return "Search results for “" + p.Query + "”"
	}
	if p.Category != "" {
		return strings.Title(strings.ReplaceAll(p.Category, "-", " "))
	}
	return "Explore handmade finds"
}

func listingDescription(p viewmodels.CustomerPage) string {
	if p.Query != "" {
		return "Thoughtfully made pieces that match your search, from verified Indian makers."
	}
	if p.Category != "" {
		return "Thoughtful pieces from independent Indian makers, made slowly and meant to last."
	}
	return "Timeless pieces and the stories of the people who make them."
}

func normalizedSort(sort string) string {
	if sort == "" {
		return "relevance"
	}
	return sort
}

func hasFilter(filters url.Values, key, value string) bool {
	for _, candidate := range filters[key] {
		if candidate == value {
			return true
		}
	}
	return false
}

func filterCount(counts map[string]int, key, value string) string {
	if count := counts[key+":"+value]; count > 0 {
		return "(" + strconv.Itoa(count) + ")"
	}
	return ""
}

func listingURL(p viewmodels.CustomerPage, page int) string {
	values := url.Values{}
	for key, entries := range p.Filters {
		if key == "page" {
			continue
		}
		for _, entry := range entries {
			values.Add(key, entry)
		}
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	query := values.Encode()
	if query == "" {
		return "/products"
	}
	return "/products?" + query
}

func makerPath(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "mithila arts":
		return "/makers/mithila-arts"
	case "weaver's touch":
		return "/makers/weavers-touch"
	case "clay & co.":
		return "/makers/clay-and-co"
	case "the rustic home":
		return "/makers/rustic-home"
	default:
		return "/makers/mithila-arts"
	}
}

func productLead(product viewmodels.Product) string {
	switch strings.ToLower(product.Category) {
	case "arts":
		return "A hand-painted story with a place in every room."
	case "crafts":
		return "A considered object shaped slowly by an independent maker."
	case "home & living":
		return "Useful, enduring craft for everyday rituals at home."
	case "fashion":
		return "Made to move with you, with texture and story in every detail."
	case "gifts":
		return "A thoughtful handmade piece for a moment worth remembering."
	default:
		return "A timeless piece made with care and intention."
	}
}

func productDescription(product viewmodels.Product) string {
	switch strings.ToLower(product.Category) {
	case "arts":
		return "This hand-painted artwork brings an enduring Indian folk tradition into contemporary spaces, with every mark carrying the character of its maker."
	case "crafts":
		return "Made in a small batch by an independent Indian artisan, this durable craft piece is designed to be used, kept and passed along."
	case "home & living":
		return "A useful, tactile object made for everyday living. Natural materials and a thoughtful finish keep the maker's hand visible."
	case "fashion":
		return "A practical, beautiful everyday piece woven or printed in small batches, with the material and maker story kept close."
	case "gifts":
		return "A meaningful handmade gift, carefully finished and ready to bring a little more warmth to an everyday celebration."
	default:
		return "A beautiful handmade piece from an independent Indian maker, created slowly and meant to last."
	}
}

func productFabric(product viewmodels.Product) string {
	switch strings.ToLower(product.Category) {
	case "arts":
		return "Hand-painted natural board"
	case "crafts":
		return "Hand-finished metal"
	case "home & living":
		return "Natural clay and earth pigments"
	case "fashion":
		return "Handwoven cotton"
	case "gifts":
		return "Fired terracotta"
	default:
		return "Natural materials"
	}
}

func productColor(product viewmodels.Product) string {
	if strings.Contains(strings.ToLower(product.Name), "maroon") {
		return "Royal maroon"
	}
	if strings.Contains(strings.ToLower(product.Name), "blue") || strings.Contains(strings.ToLower(product.Name), "tote") {
		return "Indigo blue"
	}
	if strings.Contains(strings.ToLower(product.Name), "terracotta") || strings.Contains(strings.ToLower(product.Category), "gifts") {
		return "Terracotta"
	}
	return "Natural earth"
}

func productWeave(product viewmodels.Product) string {
	if strings.EqualFold(product.Category, "Arts") {
		return "Folk painting"
	}
	if strings.EqualFold(product.Category, "Fashion") || strings.Contains(strings.ToLower(product.Name), "saree") {
		return "Handloom"
	}
	return "Hand-finished"
}

func productCare(product viewmodels.Product) string {
	if strings.EqualFold(product.Category, "Fashion") || strings.Contains(strings.ToLower(product.Name), "saree") {
		return "Gentle dry clean"
	}
	return "Dust with a soft cloth"
}

func productStory(product viewmodels.Product) string {
	return product.Seller + " works with independent artisans in " + product.Location + ", keeping local material knowledge and patient making at the centre of every piece."
}

func productStockMessage(product viewmodels.Product) string {
	if product.InStock {
		return "In stock"
	}
	return "Currently unavailable"
}
