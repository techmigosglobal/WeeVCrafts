package admin

import (
	"net/url"
	"strings"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

func firstAdminOrder(orders []viewmodels.AdminOrder) viewmodels.AdminOrder {
	if len(orders) == 0 {
		return viewmodels.AdminOrder{}
	}
	return orders[0]
}

func firstAdminReturn(returns []viewmodels.AdminReturn) viewmodels.AdminReturn {
	if len(returns) == 0 {
		return viewmodels.AdminReturn{}
	}
	return returns[0]
}

func navClass(key, current string) string {
	if key == current {
		return "admin-nav-link active"
	}
	return "admin-nav-link"
}

func ariaCurrent(active bool) string {
	if active {
		return "page"
	}
	return ""
}

func adminPath(active string) string {
	if active == "dashboard" || active == "" {
		return "/admin"
	}
	return "/admin/" + active
}

func trendClass(positive bool) string {
	if positive {
		return "trend positive"
	}
	return "trend negative"
}

func trendIcon(positive bool) string {
	if positive {
		return "arrow_upward"
	}
	return "arrow_downward"
}

func statusClass(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "approved"), strings.Contains(value, "active"), strings.Contains(value, "completed"), strings.Contains(value, "delivered"), strings.Contains(value, "success"), strings.Contains(value, "in stock"), strings.Contains(value, "resolved"), strings.Contains(value, "paid"):
		return "success"
	case strings.Contains(value, "rejected"), strings.Contains(value, "cancel"), strings.Contains(value, "suspend"), strings.Contains(value, "dispute"), strings.Contains(value, "danger"), strings.Contains(value, "rto"), strings.Contains(value, "failed"):
		return "danger"
	case strings.Contains(value, "review"), strings.Contains(value, "processing"), strings.Contains(value, "shipped"), strings.Contains(value, "progress"), strings.Contains(value, "export"), strings.Contains(value, "changed"), strings.Contains(value, "under"):
		return "review"
	default:
		return "pending"
	}
}

func stockClass(status string) string {
	if status == "Low Stock" {
		return "stock low"
	}
	return "stock good"
}

func typeClass(value string) string {
	if strings.EqualFold(value, "Export") {
		return "review"
	}
	return "success"
}

func priorityClass(value string) string {
	if strings.EqualFold(value, "High") {
		return "danger"
	}
	return "pending"
}

func returnClass(status string) string {
	if status == "Dispute" {
		return "selected"
	}
	return ""
}

func auditClass(action string) string {
	if strings.EqualFold(action, "Approved") || strings.EqualFold(action, "Resolved") {
		return "success"
	}
	if strings.EqualFold(action, "Suspended") {
		return "danger"
	}
	return "review"
}

func actionNotice(prefix, value string) string {
	return url.QueryEscape(prefix + value)
}

func workspaceTabClass(active bool) string {
	if active {
		return "active"
	}
	return ""
}

func adminParentWorkflow(workspace string) string {
	switch workspace {
	case "seller-approvals", "seller-details", "seller-suspensions":
		return "Sellers"
	case "product-details", "category-management", "brands":
		return "Products & Categories"
	case "coupons":
		return "Marketing & Content"
	case "commission-rules", "settlements", "finance-reconciliation", "payments", "failed-payments":
		return "Finance"
	case "refunds", "disputes":
		return "Returns & Refunds"
	case "audit-logs", "system-health", "search-indexing-health", "worker-queue-health", "settings":
		return "Analytics & Settings"
	case "security-events", "roles":
		return "Security & Roles"
	default:
		return "Admin Dashboard"
	}
}

func adminParentPath(workspace string) string {
	switch workspace {
	case "seller-approvals", "seller-details", "seller-suspensions":
		return "/admin/sellers"
	case "product-details", "category-management", "brands":
		return "/admin/products"
	case "coupons":
		return "/admin/marketing"
	case "commission-rules", "settlements", "finance-reconciliation", "payments", "failed-payments":
		return "/admin/finance"
	case "refunds", "disputes":
		return "/admin/returns"
	case "audit-logs", "system-health", "search-indexing-health", "worker-queue-health", "settings":
		return "/admin/analytics"
	case "security-events", "roles":
		return "/admin/security"
	default:
		return "/admin"
	}
}
