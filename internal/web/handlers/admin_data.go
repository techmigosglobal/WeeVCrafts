package handlers

import (
	"net/url"
	"strings"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

func adminPage(path string, values url.Values) viewmodels.AdminPage {
	active := strings.TrimPrefix(path, "/admin/")
	if path == "/admin" || active == "" {
		active = "dashboard"
	}
	requested := active
	switch active {
	case "brands":
		active = "products"
	case "customers", "reviews":
		active = "support"
	case "payments", "failed-payments":
		active = "finance"
	case "disputes", "refunds":
		active = "returns"
	case "roles", "security-events":
		active = "security"
	case "seller-approvals", "seller-details", "seller-suspensions":
		active = "sellers"
	case "product-details", "category-management":
		active = "products"
	case "coupons":
		active = "marketing"
	case "commission-rules", "settlements", "finance-reconciliation":
		active = "finance"
	case "audit-logs", "system-health", "search-indexing-health", "worker-queue-health", "settings":
		active = "analytics"
	}
	page := viewmodels.AdminPage{
		Active: active, Workspace: requested,
		Query:  values.Get("q"),
		Range:  values.Get("range"),
		Notice: values.Get("notice"),
	}
	if page.Notice == "" && requested != active {
		page.Notice = strings.ReplaceAll(requested, "-", " ") + " workspace opened"
	}
	if page.Range == "" {
		page.Range = "Last 30 Days"
	}

	switch active {
	case "sellers":
		page.Title = "Sellers"
		page.Subtitle = "Manage seller applications, verify artisans, monitor performance and help them grow with WeeVCrafts."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "groups", Value: "326", Label: "Total Sellers", Delta: "8%", Positive: true, Period: "vs. last month"},
			{Icon: "assignment", Value: "28", Label: "Pending Applications", Delta: "25%", Period: "vs. last month"},
			{Icon: "verified", Value: "214", Label: "Active Sellers", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "pause_circle", Value: "18", Label: "Suspended Sellers", Delta: "38%", Period: "vs. last month"},
			{Icon: "public", Value: "214", Label: "Export Approved", Delta: "22%", Positive: true, Period: "vs. last month"},
		}
		page.Sellers = []viewmodels.AdminSeller{
			{Name: "Heritage Looms", Location: "Varanasi, UP", Category: "Sarees", Applied: "26 Apr 2024", Status: "Submitted", Image: "/assets/images/mock/saree-maroon.webp"},
			{Name: "Desert Craft Co.", Location: "Jodhpur, RJ", Category: "Home Decor", Applied: "25 Apr 2024", Status: "Pending KYC", Image: "/assets/images/mock/box.webp"},
			{Name: "Kashmir Creations", Location: "Srinagar, J&K", Category: "Handicrafts", Applied: "24 Apr 2024", Status: "Under Review", Image: "/assets/images/mock/cushion.webp"},
			{Name: "Madhubani Magic", Location: "Madhubani, BR", Category: "Paintings", Applied: "23 Apr 2024", Status: "Submitted", Image: "/assets/images/mock/painting.webp"},
			{Name: "Coastal Clay Studio", Location: "Kochi, KL", Category: "Pottery", Applied: "22 Apr 2024", Status: "Pending KYC", Image: "/assets/images/mock/mugs.webp"},
		}
	case "products":
		page.Title = "Products & Categories"
		page.Subtitle = "Manage the WeeVCrafts catalog. Approve products, organize categories, define attributes, manage claims and ensure compliance."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "inventory_2", Value: "5,842", Label: "Total Products", Delta: "14%", Positive: true, Period: "vs. last month"},
			{Icon: "schedule", Value: "326", Label: "Pending Approvals", Delta: "8%", Period: "vs. last month"},
			{Icon: "check_circle", Value: "5,214", Label: "Approved Products", Delta: "16%", Positive: true, Period: "vs. last month"},
			{Icon: "description", Value: "402", Label: "Drafts", Delta: "5%", Positive: true, Period: "vs. last month"},
			{Icon: "verified_user", Value: "128", Label: "Flagged (Compliance)", Delta: "12%", Period: "vs. last month"},
		}
		page.Products = []viewmodels.AdminProduct{
			{Name: "Chanderi Silk Saree", SKU: "WC#2504267813", Seller: "Weaves of MP", Category: "Sarees > Chanderi", Price: "INR 2,499", Submitted: "26 Apr 2024", Status: "Pending", Image: "/assets/images/mock/saree-maroon.webp"},
			{Name: "Brass Elephant Figurine", SKU: "WC#2504267120", Seller: "The Rustic Home", Category: "Handicrafts > Metal Crafts", Price: "INR 2,849", Submitted: "25 Apr 2024", Status: "Pending", Image: "/assets/images/mock/elephant.webp"},
			{Name: "Madhubani Painting", SKU: "WC#2504266981", Seller: "Madhubani Magic", Category: "Paintings > Madhubani", Price: "INR 6,247", Submitted: "24 Apr 2024", Status: "Under Review", Image: "/assets/images/mock/painting.webp"},
			{Name: "Ceramic Mug - Blue Floral", SKU: "WC#2504266552", Seller: "Clay & Co.", Category: "Home Decor > Ceramic", Price: "INR 799", Submitted: "23 Apr 2024", Status: "Pending", Image: "/assets/images/mock/mugs.webp"},
			{Name: "Wooden Jewellery Box", SKU: "WC#2504266311", Seller: "Artisans of RJ", Category: "Home Decor > Wooden Crafts", Price: "INR 2,199", Submitted: "22 Apr 2024", Status: "Pending", Image: "/assets/images/mock/box.webp"},
		}
	case "inventory":
		page.Title = "Own Inventory"
		page.Subtitle = "Products owned or directly sold by WeeVCrafts. Manage stock, warehouses, and inventory operations."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "category", Value: "INR 1,26,48,320", Label: "Total Inventory Value (GMV)", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "inventory_2", Value: "1,248", Label: "Total Units in Stock", Delta: "8%", Positive: true, Period: "vs. last month"},
			{Icon: "shopping_cart", Value: "86", Label: "Own Products (SKUs)", Delta: "10%", Positive: true, Period: "vs. last month"},
			{Icon: "warehouse", Value: "3", Label: "Warehouses / Locations", Delta: "0%", Period: "vs. last month"},
		}
		page.Inventory = []viewmodels.AdminInventory{
			{Name: "Kanchipuram Silk Saree", SKU: "WC-SAR-001", Stock: "3", Reorder: "20", Reserved: "0", Available: "3", Location: "Noida", Value: "INR 44,970", Status: "Low Stock", Image: "/assets/images/mock/saree-maroon.webp"},
			{Name: "Tussar Silk Saree", SKU: "WC-SAR-002", Stock: "28", Reorder: "20", Reserved: "5", Available: "23", Location: "Varanasi", Value: "INR 3,35,720", Status: "In Stock", Image: "/assets/images/mock/saree-blue.webp"},
			{Name: "Blue Pottery Ceramic Mug", SKU: "WC-CER-014", Stock: "5", Reorder: "25", Reserved: "0", Available: "5", Location: "Jaipur", Value: "INR 7,450", Status: "Low Stock", Image: "/assets/images/mock/mugs.webp"},
			{Name: "Madhubani Wall Painting", SKU: "WC-PNT-007", Stock: "2", Reorder: "10", Reserved: "1", Available: "1", Location: "Noida", Value: "INR 12,980", Status: "Low Stock", Image: "/assets/images/mock/painting.webp"},
			{Name: "Brass Elephant Figurine", SKU: "WC-DCR-021", Stock: "4", Reorder: "15", Reserved: "2", Available: "2", Location: "Bengaluru", Value: "INR 18,760", Status: "Low Stock", Image: "/assets/images/mock/elephant.webp"},
		}
	case "orders":
		page.Title = "Orders & Shipping"
		page.Subtitle = "Manage marketplace orders, seller suborders, shipments, and deliveries across India and worldwide."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "shopping_cart", Value: "1,842", Label: "Total Marketplace Orders", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "receipt_long", Value: "1,968", Label: "Seller Suborders", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "local_shipping", Value: "1,412", Label: "Orders Shipped", Delta: "18%", Positive: true, Period: "vs. last month"},
			{Icon: "inventory_2", Value: "1,276", Label: "Orders Delivered", Delta: "16%", Positive: true, Period: "vs. last month"},
			{Icon: "flight", Value: "214", Label: "Export Orders", Delta: "50%", Positive: true, Period: "vs. last month"},
			{Icon: "public", Value: "INR 12,36,450", Label: "Export GMV", Delta: "62%", Positive: true, Period: "vs. last month"},
			{Icon: "undo", Value: "56", Label: "RTO Orders", Delta: "28%", Period: "vs. last month"},
			{Icon: "pie_chart", Value: "3.0%", Label: "RTO Rate", Delta: "0.8%", Period: "vs. last month"},
		}
		page.Orders = []viewmodels.AdminOrder{
			{Number: "#WC2504267819", Date: "26 Apr 2024", Customer: "Priya Sharma", Items: "2 items / 2 suborders", Value: "INR 14,797", Type: "Domestic", Destination: "Varanasi, UP", Fulfillment: "Shipped", Delivery: "In Transit", RTO: "-"},
			{Number: "#WC2504267631", Date: "25 Apr 2024", Customer: "Rohit Mehta", Items: "3 items / 3 suborders", Value: "INR 2,499", Type: "Domestic", Destination: "Delhi, Delhi", Fulfillment: "Delivered", Delivery: "Delivered", RTO: "-"},
			{Number: "#WC2504267284", Date: "24 Apr 2024", Customer: "Ananya Iyer", Items: "1 item / 1 suborder", Value: "INR 8,298", Type: "Export", Destination: "New York, USA", Fulfillment: "Shipped", Delivery: "In Transit", RTO: "-"},
			{Number: "#WC2504267120", Date: "23 Apr 2024", Customer: "Suresh Nair", Items: "2 items / 2 suborders", Value: "INR 1,899", Type: "Domestic", Destination: "Kochi, KL", Fulfillment: "Processing", Delivery: "-", RTO: "-"},
			{Number: "#WC2504266981", Date: "22 Apr 2024", Customer: "Meera Krishnan", Items: "1 item / 1 suborder", Value: "INR 6,247", Type: "Domestic", Destination: "Bengaluru, KA", Fulfillment: "Delivered", Delivery: "Delivered", RTO: "-"},
		}
	case "returns":
		page.Title = "Returns & Refunds"
		page.Subtitle = "Manage return requests, refund decisions, replacements and disputes across the WeeVCrafts marketplace."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "undo", Value: "214", Label: "Total Return Requests", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "hourglass_top", Value: "48", Label: "Pending Review", Delta: "8%", Period: "vs. last month"},
			{Icon: "warning", Value: "12", Label: "Disputes", Delta: "50%", Period: "vs. last month"},
			{Icon: "inventory_2", Value: "36", Label: "Replacement Requests", Delta: "24%", Positive: true, Period: "vs. last month"},
			{Icon: "currency_rupee", Value: "INR 2,36,450", Label: "Refunds Processed", Delta: "18%", Positive: true, Period: "vs. last month"},
		}
		page.Returns = []viewmodels.AdminReturn{
			{Number: "#WC2504267819", Customer: "Priya Sharma", Product: "Kashmiri Silk Saree", Reason: "Item arrived damaged", Date: "24 Apr 2024", Status: "Dispute", Amount: "INR 14,797", Image: "/assets/images/mock/saree-maroon.webp"},
			{Number: "#WC2504267631", Customer: "Rohit Mehta", Product: "Handpainted Ceramic Mug", Reason: "Damaged item", Date: "24 Apr 2024", Status: "Pending", Amount: "INR 2,499", Image: "/assets/images/mock/mugs.webp"},
			{Number: "#WC2504267284", Customer: "Ananya Iyer", Product: "Wooden Jewellery Box", Reason: "Changed my mind", Date: "24 Apr 2024", Status: "Replacement", Amount: "INR 8,298", Image: "/assets/images/mock/box.webp"},
			{Number: "#WC2504267120", Customer: "Suresh Nair", Product: "Brass Elephant Figurine", Reason: "Not as described", Date: "23 Apr 2024", Status: "Pending", Amount: "INR 1,899", Image: "/assets/images/mock/elephant.webp"},
			{Number: "#WC2504266981", Customer: "Meera Krishnan", Product: "Madhubani Painting", Reason: "Quality concern", Date: "22 Apr 2024", Status: "Under Review", Amount: "INR 6,247", Image: "/assets/images/mock/painting.webp"},
		}
	case "finance":
		page.Title = "Finance"
		page.Subtitle = "Financial control center for a transparent and growing marketplace."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "paid", Value: "INR 34,56,780", Label: "Total Payments", Delta: "18%", Positive: true, Period: "vs. last month"},
			{Icon: "hourglass_top", Value: "INR 7,86,320", Label: "Pending Settlements", Delta: "12%", Period: "vs. last month"},
			{Icon: "undo", Value: "INR 1,24,560", Label: "Refunded Amount", Delta: "28%", Positive: true, Period: "vs. last month"},
			{Icon: "database", Value: "INR 2,41,835", Label: "Commission Revenue", Delta: "16%", Positive: true, Period: "vs. last month"},
			{Icon: "receipt_long", Value: "1,842", Label: "Export Receipts", Delta: "22%", Positive: true, Period: "vs. last month"},
			{Icon: "payments", Value: "INR 6,12,450", Label: "COD Collections", Delta: "10%", Positive: true, Period: "vs. last month"},
		}
		page.Settlements = []viewmodels.AdminSettlement{
			{Seller: "Weaver's Touch", Orders: "256", Gross: "INR 5,02,340", Commission: "INR 25,117", Amount: "INR 4,77,223", Status: "Pending", Date: "28 Apr 2024", Image: "/assets/images/mock/saree-blue.webp"},
			{Seller: "Heritage Looms", Orders: "198", Gross: "INR 3,84,210", Commission: "INR 19,211", Amount: "INR 3,65,000", Status: "Completed", Date: "25 Apr 2024", Image: "/assets/images/mock/saree-maroon.webp"},
			{Seller: "Desert Craft Co.", Orders: "167", Gross: "INR 2,96,430", Commission: "INR 14,822", Amount: "INR 2,81,608", Status: "Completed", Date: "24 Apr 2024", Image: "/assets/images/mock/box.webp"},
			{Seller: "Kashmir Creations", Orders: "143", Gross: "INR 2,11,540", Commission: "INR 10,577", Amount: "INR 2,00,963", Status: "Processing", Date: "29 Apr 2024", Image: "/assets/images/mock/cushion.webp"},
		}
		page.Transactions = []viewmodels.AdminTransaction{
			{Date: "26 Apr 2024, 10:24 AM", Type: "Payment", Reference: "#PAY25603421", Description: "Payment received from customer", Amount: "INR 2,499", Status: "Success"},
			{Date: "26 Apr 2024, 09:15 AM", Type: "Commission", Reference: "#COM25603421", Description: "5% commission (Order #12345)", Amount: "INR 125", Status: "Success"},
			{Date: "25 Apr 2024, 06:30 PM", Type: "Settlement", Reference: "#SET25602871", Description: "Settlement to Weaver's Touch", Amount: "- INR 3,92,110", Status: "Success"},
			{Date: "24 Apr 2024, 11:12 AM", Type: "Refund", Reference: "#REF25601432", Description: "Refund to customer (Order #12301)", Amount: "- INR 1,499", Status: "Success"},
		}
	case "support":
		page.Title = "Customers & Support"
		page.Subtitle = "Customers are at the heart of WeeVCrafts. Monitor satisfaction, resolve issues, and build lasting relationships."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "groups", Value: "1,24,860", Label: "Total Customers", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "person", Value: "28,450", Label: "Active Customers", Delta: "14%", Positive: true, Period: "vs. last month"},
			{Icon: "headset_mic", Value: "2,184", Label: "Support Cases", Delta: "8%", Period: "vs. last month"},
			{Icon: "description", Value: "214", Label: "Open Cases", Delta: "22%", Period: "vs. last month"},
			{Icon: "sentiment_satisfied", Value: "96%", Label: "Customer Satisfaction", Delta: "3%", Positive: true, Period: "vs. last month"},
		}
		page.SupportCases = []viewmodels.AdminSupportCase{
			{Case: "#WCS30427621", Customer: "Priya Sharma", Subject: "Order not delivered", Priority: "High", Date: "26 Apr 2024", Status: "Open"},
			{Case: "#WCS30427510", Customer: "Neha Kapoor", Subject: "Damaged item", Priority: "High", Date: "25 Apr 2024", Status: "Open"},
			{Case: "#WCS30426891", Customer: "Arjun Mehta", Subject: "Refund not received", Priority: "Medium", Date: "24 Apr 2024", Status: "In Progress"},
			{Case: "#WCS30426577", Customer: "Aisha Khan", Subject: "Wrong product", Priority: "High", Date: "23 Apr 2024", Status: "Open"},
			{Case: "#WCS30426012", Customer: "Vikram Desai", Subject: "Quality issue", Priority: "Medium", Date: "23 Apr 2024", Status: "Under Review"},
		}
	case "marketing":
		page.Title = "Marketing & Content"
		page.Subtitle = "Create compelling stories, run campaigns and showcase the best of Indian craftsmanship."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "bar_chart", Value: "INR 12,46,380", Label: "Campaign Revenue (GMV)", Delta: "28%", Positive: true, Period: "vs. last month"},
			{Icon: "visibility", Value: "4,82,671", Label: "Campaign Visits", Delta: "22%", Positive: true, Period: "vs. last month"},
			{Icon: "groups", Value: "6.8%", Label: "Campaign Conversion", Delta: "1.4%", Positive: true, Period: "vs. last month"},
			{Icon: "sell", Value: "32", Label: "Active Coupons", Delta: "14%", Positive: true, Period: "vs. last month"},
			{Icon: "description", Value: "14", Label: "Content Pieces", Delta: "27%", Positive: true, Period: "vs. last month"},
		}
		page.Campaigns = []viewmodels.AdminCampaign{
			{Name: "Crafted for Summer", Type: "Sale", Start: "01 Apr 2024", End: "30 Apr 2024", Status: "Active", Channel: "Website + Email", Reach: "1,24,360", Revenue: "INR 4,82,390"},
			{Name: "Handmade Home Week", Type: "Offer", Start: "15 Apr 2024", End: "30 Apr 2024", Status: "Active", Channel: "Website + Social", Reach: "96,210", Revenue: "INR 3,12,450"},
			{Name: "Makers of India", Type: "Awareness", Start: "01 Apr 2024", End: "15 May 2024", Status: "Active", Channel: "Social + Email", Reach: "2,18,450", Revenue: "INR 2,76,890"},
			{Name: "Festive Craft Deals", Type: "Sale", Start: "10 Mar 2024", End: "31 Mar 2024", Status: "Ended", Channel: "Website", Reach: "1,98,320", Revenue: "INR 1,74,650"},
		}
	case "analytics":
		page.Title = "Analytics & Settings"
		page.Subtitle = "Data-driven growth. A safer, stronger marketplace for Indian artisans."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "shopping_cart", Value: "INR 48,23,650", Label: "Total GMV", Delta: "18%", Positive: true, Period: "vs. previous 30 days"},
			{Icon: "groups", Value: "28,450", Label: "Total Sellers", Delta: "14%", Positive: true, Period: "vs. previous 30 days"},
			{Icon: "inventory_2", Value: "12,840", Label: "Active Products", Delta: "22%", Positive: true, Period: "vs. previous 30 days"},
			{Icon: "shopping_cart", Value: "78,760", Label: "Total Orders", Delta: "16%", Positive: true, Period: "vs. previous 30 days"},
			{Icon: "undo", Value: "2,341", Label: "Returns & Refunds", Delta: "8%", Period: "vs. previous 30 days"},
		}
		page.AuditEntries = []viewmodels.AdminAuditEntry{
			{Date: "27 Apr 2024, 11:42 AM", User: "Rohan Mehta", Action: "Updated", Entity: "Policy", Details: "Updated Return Policy to v2.1", IP: "122.168.1.24"},
			{Date: "27 Apr 2024, 10:21 AM", User: "Priya Sharma", Action: "Approved", Entity: "Seller", Details: "Approved Heritage Looms", IP: "122.168.1.11"},
			{Date: "26 Apr 2024, 06:15 PM", User: "Arjun Nair", Action: "Changed", Entity: "Settings", Details: "Updated COD threshold", IP: "122.168.1.56"},
			{Date: "26 Apr 2024, 04:03 PM", User: "Rohan Mehta", Action: "Exported", Entity: "Report", Details: "Downloaded Sales Report", IP: "122.168.1.24"},
			{Date: "26 Apr 2024, 11:18 AM", User: "Neha Kapoor", Action: "Suspended", Entity: "Seller", Details: "Suspended Kashmir Creations", IP: "122.168.1.39"},
		}
		page.Configs = []viewmodels.AdminConfig{
			{Icon: "category", Name: "Default Return Window", Value: "7 Days", Detail: "Default return period for all products"},
			{Icon: "local_shipping", Name: "COD Rules", Value: "COD up to INR 5,000", Detail: "Cash on Delivery limit per order"},
			{Icon: "public", Name: "Enabled Countries", Value: "12 Countries", Detail: "USA, UK, UAE, Australia +8 more"},
			{Icon: "database", Name: "Commission Settings", Value: "5% - 16%", Detail: "Category-wise commissions"},
			{Icon: "schedule", Name: "Auto Order Cancellation", Value: "3 Days", Detail: "Cancel unshipped orders automatically"},
			{Icon: "person_add", Name: "New Seller Approval", Value: "Manual Review", Detail: "All new sellers require admin approval"},
		}
	case "security":
		page.Title = "Security & Access"
		page.Subtitle = "Review roles, sensitive permissions, break-glass access and the audit trail for privileged actions."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "verified_user", Value: "8", Label: "Marketplace Roles", Delta: "0%", Period: "configured roles"},
			{Icon: "admin_panel_settings", Value: "24", Label: "Permission Rules", Delta: "100%", Positive: true, Period: "reviewed"},
			{Icon: "security_update_good", Value: "100%", Label: "Admin MFA Coverage", Delta: "0%", Positive: true, Period: "preview policy"},
			{Icon: "history", Value: "42", Label: "Privileged Events", Delta: "18%", Period: "last 24 hours"},
		}
	default:
		page.Active = "dashboard"
		page.Title = "Admin Dashboard"
		page.Subtitle = "Complete overview of the WeeVCrafts marketplace. Monitor, manage and grow our handcrafted ecosystem."
		page.Stats = []viewmodels.AdminStat{
			{Icon: "shopping_cart", Value: "INR 48,23,650", Label: "Gross Merchandise Value (GMV)", Delta: "18%", Positive: true, Period: "vs. last month"},
			{Icon: "receipt_long", Value: "1,842", Label: "Total Orders", Delta: "12%", Positive: true, Period: "vs. last month"},
			{Icon: "database", Value: "INR 2,41,183", Label: "Commission Revenue (5%)", Delta: "16%", Positive: true, Period: "vs. last month"},
			{Icon: "groups", Value: "326", Label: "Active Sellers", Delta: "8%", Positive: true, Period: "vs. last month"},
			{Icon: "person", Value: "28,450", Label: "Total Customers", Delta: "14%", Positive: true, Period: "vs. last month"},
			{Icon: "person_add", Value: "12", Label: "Pending Seller Approvals", Delta: "25%", Period: "vs. last month"},
			{Icon: "undo", Value: "18", Label: "Pending Returns/Refunds", Delta: "38%", Period: "vs. last month"},
			{Icon: "public", Value: "214", Label: "Export Orders", Delta: "50%", Positive: true, Period: "vs. last month"},
			{Icon: "warehouse", Value: "INR 8,76,320", Label: "Own Inventory Sales", Delta: "22%", Positive: true, Period: "vs. last month"},
		}
		page.Bars = []viewmodels.AdminBar{
			{Label: "28 Mar", Height: "48", Value: "INR 7.2L"}, {Label: "2 Apr", Height: "64", Value: "INR 9.4L"}, {Label: "7 Apr", Height: "76", Value: "INR 11.0L"}, {Label: "12 Apr", Height: "69", Value: "INR 10.1L"}, {Label: "17 Apr", Height: "88", Value: "INR 12.8L"}, {Label: "22 Apr", Height: "96", Value: "INR 14.2L"}, {Label: "27 Apr", Height: "100", Value: "INR 16.4L"},
		}
		page.Donut = []viewmodels.AdminDonut{{Label: "Sarees", Value: "42%", Color: "red"}, {Label: "Handicrafts", Value: "28%", Color: "green"}, {Label: "Home Decor", Value: "12%", Color: "gold"}, {Label: "Paintings & Arts", Value: "10%", Color: "orange"}, {Label: "Jewellery", Value: "5%", Color: "sand"}, {Label: "Others", Value: "3%", Color: "gray"}}
	}
	if requested == active {
		page.Workspace = ""
	} else {
		switch requested {
		case "brands":
			page.Title = "Brands & Categories"
			page.Subtitle = "Organize the marketplace taxonomy, merchandising labels and maker-facing category rules."
		case "seller-approvals":
			page.Title = "Seller Approvals"
			page.Subtitle = "Review verification evidence and approve makers without losing the decision trail."
		case "seller-details":
			page.Title = "Seller Details"
			page.Subtitle = "Inspect seller profile, verification, catalogue and fulfilment context."
		case "seller-suspensions":
			page.Title = "Seller Suspensions"
			page.Subtitle = "Review risk signals and apply a documented seller-access decision."
		case "product-details":
			page.Title = "Product Details"
			page.Subtitle = "Review product content, maker context, compliance claims and moderation history."
		case "category-management":
			page.Title = "Category Management"
			page.Subtitle = "Organize category rules, attributes and merchandising placement."
		case "customers":
			page.Title = "Customers"
			page.Subtitle = "Search customer profiles, order context, support history and communication preferences."
		case "payments":
			page.Title = "Payment Operations"
			page.Subtitle = "Review captured payments, settlements, refunds and reconciliation status."
		case "failed-payments":
			page.Title = "Failed Payments"
			page.Subtitle = "Investigate payment failures and route safe retry or support actions."
			page.Transactions = []viewmodels.AdminTransaction{
				{Date: "26 Apr 2024, 02:18 PM", Type: "Payment", Reference: "#PAY25603488", Description: "UPI collect request expired", Amount: "INR 3,299", Status: "Failed"},
				{Date: "25 Apr 2024, 07:40 PM", Type: "Payment", Reference: "#PAY25602912", Description: "Bank declined the card authorization", Amount: "INR 8,298", Status: "Failed"},
				{Date: "24 Apr 2024, 11:03 AM", Type: "Refund", Reference: "#REF25601438", Description: "Refund callback pending reconciliation", Amount: "INR 1,499", Status: "Under Review"},
			}
		case "disputes":
			page.Title = "Disputes"
			page.Subtitle = "Review return evidence and make traceable marketplace decisions."
		case "refunds":
			page.Title = "Refund Operations"
			page.Subtitle = "Review refund requests, provider references and reconciliation status."
		case "reviews":
			page.Title = "Review Moderation"
			page.Subtitle = "Moderate customer reviews while preserving maker and customer context."
		case "coupons":
			page.Title = "Coupons"
			page.Subtitle = "Manage coupon eligibility, limits, dates and marketplace-funded offers."
		case "commission-rules":
			page.Title = "Commission Rules"
			page.Subtitle = "Review category commission rules before they affect seller settlements."
		case "settlements":
			page.Title = "Seller Settlements"
			page.Subtitle = "Review seller payout batches, adjustments and release status."
		case "finance-reconciliation":
			page.Title = "Finance Reconciliation"
			page.Subtitle = "Compare payment, refund, commission and settlement records."
		case "roles":
			page.Title = "Role Management"
			page.Subtitle = "Manage least-privilege roles, permission rules and privileged access review."
		case "audit-logs":
			page.Title = "Audit Logs"
			page.Subtitle = "Trace privileged actions, actor context and request IDs."
		case "system-health":
			page.Title = "System Health"
			page.Subtitle = "Review dependency health, availability signals and operational notices."
		case "search-indexing-health":
			page.Title = "Search Indexing Health"
			page.Subtitle = "Review indexing freshness, rebuild state and fallback behaviour."
		case "worker-queue-health":
			page.Title = "Worker Queue Health"
			page.Subtitle = "Review background work, retries and bounded queue pressure."
		case "security-events":
			page.Title = "Security Events"
			page.Subtitle = "Review sensitive access events and protected-action outcomes."
		case "settings":
			page.Title = "Marketplace Settings"
			page.Subtitle = "Review policy, fulfilment, payment and marketplace configuration."
		}
	}
	page = filterAdminPreview(page, values)
	return page
}

func filterAdminPreview(page viewmodels.AdminPage, values url.Values) viewmodels.AdminPage {
	query := strings.ToLower(strings.TrimSpace(values.Get("q")))
	if query != "" {
		page.Sellers = filterAdminSellers(page.Sellers, query)
		page.Products = filterAdminProducts(page.Products, query)
		page.Inventory = filterAdminInventory(page.Inventory, query)
		page.Orders = filterAdminOrders(page.Orders, query)
		page.Returns = filterAdminReturns(page.Returns, query)
		page.SupportCases = filterAdminSupportCases(page.SupportCases, query)
		page.Settlements = filterAdminSettlements(page.Settlements, query)
		page.Transactions = filterAdminTransactions(page.Transactions, query)
	}

	category := strings.ToLower(strings.TrimSpace(values.Get("category")))
	if category != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.AdminProduct) bool {
			return strings.Contains(strings.ToLower(item.Category), strings.ReplaceAll(category, "-", " "))
		})
	}
	if status := strings.ToLower(strings.TrimSpace(values.Get("status"))); status != "" {
		page.Products = filterBy(page.Products, func(item viewmodels.AdminProduct) bool {
			return strings.Contains(strings.ToLower(item.Status), status)
		})
		page.Orders = filterBy(page.Orders, func(item viewmodels.AdminOrder) bool {
			return strings.Contains(strings.ToLower(item.Fulfillment), status)
		})
	}
	if stock := strings.ToLower(strings.TrimSpace(values.Get("stock"))); stock != "" {
		page.Inventory = filterBy(page.Inventory, func(item viewmodels.AdminInventory) bool {
			return stock == "low" && strings.Contains(strings.ToLower(item.Status), "low") || stock == "in-stock" && strings.Contains(strings.ToLower(item.Status), "in stock")
		})
	}
	if warehouse := strings.ToLower(strings.TrimSpace(values.Get("warehouse"))); warehouse != "" {
		page.Inventory = filterBy(page.Inventory, func(item viewmodels.AdminInventory) bool {
			return strings.Contains(strings.ToLower(item.Location), warehouse)
		})
	}
	if orderType := strings.ToLower(strings.TrimSpace(values.Get("type"))); orderType != "" {
		page.Orders = filterBy(page.Orders, func(item viewmodels.AdminOrder) bool {
			return strings.EqualFold(item.Type, orderType)
		})
	}
	return page
}

func filterBy[T any](items []T, keep func(T) bool) []T {
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterAdminSellers(items []viewmodels.AdminSeller, query string) []viewmodels.AdminSeller {
	return filterBy(items, func(item viewmodels.AdminSeller) bool {
		return containsAny(query, item.Name, item.Location, item.Category, item.Status)
	})
}

func filterAdminProducts(items []viewmodels.AdminProduct, query string) []viewmodels.AdminProduct {
	return filterBy(items, func(item viewmodels.AdminProduct) bool {
		return containsAny(query, item.Name, item.SKU, item.Seller, item.Category, item.Status)
	})
}

func filterAdminInventory(items []viewmodels.AdminInventory, query string) []viewmodels.AdminInventory {
	return filterBy(items, func(item viewmodels.AdminInventory) bool {
		return containsAny(query, item.Name, item.SKU, item.Location, item.Status)
	})
}

func filterAdminOrders(items []viewmodels.AdminOrder, query string) []viewmodels.AdminOrder {
	return filterBy(items, func(item viewmodels.AdminOrder) bool {
		return containsAny(query, item.Number, item.Customer, item.Type, item.Destination, item.Fulfillment, item.Delivery)
	})
}

func filterAdminReturns(items []viewmodels.AdminReturn, query string) []viewmodels.AdminReturn {
	return filterBy(items, func(item viewmodels.AdminReturn) bool {
		return containsAny(query, item.Number, item.Customer, item.Product, item.Reason, item.Status)
	})
}

func filterAdminSupportCases(items []viewmodels.AdminSupportCase, query string) []viewmodels.AdminSupportCase {
	return filterBy(items, func(item viewmodels.AdminSupportCase) bool {
		return containsAny(query, item.Case, item.Customer, item.Subject, item.Priority, item.Status)
	})
}

func filterAdminSettlements(items []viewmodels.AdminSettlement, query string) []viewmodels.AdminSettlement {
	return filterBy(items, func(item viewmodels.AdminSettlement) bool {
		return containsAny(query, item.Seller, item.Status, item.Date)
	})
}

func filterAdminTransactions(items []viewmodels.AdminTransaction, query string) []viewmodels.AdminTransaction {
	return filterBy(items, func(item viewmodels.AdminTransaction) bool {
		return containsAny(query, item.Reference, item.Type, item.Description, item.Status)
	})
}

func containsAny(query string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
