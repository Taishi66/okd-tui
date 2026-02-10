package tui

import (
	"fmt"
	"strings"
)

func renderContainerSelector(podName string, choices []string, cursor int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  Container pour %s :\n\n", podName))
	for i, name := range choices {
		if i == cursor {
			b.WriteString(fmt.Sprintf("  > %s\n", selectedStyle.Render(name)))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", name))
		}
	}
	b.WriteString("\n  j/k:nav  enter:sélectionner  esc:annuler\n")
	return b.String()
}
