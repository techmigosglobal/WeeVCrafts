package supportportal

import "strings"

func supportNavClass(key, current string) string {
	if key == current {
		return "support-nav-link active"
	}
	return "support-nav-link"
}

func supportPath(active string) string {
	if active == "dashboard" || active == "" {
		return "/support-portal"
	}
	return "/support-portal/" + active
}

func supportToneClass(tone string) string {
	if tone == "" {
		return "red"
	}
	return tone
}

func supportStatusClass(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "resolved"), strings.Contains(value, "active"), strings.Contains(value, "online"), strings.Contains(value, "within"):
		return "success"
	case strings.Contains(value, "urgent"), strings.Contains(value, "high"), strings.Contains(value, "escalated"), strings.Contains(value, "breached"):
		return "danger"
	case strings.Contains(value, "pending"), strings.Contains(value, "medium"), strings.Contains(value, "away"), strings.Contains(value, "open"):
		return "warning"
	default:
		return "neutral"
	}
}

func supportPriorityClass(value string) string {
	switch strings.ToLower(value) {
	case "high":
		return "priority-high"
	case "medium":
		return "priority-medium"
	default:
		return "priority-low"
	}
}

func supportArrowIcon(positive bool) string {
	if positive {
		return "arrow_upward"
	}
	return "arrow_downward"
}

func supportTrendClass(positive bool) string {
	if positive {
		return "trend-good"
	}
	return "trend-bad"
}

func supportStatIcon(icon string) string {
	if icon == "" {
		return "support_agent"
	}
	return icon
}

func supportIntroImage(active string) string {
	if active == "customers" || active == "orders" {
		return "/assets/images/customer/maker-mithila.png"
	}
	return "/assets/images/customer/hero-studio.png"
}

func supportDefaultIntro(active string) bool {
	return active != "customers" && active != "orders"
}

func supportCaseRowClass(index int) string {
	if index == 0 {
		return "support-case-row selected"
	}
	return "support-case-row"
}

func supportDirectoryRowClass(index int) string {
	if index == 0 {
		return "directory-row selected"
	}
	return "directory-row"
}

func supportAvatar(initials string) string {
	if initials == "" {
		return "AR"
	}
	return initials
}

func supportConversationClass(tone string) string {
	if tone == "" {
		return "support-conversation"
	}
	return "support-conversation " + tone
}

func supportTimelineClass(tone string) string {
	if tone == "" {
		return "support-timeline-entry"
	}
	return "support-timeline-entry " + tone
}

func supportTypeClass(value string) string {
	if strings.EqualFold(value, "seller") {
		return "directory-type seller-type"
	}
	return "directory-type customer-type"
}
