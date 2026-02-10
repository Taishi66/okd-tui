package tui

import (
	"fmt"
	"strings"

	"github.com/jclamy/okd-tui/internal/domain"
)

func renderProjectList(namespaces []domain.NamespaceInfo, cursor, width, maxVisible int, activeNS string) string {
	if len(namespaces) == 0 {
		return "  Aucun projet accessible\n"
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-40s %-12s %s", "NAME", "STATUS", "AGE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}

	for i := start; i < len(namespaces) && i < start+maxVisible; i++ {
		ns := namespaces[i]
		marker := "  "
		if ns.Name == activeNS {
			marker = "> "
		}
		line := fmt.Sprintf("%s%-40s %-12s %s",
			marker, truncate(ns.Name, 39), colorizeStatus(ns.Status), ns.Age)

		if i == cursor {
			b.WriteString(selectedStyle.Width(width).Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func projectHelpKeys() string {
	return "j/k:nav  g/G:début/fin  enter:sélectionner  /:filtre  r:refresh  q:quit"
}
