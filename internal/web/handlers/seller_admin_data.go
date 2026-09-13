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
	if active == "payments" || active == "payouts" {
		active = "earnings"
	}
	if active == "profile" || active == "store-profile" {
		active = "settings"
	}

	page := viewmodels.SellerAdminPage{
		Active: active,
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
			{Name: "Priya Sharma", Email: "priya@weaverstouch.in", Role: "Store owner", Status: "Active", LastActive: "Now", Permissions: "Everything", Image: "/assets/images/customer/maker-mithila.png"},
			{Name: "Rohan Mehta", Email: "rohan@weaverstouch.in", Role: "Catalog manager", Status: "Active", LastActive: "12 min ago", Permissions: "Products, media, inventory", Image: "/assets/images/customer/saree-blue.png"},
			{Name: "Aditi Rao", Email: "aditi@weaverstouch.in", Role: "Order assistant", Status: "Active", LastActive: "1 hour ago", Permissions: "Orders, fulfilment, returns", Image: "/assets/images/customer/ceramic-mugs.png"},
			{Name: "Neha Kapoor", Email: "neha@weaverstouch.in", Role: "Finance viewer", Status: "Invite pending", LastActive: "Not active yet", Permissions: "Read-only earnings", Image: "/assets/images/customer/wooden-box.png"},
		}
	default:
		page.Active = "dashboard"
		page.Title = "Dashboard"
		page.Subtitle = "Welcome back! Here's what's happening with your store today."
		page.Stats = sellerDashboardStats()
	}
	return page
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
		{Name: "Chanderi Silk Cotton Saree - Royal Maroon", SKU: "WT-CSC-001", Category: "Sarees", Price: "INR 5,999", Variants: "6 colors", Stock: "12", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/saree-maroon.png"},
		{Name: "Maheshwari Dupatta - Indigo Waves", SKU: "WT-MD-002", Category: "Dupattas", Price: "INR 2,499", Variants: "4 colors", Stock: "25", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/saree-blue.png"},
		{Name: "Block Print Cushion Cover - Marigold", SKU: "WT-BPC-003", Category: "Home Decor", Price: "INR 899", Variants: "3 designs", Stock: "48", Status: "Under Review", Availability: "D E", Image: "/assets/images/customer/black-cushion.png"},
		{Name: "Handwoven Tote Bag", SKU: "WT-HTB-004", Category: "Bags & Accessories", Price: "INR 1,899", Variants: "2 variants", Stock: "0", Status: "Out of Stock", Availability: "D E", Image: "/assets/images/customer/blue-tote.png"},
		{Name: "Maheshwari Silk Saree - Forest Green", SKU: "WT-MSS-005", Category: "Sarees", Price: "INR 6,499", Variants: "4 colors", Stock: "8", Status: "Changes Required", Availability: "D", Image: "/assets/images/customer/saree-blue.png"},
		{Name: "Block Print Table Runner - Vine Motif", SKU: "WT-BPR-006", Category: "Home Decor", Price: "INR 1,299", Variants: "2 variants", Stock: "15", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/black-cushion.png"},
		{Name: "Chanderi Dupatta - Blush Pink", SKU: "WT-CD-007", Category: "Dupattas", Price: "INR 2,299", Variants: "3 colors", Stock: "6", Status: "Rejected", Availability: "D E", Image: "/assets/images/customer/saree-maroon.png"},
		{Name: "Madhubani Wall Art - Tree of Life", SKU: "WT-MWA-008", Category: "Wall Decor", Price: "INR 2,999", Variants: "1 design", Stock: "10", Status: "Approved", Availability: "D E", Image: "/assets/images/customer/madhubani-tree.png"},
	}
}

func sellerInventory() []viewmodels.SellerAdminInventory {
	return []viewmodels.SellerAdminInventory{
		{Name: "Chanderi Silk Cotton Saree", SKU: "CWC-SAREE-001", Variant: "Royal Maroon", Location: "Bengaluru", Available: "2", Reserved: "1", Reorder: "5", Status: "Low Stock", Image: "/assets/images/customer/saree-maroon.png"},
		{Name: "Chanderi Silk Cotton Saree", SKU: "CWC-SAREE-002", Variant: "Peacock Blue", Location: "Bengaluru", Available: "14", Reserved: "0", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/saree-blue.png"},
		{Name: "Tussar Silk Dupatta", SKU: "CWC-DUP-001", Variant: "Mustard", Location: "Bengaluru", Available: "1", Reserved: "0", Reorder: "5", Status: "Low Stock", Image: "/assets/images/customer/saree-maroon.png"},
		{Name: "Tussar Silk Dupatta", SKU: "CWC-DUP-002", Variant: "Olive Green", Location: "Bengaluru", Available: "12", Reserved: "2", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/saree-blue.png"},
		{Name: "Block Print Cotton Saree", SKU: "CWC-SAREE-003", Variant: "Indigo", Location: "Bengaluru", Available: "0", Reserved: "1", Reorder: "5", Status: "Out of Stock", Image: "/assets/images/customer/saree-blue.png"},
		{Name: "Handpainted Ceramic Mugs", SKU: "CWC-MUG-001", Variant: "Set of 2", Location: "Bengaluru", Available: "3", Reserved: "2", Reorder: "5", Status: "Low Stock", Image: "/assets/images/customer/ceramic-mugs.png"},
		{Name: "Carved Wooden Jewellery Box", SKU: "CWC-BOX-001", Variant: "Natural Brown", Location: "Bengaluru", Available: "8", Reserved: "0", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/wooden-box.png"},
		{Name: "Madhubani Wall Art", SKU: "CWC-ART-001", Variant: "Tree of Life", Location: "Bengaluru", Available: "5", Reserved: "1", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/madhubani-tree.png"},
		{Name: "Ikat Cotton Saree", SKU: "CWC-SAREE-004", Variant: "Rust Orange", Location: "Bengaluru", Available: "0", Reserved: "0", Reorder: "5", Status: "Out of Stock", Image: "/assets/images/customer/saree-maroon.png"},
		{Name: "Linen Stole", SKU: "CWC-STOLE-001", Variant: "Beige", Location: "Bengaluru", Available: "11", Reserved: "0", Reorder: "5", Status: "In Stock", Image: "/assets/images/customer/saree-blue.png"},
	}
}

func sellerOrders() []viewmodels.SellerAdminOrder {
	return []viewmodels.SellerAdminOrder{
		{Number: "#WC2504267831", Date: "26 Apr 2024, 10:24 AM", Customer: "Priya Sharma", Items: "2 items", Value: "INR 5,998", Type: "Domestic", Payment: "Paid", Status: "New", Destination: "Bengaluru, KA", Image: "/assets/images/customer/saree-maroon.png"},
		{Number: "#WC2504267790", Date: "25 Apr 2024, 04:15 PM", Customer: "Rahul Mehta", Items: "1 item", Value: "INR 2,499", Type: "Domestic", Payment: "Paid", Status: "Processing", Destination: "Mumbai, MH", Image: "/assets/images/customer/saree-blue.png"},
		{Number: "#WC2504253312", Date: "24 Apr 2024, 11:30 AM", Customer: "Ananya Nair", Items: "3 items", Value: "INR 8,298", Type: "Domestic", Payment: "Paid", Status: "Ready to Ship", Destination: "Chennai, TN", Image: "/assets/images/customer/ceramic-mugs.png"},
		{Number: "#WC2504189921", Date: "18 Apr 2024, 03:42 PM", Customer: "Sophia Lee", Items: "2 items", Value: "INR 12,499", Type: "Export", Payment: "Paid", Status: "Shipped", Destination: "New York, USA", Image: "/assets/images/customer/madhubani-tree.png"},
		{Number: "#WC250415520", Date: "01 Apr 2024, 09:45 AM", Customer: "Meera Krishnan", Items: "4 items", Value: "INR 6,247", Type: "Domestic", Payment: "Refunded", Status: "Delivered", Destination: "Bengaluru, KA", Image: "/assets/images/customer/wooden-box.png"},
	}
}

func sellerReturns() []viewmodels.SellerAdminReturn {
	return []viewmodels.SellerAdminReturn{
		{Number: "#RTN250427001", Order: "#WC2504267819", Customer: "Priya Sharma", Product: "Chanderi Silk Cotton Saree - Royal Maroon", Reason: "Received a different item", Date: "28 Apr 2024, 10:24 AM", Status: "New Request", Amount: "INR 5,999", Image: "/assets/images/customer/saree-maroon.png"},
		{Number: "#RTN2504269982", Order: "#WC2504267990", Customer: "Rahul Mehta", Product: "Handpainted Ceramic Mugs (Set of 2)", Reason: "Damaged item", Date: "26 Apr 2024", Status: "Awaiting Response", Amount: "INR 1,299", Image: "/assets/images/customer/ceramic-mugs.png"},
		{Number: "#RTN2504245561", Order: "#WC2504245331", Customer: "Sneha Iyer", Product: "Madhubani Painting - Tree of Life", Reason: "Not as described", Date: "24 Apr 2024", Status: "Under Review", Amount: "INR 2,499", Image: "/assets/images/customer/madhubani-tree.png"},
		{Number: "#RTN2504203321", Order: "#WC2504201172", Customer: "Arjun Nair", Product: "Carved Wooden Jewellery Box", Reason: "Item damaged", Date: "20 Apr 2024", Status: "Approved", Amount: "INR 1,299", Image: "/assets/images/customer/wooden-box.png"},
		{Number: "#RTN2504187712", Order: "#WC2504189921", Customer: "Kavita Rao", Product: "Handpainted Ceramic Mugs (Set of 2)", Reason: "Change of mind", Date: "18 Apr 2024", Status: "Rejected", Amount: "INR 2,598", Image: "/assets/images/customer/ceramic-mugs.png"},
		{Number: "#RTN2504156677", Order: "#WC250415520", Customer: "Vikram Desai", Product: "Chanderi Silk Cotton Saree - Royal Maroon", Reason: "Size issue", Date: "15 Apr 2024", Status: "Completed", Amount: "INR 5,999", Image: "/assets/images/customer/saree-maroon.png"},
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
