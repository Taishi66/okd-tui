package tui

import (
	"fmt"
	"strings"
)

type yamlViewState struct {
	resourceName string
	resourceType string
	content      string
	lines        []string
	offset       int
}

func (ys *yamlViewState) setContent(content string) {
	ys.content = content
	ys.lines = strings.Split(content, "\n")
	ys.offset = 0
}

func (ys *yamlViewState) scrollDown(amount, viewHeight int) {
	maxOffset := len(ys.lines) - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	ys.offset = min(ys.offset+amount, maxOffset)
}

func (ys *yamlViewState) scrollUp(amount int) {
	ys.offset = max(ys.offset-amount, 0)
}

func renderYAMLView(ys *yamlViewState, width, viewHeight int) string {
	if ys.content == "" {
		return "  Pas de YAML disponible\n"
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("  YAML: %s/%s [%d lignes]", ys.resourceType, ys.resourceName, len(ys.lines))
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	// Content
	end := min(ys.offset+viewHeight, len(ys.lines))
	for i := ys.offset; i < end; i++ {
		line := ys.lines[i]
		if len(line) > width-2 {
			line = line[:width-2]
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func yamlHelpKeys() string {
	return "pgup/pgdn:scroll  G:fin  esc:retour"
}
