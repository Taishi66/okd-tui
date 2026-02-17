package tui

import (
	"fmt"
	"strings"

	"github.com/Taishi66/okd-tui/internal/domain"
)

func renderPodList(pods []domain.PodInfo, cursor, width, maxVisible int) string {
	if len(pods) == 0 {
		return "  Aucun pod dans ce namespace\n"
	}

	var b strings.Builder

	// Responsive columns
	if width >= 100 {
		header := fmt.Sprintf("  %-42s %-18s %-7s %-10s %s", "NAME", "STATUS", "READY", "RESTARTS", "AGE")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
	} else {
		header := fmt.Sprintf("  %-35s %-18s %s", "NAME", "STATUS", "READY")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
	}

	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}

	for i := start; i < len(pods) && i < start+maxVisible; i++ {
		p := pods[i]
		var line string
		if width >= 100 {
			line = fmt.Sprintf("  %-42s %-18s %-7s %-10d %s",
				truncate(p.Name, 41),
				colorizeStatus(p.Status),
				p.Ready,
				p.Restarts,
				p.Age)
		} else {
			line = fmt.Sprintf("  %-35s %-18s %s",
				truncate(p.Name, 34),
				colorizeStatus(p.Status),
				p.Ready)
		}

		if i == cursor {
			b.WriteString(selectedStyle.Width(width).Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func podHelpKeys() string {
	return "j/k:nav  g/G:début/fin  enter:logs  s:shell  d:suppr  y:yaml  t:tri  /:filtre  r:refresh  q:quit"
}
