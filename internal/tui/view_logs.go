package tui

import (
	"fmt"
	"strings"
)

type logState struct {
	podName   string
	content   string
	lines     []string
	offset    int
	previous  bool
}

func (ls *logState) setContent(content string) {
	ls.content = content
	ls.lines = strings.Split(content, "\n")
	// Jump to bottom
	ls.offset = 0
}

func (ls *logState) scrollDown(amount, viewHeight int) {
	maxOffset := len(ls.lines) - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	ls.offset = min(ls.offset+amount, maxOffset)
}

func (ls *logState) scrollUp(amount int) {
	ls.offset = max(ls.offset-amount, 0)
}

func (ls *logState) jumpToBottom(viewHeight int) {
	maxOffset := len(ls.lines) - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	ls.offset = maxOffset
}

func renderLogs(ls *logState, width, viewHeight int) string {
	if ls.content == "" {
		return "  Pas de logs disponibles\n"
	}

	var b strings.Builder

	// Header
	mode := "current"
	if ls.previous {
		mode = "previous"
	}
	logHeader := fmt.Sprintf("  Logs: %s (%s) [%d lignes]", ls.podName, mode, len(ls.lines))
	b.WriteString(headerStyle.Render(logHeader))
	b.WriteString("\n")

	// Content
	end := min(ls.offset+viewHeight, len(ls.lines))
	for i := ls.offset; i < end; i++ {
		line := ls.lines[i]
		if len(line) > width-2 {
			line = line[:width-2]
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func logHelpKeys(previous bool) string {
	if previous {
		return "pgup/pgdn:scroll  G:fin  p:logs courants  esc:retour"
	}
	return "pgup/pgdn:scroll  G:fin  p:logs précédents  esc:retour"
}
