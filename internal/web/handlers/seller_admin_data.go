package handlers

import (
	"net/url"
	"strings"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

func sellerAdminPage(path string, values url.Values) viewmodels.SellerAdminPage {
	active := strings.TrimPrefix(path, "/seller-admin/")
	if path == "/seller-admin" || active == "" {
		active = "dashboard"
	}
	requested := active
	if active == "payments" || active == "payouts" {
		active = "earnings"
	}
	if active == "profile" || active == "store-profile" {
		active = "settings"
	}
	if active == "verification" || active == "documents" {
		active = "onboarding"
	}
	if active == "product-list" || active == "add-product" || active == "edit-product" || active == "variants" || active == "media" || active == "pricing" {
		active = "products"
	}
	if active == "inventory-transactions" {
		active = "inventory"
	}
	if active == "order-details" || active == "fulfillment" {
		active = "orders"
	}
	if active == "refund-status" {
		active = "returns"
	}
	if active == "promotions" || active == "reviews" || active == "messages" || active == "support" {
		active = "settings"
	}
	if active == "commission" {
		active = "earnings"
	}
	if active == "audit" {
		active = "analytics"
	}
	if active == "seller-profile" {
		active = "settings"
	}
	if active == "staff" || active == "staff-permissions" {
		active = "team"
	}

	page := viewmodels.SellerAdminPage{
		Active: active, Workspace: requested,
		Query:  values.Get("q"),
		Range:  values.Get("range"),
		Tab:    values.Get("tab"),
		Notice: values.Get("notice"),
	}
	if page.Range == "" {
		page.Range = "Last 7 Days"
	}

	switch active {
	case "onboarding":
		page.Title = "Seller Onboarding"
		page.Subtitle = "Complete your onboarding to start selling on WeeVCrafts."
	case "dashboard":
		page.Title = "Dashboard"
		page.Subtitle = "Welcome back! Here's what's happening with your store today."
		page.Stats = sellerDashboardStats()
	case "products":
		page.Title = "Products"
		page.Subtitle = "Manage your products, keep your catalog fresh, and bring handcrafted stories to customers everywhere."
		page.Stats = []viewmodels.SellerAdminStat{
			{Value: "12", Label: "All products"}, {Value: "2", Label: "Drafts"}, {Value: "2", Label: "Under review"}, {Value: "5", Label: "Approved"}, {Value: "1", Label: "Changes required"}, {Value: "1", Label: "Rejected"}, {Value: "1", Label: "Out of stock"},
		}
		page.Products = sellerProducts()
	case "inventory":
		page.Title = "Inventory"
		page.Subtitle = "Manage your stock, track availability and never miss a sale."
		page.Stats = []viewmodels.SellerAdminStat{
			{Icon: "inventory_2", Value: "48", Label: "Total SKUs", Period: "Across all products", Tone: "red"},
			{Icon: "inventory_2", Value: "36", Label: "In Stock", Period: "75% of inventory", Tone: "green"},
			{Icon: "warning", Value: "8", Label: "Low Stock", Period: "Needs attention", Tone: "gold"},
			{Icon: "cancel", Value: "4", Label: "Out of Stock", Period: "Restock soon", Tone: "red"},
			{Icon: "bookmark", Value: "6", Label: "Reserved", Period: "In active orders", Tone: "purple"},
			{Icon: "schedule", Value: "3", Label: "Pending Restock", Period: "On the way", Tone: "blue"},
		}
		page.Inventory = sellerInventory()
	case "orders":
		page.Title = "Orders & Shipping"
		page.Subtitle = "Manage, fulfill and track your orders with ease."
		page.Stats = []viewmodels.SellerAdminStat{
			{Value: "24", Label: "All Orders"}, {Value: "4", Label: "New"}, {Value: "6", Label: "Processing"}, {Value: "5", Label: "Ready to Ship"}, {Value: "7", Label: "Shipped"}, {Value: "2", Label: "Delivered"}, {Value: "0", Label: "RTO"},
		}
		page.Orders = sellerOrders()
	case "returns":
		page.Title = "Returns"
		page.Subtitle = "Manage return requests from your customers. Review, respond and track progress. The final decision on all return requests is made by WeeVCrafts."
		page.Stats = []viewmodels.SellerAdminStat{
			{Value: "2", Label: "New Requests"}, {Value: "1", Label: "Awaiting Response"}, {Value: "3", Label: "Under Review"}, {Value: "5", Label: "Approved"}, {Value: "1", Label: "Rejected"}, {Value: "8", Label: "Completed"},
		}
		page.Returns = sellerReturns()
	case "earnings":
		page.Title = "Earnings & Settlements"
		page.Subtitle = "Track your sales, commissions, and payouts. Transparent. Simple. Secure."
		page.Stats = []viewmodels.SellerAdminStat{
			{Icon: "shopping_cart", Value: "INR 1,24,750", Label: "Gross Sales", Period: "From 32 orders", Tone: "red"},
			{Icon: "percent", Value: "INR 6,238", Label: "Commission (5%)", Period: "Deducted by WeeVCrafts", Tone: "red"},
			{Icon: "local_shipping", Value: "INR 2,450", Label: "Shipping Adjustments", Period: "Charges / reimbursements", Tone: "red"},
			{Icon: "schedule", Value: "INR 18,990", Label: "Pending Delivery Value", Period: "In transit orders", Tone: "gold"},
			{Icon: "account_balance_wallet", Value: "INR 28,340", Label: "Settlement Eligible", Period: "Ready for next payout", Tone: "green"},
			{Icon: "check_circle", Value: "INR 68,732", Label: "Settled Payouts", Period: "Total paid to date", Tone: "green"},
		}
		page.Ledger = sellerLedger()
		page.Settlements = sellerSettlements()
	case "analytics":
		page.Title = "Analytics"
		page.Subtitle = "Insights to help your handmade business grow."
		page.Stats = []viewmodels.SellerAdminStat{
			{Icon: "currency_rupee", Value: "INR 1,62,430", Label: "Total Revenue", Delta: "28%", Positive: true, Period: "vs previous 6 months"},
			{Icon: "inventory_2", Value: "52", Label: "Orders Placed", Delta: "18%", Positive: true, Period: "vs previous 6 months"},
			{Icon: "category", Value: "74", Label: "Units Sold", Delta: "32%", Positive: true, Period: "vs previous 6 months"},
			{Icon: "shopping_bag", Value: "INR 3,124", Label: "Average Order Value", Delta: "8%", Positive: true, Period: "vs previous 6 months"},
			{Icon: "groups", Value: "18", Label: "Repeat Customers", Delta: "50%", Positive: true, Period: "vs previous 6 months"},
			{Icon: "cancel", Value: "3.8%", Label: "Cancellation Rate", Delta: "36%", Period: "vs previous 6 months"},
		}
		page.AnalyticsBars = []viewmodels.SellerAdminBar{
			{Label: "Nov", Current: "38", Previous: "24"}, {Label: "Dec", Current: "52", Previous: "43"}, {Label: "Jan", Current: "76", Previous: "58"}, {Label: "Feb", Current: "65", Previous: "61"}, {Label: "Mar", Current: "84", Previous: "70"}, {Label: "Apr", Current: "100", Previous: "81"},
		}
	case "settings":
		page.Title = "Store & Account"
		page.Subtitle = "Manage your storefront, business details and account settings — all in one place."
	case "team":
		page.Title = "Team & Permissions"
		page.Subtitle = "Invite trusted staff and give each person only the access they need to help run your store."
		page.Stats = []viewmodels.SellerAdminStat{
			{Icon: "groups", Value: "4", Label: "Team members", Period: "Seller workspace"},
			{Icon: "verified_user", Value: "3", Label: "Active members", Period: "Access reviewed"},
			{Icon: "schedule", Value: "1", Label: "Pending invite", Period: "Awaiting acceptance", Tone: "gold"},
		}
		page.Team = []viewmodels.SellerTeamMember{
			{Name: "Priya Sharma", Email: "priya@weaverstouch.in", Role: "Store owner", Status: "Active", LastActive: "Now", Permissions: "Everything", Image: "/assets/images/customer/maker-mithila.webp"},
			{Name: "Rohan Mehta", Email: "rohan@weaverstouch.in", Role: "Catalog manager", Status: "Active", LastActive: "12 min ago", Permissions: "Products, media, inventory", Image: "/assets/images/customer/saree-blue.webp"},
			{Name: "Aditi Rao", Email: "aditi@weaverstouch.in", Role: "Order assistant", Status: "Active", LastActive: "1 hour ago", Permissions: "Orders, fulfilment, returns", Image: "/assets/images/customer/ceramic-mugs.webp"},
			{Name: "Neha Kapoor", Email: "neha@weaverstouch.in", Role: "Finance viewer", Status: "Invite pending", LastActive: "Not active yet", Permissions: "Read-only earnings", Image: "/assets/images/customer/wooden-box.webp"},
		}
	default:
		page.Active = "dashboard"
		page.Title = "Dashboard"
		page.Subtitle = "Welcome back! Here's what's happening with your store today."
		page.Stats = sellerDashboardStats()
	}
	if requested == active {
		page.Workspace = ""
	} else {
		switch requested {
		case "verification":
			page.Title = "Verification & Documents"
			page.Subtitle = "Review identity, business and bank-document readiness before marketplace approval."
		case "documents":
			page.Title = "Verification Documents"
			page.Subtitle = "Keep seller verification evidence complete, current and access-controlled."
		case "product-list":
			page.Title = "Product List"
			page.Subtitle = "Review catalogue status, availability and marketplace approval state."
		case "add-product":
			page.Title = "Add Product"
			page.Subtitle = "Create a complete product record with variants, media, pricing and compliance details."
		case "edit-product":
			page.Title = "Edit Product"
			page.Subtitle = "Update product content while preserving review and audit context."
		case "variants":
			page.Title = "Product Variants"
			page.Subtitle = "Manage variant-level price, SKU and stock information."
		case "media":
			page.Title = "Product Media"
			page.Subtitle = "Review verified media references and the product gallery order."
		case "pricing":
			page.Title = "Product Pricing"
			page.Subtitle = "Review listed price, comparison price and seller-funded promotion context."
		case "inventory-transactions":
			page.Title = "Inventory Transactions"
			page.Subtitle = "Trace stock adjustments, reservations and restock events by SKU."
		case "order-details":
			page.Title = "Order Details"
			page.Subtitle = "Review customer, item, payment and fulfilment context before acting."
		case "fulfillment":
			page.Title = "Fulfilment"
			page.Subtitle = "Prepare pickup, shipping and delivery updates for seller orders."
		case "refund-status":
			page.Title = "Refund Status"
			page.Subtitle = "Track return decisions and refund progress without changing marketplace payment state."
		case "promotions":
			page.Title = "Promotions & Offers"
			page.Subtitle = "Create seller-funded offers with clear dates, eligibility and performance context."
		case "commission":
			page.Title = "Commission Reports"
			page.Subtitle = "Understand marketplace commission, shipping adjustments and the amount eligible for settlement."
		case "audit":
			page.Title = "Activity & Audit"
			page.Subtitle = "Review catalogue, fulfilment and payout events for your seller account."
		case "reviews":
			page.Title = "Reviews"
			page.Subtitle = "Read customer feedback and follow up on product quality signals."
		case "messages":
			page.Title = "Messages"
			page.Subtitle = "Keep customer conversations and order context together."
		case "support":
			page.Title = "Help & Support"
			page.Subtitle = "Open a support request and track seller operations questions."
		case "seller-profile":
			page.Title = "Seller Profile"
			page.Subtitle = "Manage public maker information, business details and storefront identity."
		case "staff":
			page.Title = "Staff"
			page.Subtitle = "Review the people who help operate this seller workspace."
		case "staff-permissions":
			page.Title = "Staff Permissions"
			page.Subtitle = "Assign only the seller-scoped permissions each team member needs."
		}
	}
	return filterSellerPreview(page, values)
}

func filterSellerPreview(page viewmodels.SellerAdminPage, values url.Values) viewmodels.SellerAdminPage {
	query := strings.ToLower(strings.TrimSpace(values.Get("q")))
	if query != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.SellerAdminProduct) bool {
			return containsAny(query, item.Name, item.SKU, item.Category, item.Status)
		})
		page.Inventory = filterBy(page.Inventory, func(item viewmodels.SellerAdminInventory) bool {
			return containsAny(query, item.Name, item.SKU, item.Variant, item.Location, item.Status)
		})
		page.Orders = filterBy(page.Orders, func(item viewmodels.SellerAdminOrder) bool {
			return containsAny(query, item.Number, item.Customer, item.Items, item.Type, item.Payment, item.Status, item.Destination)
		})
		page.Returns = filterBy(page.Returns, func(item viewmodels.SellerAdminReturn) bool {
			return containsAny(query, item.Number, item.Order, item.Customer, item.Product, item.Reason, item.Status)
		})
		page.Ledger = filterBy(page.Ledger, func(item viewmodels.SellerAdminLedger) bool { return containsAny(query, item.Number, item.Status) })
	}
	if category := strings.ToLower(strings.TrimSpace(values.Get("category"))); category != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.SellerAdminProduct) bool {
			return strings.Contains(strings.ToLower(item.Category), strings.ReplaceAll(category, "-", " "))
		})
	}
	if status := strings.ToLower(strings.TrimSpace(values.Get("status"))); status != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.SellerAdminProduct) bool {
			return strings.Contains(strings.ToLower(item.Status), status)
		})
		page.Inventory = filterBy(page.Inventory, func(item viewmodels.SellerAdminInventory) bool {
			return status == "low-stock" && strings.Contains(strings.ToLower(item.Status), "low") || status != "low-stock" && strings.Contains(strings.ToLower(item.Status), status)
		})
	}
	if availability := strings.ToLower(strings.TrimSpace(values.Get("availability"))); availability != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.SellerAdminProduct) bool {
			code := "D"
			if availability == "export" {
				code = "E"
			}
			return strings.Contains(item.Availability, code)
		})
	}
	if location := strings.ToLower(strings.TrimSpace(values.Get("location"))); location != "" {
		page.Inventory = filterBy(page.Inventory, func(item viewmodels.SellerAdminInventory) bool {
			return strings.Contains(strings.ToLower(item.Location), location)
		})
	}
	if orderType := strings.ToLower(strings.TrimSpace(values.Get("type"))); orderType != "" {
		page.Orders = filterBy(page.Orders, func(item viewmodels.SellerAdminOrder) bool { return strings.EqualFold(item.Type, orderType) })
	}
	if payment := strings.ToLower(strings.TrimSpace(values.Get("payment"))); payment != "" {
		page.Orders = filterBy(page.Orders, func(item viewmodels.SellerAdminOrder) bool { return strings.EqualFold(item.Payment, payment) })
	}
	if tab := strings.ToLower(strings.TrimSpace(values.Get("tab"))); tab != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.SellerAdminProduct) bool { return sellerTabMatches(tab, item.Status, item.Stock) })
		page.Orders = filterBy(page.Orders, func(item viewmodels.SellerAdminOrder) bool { return sellerTabMatches(tab, item.Status, "") })
		page.Returns = filterBy(page.Returns, func(item viewmodels.SellerAdminReturn) bool { return sellerTabMatches(tab, item.Status, "") })
	}
	return page
}

func sellerTabMatches(tab, status, stock string) bool {
	tab = strings.ReplaceAll(strings.ToLower(tab), "_", " ")
	status = strings.ToLower(status)
	switch tab {
	case "draft":
		return strings.Contains(status, "draft")
	case "review", "approval":
		return strings.Contains(status, "review") || strings.Contains(status, "awaiting")
	case "approved":
		return strings.Contains(status, "approved")
	case "changes":
		return strings.Contains(status, "change")
	case "rejected":
		return strings.Contains(status, "reject")
	case "stock", "low":
		return strings.Contains(strings.ToLower(stock), "out") || strings.Contains(status, "stock") || strings.Contains(status, "low")
	case "new":
		return strings.Contains(status, "new")
	case "processing":
		return strings.Contains(status, "processing")
	case "ready":
		return strings.Contains(status, "ready")
	case "shipped":
		return strings.Contains(status, "shipped")
	case "delivered", "completed":
		return strings.Contains(status, "delivered") || strings.Contains(status, "completed")
	case "rto":
		return strings.Contains(status, "rto")
	case "awaiting":
		return strings.Contains(status, "awaiting")
	default:
		return true
	}
}

func sellerDashboardStats() []viewmodels.SellerAdminStat {
	return []viewmodels.SellerAdminStat{
		{Icon: "shopping_cart", Value: "INR 14,797", Label: "Today's Sales", Delta: "12%", Positive: true, Period: "vs. yesterday", Tone: "red"},
		{Icon: "receipt_long", Value: "3", Label: "Orders Today", Delta: "50%", Positive: true, Period: "vs. yesterday", Tone: "red"},
		{Icon: "inventory_2", Value: "5", Label: "Products Sold", Delta: "25%", Positive: true, Period: "vs. yesterday", Tone: "red"},
		{Icon: "local_shipping", Value: "4", Label: "Pending Shipments", Period: "View orders", Tone: "red"},
		{Icon: "description", Value: "2", Label: "Pending Approvals", Period: "View details", Tone: "red"},
		{Icon: "warning", Value: "3", Label: "Low Stock Alerts", Period: "View inventory", Tone: "gold"},
		{Icon: "history", Value: "1", Label: "Return Requests", Period: "View requests", Tone: "red"},
		{Icon: "account_balance_wallet", Value: "INR 1,02,450", Label: "Available Settlement", Period: "View payments", Tone: "green"},
		{Icon: "flag", Value: "12", Label: "Domestic Orders", Delta: "33%", Positive: true, Period: "vs. last week", Tone: "gold"},
		{Icon: "public", Value: "3", Label: "Export Orders", Delta: "50%", Positive: true, Period: "vs. last week", Tone: "green"},
	}
}

func sellerProducts() []viewmodels.SellerAdminProduct {
	return []viewmodels.SellerAdminProduct{
		{Name: "Chanderi Silk Cotton Saree - Royal Maroon", SKU: "WT-CSC-001", Category: "Sarees", Price: "INR 5,999", Variants: "6 colors", Stock: "12", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/saree-maroon.webp"},
		{Name: "Maheshwari Dupatta - Indigo Waves", SKU: "WT-MD-002", Category: "Dupattas", Price: "INR 2,499", Variants: "4 colors", Stock: "25", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/saree-blue.webp"},
		{Name: "Block Print Cushion Cover - Marigold", SKU: "WT-BPC-003", Category: "Home Decor", Price: "INR 899", Variants: "3 designs", Stock: "48", Status: "Under Review", Availability: "D E", Image: "/assets/images/customer/black-cushion.webp"},
		{Name: "Handwoven Tote Bag", SKU: "WT-HTB-004", Category: "Bags & Accessories", Price: "INR 1,899", Variants: "2 variants", Stock: "0", Status: "Out of Stock", Availability: "D E", Image: "/assets/images/customer/blue-tote.webp"},
		{Name: "Maheshwari Silk Saree - Forest Green", SKU: "WT-MSS-005", Category: "Sarees", Price: "INR 6,499", Variants: "4 colors", Stock: "8", Status: "Changes Required", Availability: "D", Image: "/assets/images/customer/saree-blue.webp"},
		{Name: "Block Print Table Runner - Vine Motif", SKU: "WT-BPR-006", Category: "Home Decor", Price: "INR 1,299", Variants: "2 variants", Stock: "15", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/black-cushion.webp"},
		{Name: "Chanderi Dupatta - Blush Pink", SKU: "WT-CD-007", Category: "Dupattas", Price: "INR 2,299", Variants: "3 colors", Stock: "6", Status: "Rejected", Availability: "D E", Image: "/assets/images/customer/saree-maroon.webp"},
		{Name: "Madhubani Wall Art - Tree of Life", SKU: "WT-MWA-008", Category: "Wall Decor", Price: "INR 2,999", Variants: "1 design", Stock: "10", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/madhubani-tree.webp"},
	}
}

func sellerInventory() []viewmodels.SellerAdminInventory {
	return []viewmodels.SellerAdminInventory{
		{Name: "Chanderi Silk Cotton Saree", SKU: "CWC-SAREE-001", Variant: "Royal Maroon", Location: "Bengaluru", Available: "2", Reserved: "1", Reorder: "5", Status: "Low Stock", Image: "/assets/images/customer/saree-maroon.webp"},
		{Name: "Chanderi Silk Cotton Saree", SKU: "CWC-SAREE-002", Variant: "Peacock Blue", Location: "Bengaluru", Available: "14", Reserved: "0", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/saree-blue.webp"},
		{Name: "Tussar Silk Dupatta", SKU: "CWC-DUP-001", Variant: "Mustard", Location: "Bengaluru", Available: "1", Reserved: "0", Reorder: "5", Status: "Low Stock", Image: "/assets/images/customer/saree-maroon.webp"},
		{Name: "Tussar Silk Dupatta", SKU: "CWC-DUP-002", Variant: "Olive Green", Location: "Bengaluru", Available: "12", Reserved: "2", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/saree-blue.webp"},
		{Name: "Block Print Cotton Saree", SKU: "CWC-SAREE-003", Variant: "Indigo", Location: "Bengaluru", Available: "0", Reserved: "1", Reorder: "5", Status: "Out of Stock", Image: "/assets/images/customer/saree-blue.webp"},
		{Name: "Handpainted Ceramic Mugs", SKU: "CWC-MUG-001", Variant: "Set of 2", Location: "Bengaluru", Available: "3", Reserved: "2", Reorder: "5", Status: "Low Stock", Image: "/assets/images/customer/ceramic-mugs.webp"},
		{Name: "Carved Wooden Jewellery Box", SKU: "CWC-BOX-001", Variant: "Natural Brown", Location: "Bengaluru", Available: "8", Reserved: "0", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/wooden-box.webp"},
		{Name: "Madhubani Wall Art", SKU: "CWC-ART-001", Variant: "Tree of Life", Location: "Bengaluru", Available: "5", Reserved: "1", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/madhubani-tree.webp"},
		{Name: "Ikat Cotton Saree", SKU: "CWC-SAREE-004", Variant: "Rust Orange", Location: "Bengaluru", Available: "0", Reserved: "0", Reorder: "5", Status: "Out of Stock", Image: "/assets/images/customer/saree-maroon.webp"},
		{Name: "Linen Stole", SKU: "CWC-STOLE-001", Variant: "Beige", Location: "Bengaluru", Available: "11", Reserved: "0", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/saree-blue.webp"},
	}
}

func sellerOrders() []viewmodels.SellerAdminOrder {
	return []viewmodels.SellerAdminOrder{
		{Number: "#WC2504267831", Date: "26 Apr 2024, 10:24 AM", Customer: "Priya Sharma", Items: "2 items", Value: "INR 5,998", Type: "Domestic", Payment: "Paid", Status: "New", Destination: "Bengaluru, KA", Image: "/assets/images/customer/saree-maroon.webp"},
		{Number: "#WC2504267790", Date: "25 Apr 2024, 04:15 PM", Customer: "Rahul Mehta", Items: "1 item", Value: "INR 2,499", Type: "Domestic", Payment: "Paid", Status: "Processing", Destination: "Mumbai, MH", Image: "/assets/images/customer/saree-blue.webp"},
		{Number: "#WC2504253312", Date: "24 Apr 2024, 11:30 AM", Customer: "Ananya Nair", Items: "3 items", Value: "INR 8,298", Type: "Domestic", Payment: "Paid", Status: "Ready to Ship", Destination: "Chennai, TN", Image: "/assets/images/customer/ceramic-mugs.webp"},
		{Number: "#WC2504189921", Date: "18 Apr 2024, 03:42 PM", Customer: "Sophia Lee", Items: "2 items", Value: "INR 12,499", Type: "Export", Payment: "Paid", Status: "Shipped", Destination: "New York, USA", Image: "/assets/images/customer/madhubani-tree.webp"},
		{Number: "#WC250415520", Date: "01 Apr 2024, 09:45 AM", Customer: "Meera Krishnan", Items: "4 items", Value: "INR 6,247", Type: "Domestic", Payment: "Refunded", Status: "Delivered", Destination: "Bengaluru, KA", Image: "/assets/images/customer/wooden-box.webp"},
	}
}

func sellerReturns() []viewmodels.SellerAdminReturn {
	return []viewmodels.SellerAdminReturn{
		{Number: "#RTN250427001", Order: "#WC2504267819", Customer: "Priya Sharma", Product: "Chanderi Silk Cotton Saree - Royal Maroon", Reason: "Received a different item", Date: "28 Apr 2024, 10:24 AM", Status: "New Request", Amount: "INR 5,999", Image: "/assets/images/customer/saree-maroon.webp"},
		{Number: "#RTN2504269982", Order: "#WC2504267990", Customer: "Rahul Mehta", Product: "Handpainted Ceramic Mugs (Set of 2)", Reason: "Damaged item", Date: "26 Apr 2024", Status: "Awaiting Response", Amount: "INR 1,299", Image: "/assets/images/customer/ceramic-mugs.webp"},
		{Number: "#RTN2504245561", Order: "#WC2504245331", Customer: "Sneha Iyer", Product: "Madhubani Painting - Tree of Life", Reason: "Not as described", Date: "24 Apr 2024", Status: "Under Review", Amount: "INR 2,499", Image: "/assets/images/customer/madhubani-tree.webp"},
		{Number: "#RTN2504203321", Order: "#WC2504201172", Customer: "Arjun Nair", Product: "Carved Wooden Jewellery Box", Reason: "Item damaged", Date: "20 Apr 2024", Status: "Approved", Amount: "INR 1,299", Image: "/assets/images/customer/wooden-box.webp"},
		{Number: "#RTN2504187712", Order: "#WC2504189921", Customer: "Kavita Rao", Product: "Handpainted Ceramic Mugs (Set of 2)", Reason: "Change of mind", Date: "18 Apr 2024", Status: "Rejected", Amount: "INR 2,598", Image: "/assets/images/customer/ceramic-mugs.webp"},
		{Number: "#RTN2504156677", Order: "#WC250415520", Customer: "Vikram Desai", Product: "Chanderi Silk Cotton Saree - Royal Maroon", Reason: "Size issue", Date: "15 Apr 2024", Status: "Completed", Amount: "INR 5,999", Image: "/assets/images/customer/saree-maroon.webp"},
	}
}

func sellerLedger() []viewmodels.SellerAdminLedger {
	return []viewmodels.SellerAdminLedger{
		{Number: "#WC2504267819", Date: "26 Apr 2024", Sale: "INR 7,599", Commission: "INR 380", Adjustments: "+ INR 99", Status: "Pending", Net: "INR 7,318"},
		{Number: "#WC2504216632", Date: "21 Apr 2024", Sale: "INR 2,499", Commission: "INR 125", Adjustments: "INR 0", Status: "Delivered", Net: "INR 2,374"},
		{Number: "#WC2504149981", Date: "14 Apr 2024", Sale: "INR 8,298", Commission: "INR 415", Adjustments: "- INR 99", Status: "Processing", Net: "INR 7,784"},
		{Number: "#WC2504097712", Date: "09 Apr 2024", Sale: "INR 1,899", Commission: "INR 95", Adjustments: "INR 0", Status: "Cancelled", Net: "INR 0"},
		{Number: "#WC2504015520", Date: "01 Apr 2024", Sale: "INR 6,247", Commission: "INR 312", Adjustments: "+ INR 49", Status: "Settled", Net: "INR 5,984"},
		{Number: "#WC2503294431", Date: "29 Mar 2024", Sale: "INR 3,299", Commission: "INR 165", Adjustments: "INR 0", Status: "Settled", Net: "INR 3,134"},
		{Number: "#WC2503156621", Date: "15 Mar 2024", Sale: "INR 4,999", Commission: "INR 250", Adjustments: "- INR 99", Status: "Settled", Net: "INR 4,650"},
		{Number: "#WC2503087710", Date: "08 Mar 2024", Sale: "INR 2,199", Commission: "INR 110", Adjustments: "INR 0", Status: "Settled", Net: "INR 2,089"},
		{Number: "#WC2503012284", Date: "01 Mar 2024", Sale: "INR 5,749", Commission: "INR 287", Adjustments: "+ INR 99", Status: "Settled", Net: "INR 5,961"},
		{Number: "#WC2502221180", Date: "22 Feb 2024", Sale: "INR 3,499", Commission: "INR 175", Adjustments: "INR 0", Status: "Settled", Net: "INR 3,324"},
	}
}

func sellerSettlements() []viewmodels.SellerAdminSettlement {
	return []viewmodels.SellerAdminSettlement{
		{Number: "#STL24043001", Amount: "INR 28,340", Orders: "32 orders", Status: "Processing", Date: "Expected by 05 May 2024"},
		{Number: "#STL24041502", Amount: "INR 18,960", Orders: "21 orders", Status: "Paid", Date: "Credited on 17 Apr 2024"},
		{Number: "#STL24033101", Amount: "INR 16,450", Orders: "18 orders", Status: "Paid", Date: "Credited on 02 Apr 2024"},
		{Number: "#STL24031501", Amount: "INR 12,890", Orders: "12 orders", Status: "Paid", Date: "Credited on 18 Mar 2024"},
	}
}
