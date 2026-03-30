package link

import "strings"

const commandPartsLimit = 2

func extractCommandArgs(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	parts := strings.SplitN(trimmed, " ", commandPartsLimit)
	if len(parts) < commandPartsLimit {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
