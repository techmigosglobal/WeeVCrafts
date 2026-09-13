package admin

import (
	"net/url"
	"strings"
)

func navClass(key, current string) string {
	if key == current {
		return "admin-nav-link active"
	}
	return "admin-nav-link"
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
