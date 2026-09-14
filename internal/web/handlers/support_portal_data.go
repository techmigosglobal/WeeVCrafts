package handlers

import (
	"net/url"
	"strings"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

func supportPortalPage(path string, values url.Values) viewmodels.SupportPortalPage {
	active := strings.TrimPrefix(path, "/support-portal")
	if active == "" || active == "/" {
		active = "dashboard"
	} else {
		active = strings.TrimPrefix(active, "/")
	}
	if active == "supporter-portal" {
		active = "dashboard"
	}

	page := viewmodels.SupportPortalPage{
		Active: active,
		Query:  values.Get("q"),
		Range:  values.Get("range"),
		Tab:    values.Get("tab"),
		Notice: values.Get("notice"),
	}
	if page.Range == "" {
		page.Range = "Last 30 days"
	}

	switch active {
	case "dashboard":
		page.Title = "Support Dashboard"
		page.Subtitle = "Welcome back, Aditi! Here's what's happening with customer support today."
		page.Stats = supportDashboardStats()
		page.Cases = supportCases()
		page.Conversations = supportConversations()
	case "cases":
		page.Title = "Cases"
		page.Subtitle = "All customer and vendor support tickets and conversations."
		page.Cases = supportCases()
		page.Conversations = supportConversations()
	case "customers":
		page.Title = "Customers & Sellers"
		page.Subtitle = "Get complete customer and seller context to resolve cases faster. Search, view profiles, orders, support history and more."
		page.Cases = supportCases()
		page.Directory = supportDirectory()
		page.Timeline = supportCaseTimeline()
	case "orders":
		page.Title = "Order / Transaction View"
		page.Subtitle = "Complete order, payment, shipment, return and refund details. Investigate, resolve and support with confidence."
		page.Cases = supportCases()
		page.Timeline = supportOrderTimeline()
	default:
		page.Active = "dashboard"
		page.Title = "Support Dashboard"
		page.Subtitle = "Welcome back, Aditi! Here's what's happening with customer support today."
		page.Stats = supportDashboardStats()
		page.Cases = supportCases()
		page.Conversations = supportConversations()
	}

	return filterSupportPreview(page, values)
}

func filterSupportPreview(page viewmodels.SupportPortalPage, values url.Values) viewmodels.SupportPortalPage {
	query := strings.ToLower(strings.TrimSpace(values.Get("q")))
	if query != "" {
		page.Cases = filterBy(page.Cases, func(item viewmodels.SupportPortalCase) bool {
			return containsAny(query, item.ID, item.Subject, item.Customer, item.Category, item.Priority, item.Status, item.Seller, item.Order)
		})
		page.Conversations = filterBy(page.Conversations, func(item viewmodels.SupportPortalConversation) bool {
			return containsAny(query, item.Author, item.Role, item.Message)
		})
		page.Directory = filterBy(page.Directory, func(item viewmodels.SupportPortalDirectoryEntry) bool {
			return containsAny(query, item.Name, item.EmailPhone, item.Type, item.Location, item.Status)
		})
	}
	if tab := strings.ToLower(strings.TrimSpace(values.Get("tab"))); tab != "" {
		page.Cases = filterBy(page.Cases, func(item viewmodels.SupportPortalCase) bool {
			switch tab {
			case "open":
				return strings.EqualFold(item.Status, "Open")
			case "pending":
				return strings.EqualFold(item.Status, "Pending")
			case "urgent":
				return strings.EqualFold(item.Priority, "High")
			case "escalated":
				return strings.EqualFold(item.Status, "Escalated")
			case "resolved":
				return strings.EqualFold(item.Status, "Resolved")
			default:
				return true
			}
		})
	}
	if userType := strings.ToLower(strings.TrimSpace(values.Get("type"))); userType != "" {
		page.Directory = filterBy(page.Directory, func(item viewmodels.SupportPortalDirectoryEntry) bool { return strings.EqualFold(item.Type, userType) })
	}
	return page
}

func supportDashboardStats() []viewmodels.SupportPortalStat {
	return []viewmodels.SupportPortalStat{
		{Icon: "headset_mic", Value: "128", Label: "Open Cases", Delta: "12%", Period: "vs. yesterday", Positive: true},
		{Icon: "warning", Value: "18", Label: "Urgent Cases", Delta: "50%", Period: "vs. yesterday", Tone: "red"},
		{Icon: "schedule", Value: "42", Label: "Pending Cases", Delta: "8%", Period: "vs. yesterday", Positive: true},
		{Icon: "groups", Value: "7", Label: "Escalated Cases", Delta: "75%", Period: "vs. yesterday", Tone: "red"},
		{Icon: "storefront", Value: "36", Label: "Awaiting Seller Response", Delta: "20%", Period: "vs. yesterday", Tone: "red"},
		{Icon: "person", Value: "28", Label: "Awaiting Customer Response", Delta: "7%", Period: "vs. yesterday", Positive: true},
		{Icon: "check_circle", Value: "56", Label: "Resolved Today", Delta: "24%", Period: "vs. yesterday", Positive: true, Tone: "green"},
		{Icon: "schedule", Value: "2h 18m", Label: "Avg. First Response Time", Delta: "28%", Period: "vs. yesterday", Positive: true, Tone: "red"},
	}
}

func supportCases() []viewmodels.SupportPortalCase {
	return []viewmodels.SupportPortalCase{
		{ID: "#WCSP250426781", Subject: "Order not delivered", Customer: "Priya Sharma", CustomerInitials: "PS", Category: "Shipping", Priority: "High", Created: "26 Apr 10:24 AM", Status: "Open", Age: "2h 18m", Seller: "Weaver's Touch", Order: "#WC2504267831", Avatar: "/assets/images/customer/maker-mithila.webp", Assignee: "Aditi Rao", Channel: "Email"},
		{ID: "#WCSP250426799", Subject: "Refund not received", Customer: "Rahul Mehta", CustomerInitials: "RM", Category: "Refunds", Priority: "High", Created: "26 Apr 09:17 AM", Status: "Open", Age: "4h 05m", Seller: "Weaver's Touch", Order: "#WC2504267790", Avatar: "/assets/images/customer/ceramic-mugs.webp", Assignee: "Karan Bhat", Channel: "Chat"},
		{ID: "#WCSP250426789", Subject: "Received different item", Customer: "Ananya Nair", CustomerInitials: "AN", Category: "Returns", Priority: "Medium", Created: "26 Apr 08:56 AM", Status: "Pending", Age: "12h 30m", Seller: "Mithila Arts", Order: "#WC2504199921", Avatar: "/assets/images/customer/saree-blue.webp", Assignee: "Sneha Patel", Channel: "Email"},
		{ID: "#WCSP250426788", Subject: "Product quality issue", Customer: "Karan Bhat", CustomerInitials: "KB", Category: "Product", Priority: "Medium", Created: "26 Apr 08:32 AM", Status: "Open", Age: "9h 12m", Seller: "Weaver's Touch", Order: "#WC2504267812", Avatar: "/assets/images/customer/wooden-box.webp", Assignee: "Meera Kapoor", Channel: "Web"},
		{ID: "#WCSP250426787", Subject: "Customization request", Customer: "Sneha Patel", CustomerInitials: "SP", Category: "Customization", Priority: "Low", Created: "26 Apr 07:41 AM", Status: "Escalated", Age: "1d 6h", Seller: "CraftVilla", Order: "#WC2504267805", Avatar: "/assets/images/customer/madhubani-tree.webp", Assignee: "Arjun Nair", Channel: "Chat"},
		{ID: "#WCSP250426786", Subject: "Damaged item received", Customer: "Arjun Nair", CustomerInitials: "AR", Category: "Returns", Priority: "High", Created: "26 Apr 06:12 AM", Status: "Open", Age: "3h 40m", Seller: "Heritage Looms", Order: "#WC2504189945", Avatar: "/assets/images/customer/terracotta-lamp.webp", Assignee: "Aditi Rao", Channel: "Email"},
		{ID: "#WCSP250426785", Subject: "Seller not responding", Customer: "Vikram Desai", CustomerInitials: "VD", Category: "Seller Support", Priority: "High", Created: "26 Apr 05:40 AM", Status: "Escalated", Age: "5h 22m", Seller: "Weaver's Touch", Order: "#WC2504178821", Avatar: "/assets/images/customer/saree-maroon.webp", Assignee: "Karan Bhat", Channel: "Web"},
		{ID: "#WCSP250426784", Subject: "Cancel order request", Customer: "Neha Kulkarni", CustomerInitials: "NK", Category: "Order Mgmt", Priority: "Medium", Created: "26 Apr 04:15 AM", Status: "Pending", Age: "11h 18m", Seller: "Mithila Arts", Order: "#WC2504267783", Avatar: "/assets/images/customer/blue-tote.webp", Assignee: "Meera Kapoor", Channel: "Email"},
		{ID: "#WCSP250426783", Subject: "Payment issue", Customer: "Meera Kapoor", CustomerInitials: "MK", Category: "Payments", Priority: "Medium", Created: "26 Apr 03:02 AM", Status: "Open", Age: "16h 05m", Seller: "CraftVilla", Order: "#WC2504267751", Avatar: "/assets/images/customer/ceramic-mugs.webp", Assignee: "Sneha Patel", Channel: "Chat"},
		{ID: "#WCSP250426782", Subject: "Bulk order enquiry", Customer: "CraftVilla", CustomerInitials: "CV", Category: "Seller Support", Priority: "Low", Created: "25 Apr 08:10 PM", Status: "Pending", Age: "1d 12h", Seller: "CraftVilla", Order: "#WC2504267719", Avatar: "/assets/images/customer/wooden-box.webp", Assignee: "Arjun Nair", Channel: "Phone"},
	}
}

func supportConversations() []viewmodels.SupportPortalConversation {
	return []viewmodels.SupportPortalConversation{
		{Author: "Priya Sharma", Initials: "PS", Role: "Customer", Message: "Hi, my order #WC250426781 hasn't arrived yet. Can you please check?", Time: "10:24 AM", Tone: "customer", Image: "/assets/images/customer/maker-mithila.webp"},
		{Author: "Rahul Mehta", Initials: "RM", Role: "Customer", Message: "I wanted to check on the refund status for my cancelled order.", Time: "10:17 AM", Tone: "customer", Image: "/assets/images/customer/ceramic-mugs.webp"},
		{Author: "Sneha Iyer", Initials: "SI", Role: "Customer", Message: "The item I received is damaged. Here are the photos.", Time: "09:56 AM", Tone: "customer", Image: "/assets/images/customer/madhubani-tree.webp"},
		{Author: "Ananya Nair", Initials: "AN", Role: "Customer", Message: "Is this product handmade? Can you share more about the maker?", Time: "09:32 AM", Tone: "customer", Image: "/assets/images/customer/saree-blue.webp"},
		{Author: "Vikram Desai", Initials: "VD", Role: "Customer", Message: "The seller is not responding to my messages.", Time: "09:10 AM", Tone: "customer", Image: "/assets/images/customer/wooden-box.webp"},
	}
}

func supportDirectory() []viewmodels.SupportPortalDirectoryEntry {
	return []viewmodels.SupportPortalDirectoryEntry{
		{Name: "Priya Sharma", EmailPhone: "p***a.sharma@gmail.com · +91 ******5219", Type: "Customer", Location: "Bengaluru, KA", Orders: "12", Cases: "3", Status: "Active", LastActive: "26 Apr 2024", Image: "/assets/images/customer/maker-mithila.webp"},
		{Name: "Weaver's Touch", EmailPhone: "weaverstouch@gmail.com", Type: "Seller", Location: "Bengaluru, KA", Orders: "248", Cases: "7", Status: "Active", LastActive: "26 Apr 2024", Image: "/assets/images/customer/saree-maroon.webp"},
		{Name: "Sneha Iyer", EmailPhone: "s***a.iyer@gmail.com · +91 ******5679", Type: "Customer", Location: "Chennai, TN", Orders: "6", Cases: "2", Status: "Active", LastActive: "25 Apr 2024", Image: "/assets/images/customer/madhubani-tree.webp"},
		{Name: "Madhubani Arts", EmailPhone: "sophie@artstore.com", Type: "Seller", Location: "New York, USA", Orders: "84", Cases: "4", Status: "Active", LastActive: "24 Apr 2024", Image: "/assets/images/customer/madhubani-tree.webp"},
	}
}

func supportCaseTimeline() []viewmodels.SupportPortalTimelineEntry {
	return []viewmodels.SupportPortalTimelineEntry{
		{Time: "26 Apr 2024, 10:24 AM", Title: "Case created", Detail: "Customer reported order not delivered and requested refund.", Tone: "current"},
		{Time: "26 Apr 2024, 11:02 AM", Title: "Customer response", Detail: "I haven't received the order. Please check and process refund if not delivered.", Tone: "done"},
		{Time: "26 Apr 2024, 01:17 PM", Title: "Seller response", Detail: "Item was dispatched on 24 Apr via Shiprocket. Tracking shared.", Tone: "done"},
		{Time: "26 Apr 2024, 02:30 PM", Title: "Agent note", Detail: "Checking with logistics partner. Will update customer within 24 hours.", Tone: "pending"},
	}
}

func supportOrderTimeline() []viewmodels.SupportPortalTimelineEntry {
	return []viewmodels.SupportPortalTimelineEntry{
		{Time: "26 Apr 2024, 10:24 AM", Title: "Order Placed", Detail: "Customer placed the order via UPI", Tone: "done"},
		{Time: "26 Apr 2024, 10:24 AM", Title: "Payment Captured", Detail: "₹5,998 via PhonePe (UPI)", Tone: "done"},
		{Time: "26 Apr 2024, 10:28 AM", Title: "Order Confirmed", Detail: "Sent to seller for fulfilment", Tone: "done"},
		{Time: "26 Apr 2024, 04:15 PM", Title: "Shipment Created", Detail: "AWB: SHPRK2504189921", Tone: "done"},
		{Time: "27 Apr 2024, 10:30 AM", Title: "Picked Up", Detail: "Shipment picked up by courier partner", Tone: "done"},
		{Time: "28 Apr 2024, 08:45 AM", Title: "In Transit", Detail: "Bengaluru sorting facility", Tone: "done"},
		{Time: "29 Apr 2024, 06:20 PM", Title: "In Transit", Detail: "Out for delivery", Tone: "done"},
		{Time: "30 Apr 2024, 09:15 AM", Title: "Delivered", Detail: "Package delivered to customer", Tone: "done"},
		{Time: "05 May 2024, 11:12 AM", Title: "Return Requested", Detail: "Customer raised return request", Tone: "alert"},
		{Time: "06 May 2024, 02:30 PM", Title: "Seller Response", Detail: "Requested more details", Tone: "warning"},
		{Time: "07 May 2024, 10:45 AM", Title: "Support Intervention", Detail: "Asked seller to approve valid return", Tone: "info"},
		{Time: "08 May 2024, 04:20 PM", Title: "Refund Initiated", Detail: "Refund of ₹5,998 to original payment method", Tone: "info"},
		{Time: "10 May 2024, 11:30 AM", Title: "Refund Completed", Detail: "Amount credited to customer", Tone: "done"},
	}
}
