package telegram

import (
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

func formatLinkUpdateMessage(u linkdto.LinkUpdate) string {
	var b strings.Builder
	b.WriteString(kindLabel(u.EventKind))
	b.WriteString("\n")
	b.WriteString(u.Title)
	b.WriteString("\n\n")
	b.WriteString("Ссылка: ")
	b.WriteString(u.URL)
	b.WriteString("\nАвтор: ")
	b.WriteString(u.Author)
	b.WriteString("\nСоздано: ")
	b.WriteString(u.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	if strings.TrimSpace(u.Preview) != "" {
		b.WriteString("\n\nПревью:\n")
		b.WriteString(u.Preview)
	}
	if strings.TrimSpace(u.Description) != "" {
		b.WriteString("\n\n")
		b.WriteString(strings.TrimSpace(u.Description))
	}
	return b.String()
}

func kindLabel(kind string) string {
	switch kind {
	case linkdto.EventKindGitHubIssue:
		return "GitHub: новый Issue"
	case linkdto.EventKindGitHubPullRequest:
		return "GitHub: новый Pull Request"
	case linkdto.EventKindStackAnswer:
		return "Stack Overflow: новый ответ"
	case linkdto.EventKindStackComment:
		return "Stack Overflow: новый комментарий"
	default:
		return fmt.Sprintf("Обновление (%s)", kind)
	}
}

func formatFailureReportMessage(urls []string, detail string) string {
	var b strings.Builder
	b.WriteString("Не удалось проверить следующие ссылки:\n")
	for _, u := range urls {
		b.WriteString("• ")
		b.WriteString(u)
		b.WriteString("\n")
	}
	if strings.TrimSpace(detail) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(detail))
	}
	return strings.TrimRight(b.String(), "\n")
}
