package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jclamy/okd-tui/internal/domain"
)

var (
	eventWarningStyle = lipgloss.NewStyle().Foreground(colorWarning)
	eventNormalStyle  = lipgloss.NewStyle().Foreground(colorSuccess)
)

func renderEventList(events []domain.EventInfo, cursor, width, maxVisible int) string {
	if len(events) == 0 {
		return "  Aucun event dans ce namespace\n"
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-10s %-22s %-28s %-40s %-8s %s", "TYPE", "REASON", "OBJECT", "MESSAGE", "AGE", "COUNT")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}

	for i := start; i < len(events) && i < start+maxVisible; i++ {
		e := events[i]

		typeStr := e.Type
		if e.Type == "Warning" {
			typeStr = eventWarningStyle.Render(e.Type)
		} else {
			typeStr = eventNormalStyle.Render(e.Type)
		}

		line := fmt.Sprintf("  %-10s %-22s %-28s %-40s %-8s %d",
			typeStr,
			truncate(e.Reason, 21),
			truncate(e.Object, 27),
			truncate(e.Message, 39),
			e.Age,
			e.Count)

		if i == cursor {
			b.WriteString(selectedStyle.Width(width).Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func eventHelpKeys() string {
	return "j/k:nav  g/G:début/fin  t:tri  /:filtre  r:refresh  q:quit"
}
