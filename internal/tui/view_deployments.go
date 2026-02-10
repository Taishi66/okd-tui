package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jclamy/okd-tui/internal/domain"
)

func renderDeploymentList(deps []domain.DeploymentInfo, cursor, width, maxVisible int) string {
	if len(deps) == 0 {
		return "  Aucun deployment dans ce namespace\n"
	}

	var b strings.Builder

	if width >= 120 {
		header := fmt.Sprintf("  %-38s %-10s %-10s %-8s %s", "NAME", "READY", "AVAIL", "AGE", "IMAGE")
		b.WriteString(headerStyle.Render(header))
	} else if width >= 80 {
		header := fmt.Sprintf("  %-35s %-10s %-10s %s", "NAME", "READY", "AVAIL", "AGE")
		b.WriteString(headerStyle.Render(header))
	} else {
		header := fmt.Sprintf("  %-30s %-10s %s", "NAME", "READY", "AGE")
		b.WriteString(headerStyle.Render(header))
	}
	b.WriteString("\n")

	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}

	for i := start; i < len(deps) && i < start+maxVisible; i++ {
		d := deps[i]
		readyColor := colorizeReady(d.Ready)

		var line string
		if width >= 120 {
			line = fmt.Sprintf("  %-38s %-10s %-10d %-8s %s",
				truncate(d.Name, 37), readyColor, d.Available, d.Age,
				truncate(d.Image, width-75))
		} else if width >= 80 {
			line = fmt.Sprintf("  %-35s %-10s %-10d %s",
				truncate(d.Name, 34), readyColor, d.Available, d.Age)
		} else {
			line = fmt.Sprintf("  %-30s %-10s %s",
				truncate(d.Name, 29), readyColor, d.Age)
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

func colorizeReady(ready string) string {
	var readyN, totalN int
	fmt.Sscanf(ready, "%d/%d", &readyN, &totalN)
	if totalN > 0 && readyN == totalN {
		return lipgloss.NewStyle().Foreground(colorSuccess).Render(ready)
	}
	if readyN == 0 {
		return lipgloss.NewStyle().Foreground(colorError).Render(ready)
	}
	return lipgloss.NewStyle().Foreground(colorWarning).Render(ready)
}

func deploymentHelpKeys() string {
	return "j/k:nav  +/-:scale  s:scale set  t:tri  /:filtre  r:refresh  q:quit"
}
