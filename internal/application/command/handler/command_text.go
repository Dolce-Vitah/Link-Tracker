package handler

import "strings"

func extractCommandArgs(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) < 2 {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
