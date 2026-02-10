package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Compiled regexes for log line colorization.
var (
	reTimestamp  = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}[\.\d]*`)
	reLogLevel   = regexp.MustCompile(`\b(INFO|WARN|WARNING|ERROR|FATAL|SEVERE|DEBUG|TRACE)\b`)
	reHTTPMethod = regexp.MustCompile(`\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\b`)
	reHTTPStatus = regexp.MustCompile(`\b([2-5]\d{2})\b`)
)

type logState struct {
	podName       string
	containerName string
	content       string
	lines         []string
	offset        int
	previous      bool
	wrap          bool
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
	var logHeader string
	if ls.containerName != "" {
		logHeader = fmt.Sprintf("  Logs: %s/%s (%s) [%d lignes]", ls.podName, ls.containerName, mode, len(ls.lines))
	} else {
		logHeader = fmt.Sprintf("  Logs: %s (%s) [%d lignes]", ls.podName, mode, len(ls.lines))
	}
	b.WriteString(headerStyle.Render(logHeader))
	b.WriteString("\n")

	// Content
	usable := width - 2 // account for "  " prefix
	if usable < 1 {
		usable = 1
	}
	rendered := 0
	for i := ls.offset; i < len(ls.lines) && rendered < viewHeight; i++ {
		line := ls.lines[i]
		if ls.wrap {
			// Wrap: split logical line into visual lines
			for len(line) > 0 && rendered < viewHeight {
				chunk := line
				if len(chunk) > usable {
					chunk = line[:usable]
					line = line[usable:]
				} else {
					line = ""
				}
				b.WriteString("  ")
				b.WriteString(colorizeLine(chunk))
				b.WriteString("\n")
				rendered++
			}
		} else {
			// Truncate: crop with … indicator
			if len(line) > usable {
				line = line[:usable-1] + "…"
			}
			b.WriteString("  ")
			b.WriteString(colorizeLine(line))
			b.WriteString("\n")
			rendered++
		}
	}

	return b.String()
}

func colorizeLine(line string) string {
	if line == "" {
		return ""
	}

	// 1. Timestamps → gray
	line = reTimestamp.ReplaceAllStringFunc(line, func(m string) string {
		return lipgloss.NewStyle().Foreground(colorMuted).Render(m)
	})

	// 2. Log levels → colored bold
	line = reLogLevel.ReplaceAllStringFunc(line, func(m string) string {
		switch m {
		case "INFO":
			return lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(m)
		case "WARN", "WARNING":
			return lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render(m)
		case "ERROR", "FATAL", "SEVERE":
			return lipgloss.NewStyle().Foreground(colorError).Bold(true).Render(m)
		case "DEBUG", "TRACE":
			return lipgloss.NewStyle().Foreground(colorMuted).Render(m)
		}
		return m
	})

	// 3. HTTP methods → colored bold
	line = reHTTPMethod.ReplaceAllStringFunc(line, func(m string) string {
		switch m {
		case "GET":
			return lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(m)
		case "POST":
			return lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render(m)
		case "PUT", "PATCH":
			return lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(m)
		case "DELETE":
			return lipgloss.NewStyle().Foreground(colorError).Bold(true).Render(m)
		case "HEAD", "OPTIONS":
			return lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render(m)
		}
		return m
	})

	// 4. HTTP status codes → colored by range
	line = reHTTPStatus.ReplaceAllStringFunc(line, func(m string) string {
		switch m[0] {
		case '2':
			return lipgloss.NewStyle().Foreground(colorSuccess).Render(m)
		case '3':
			return lipgloss.NewStyle().Foreground(colorPrimary).Render(m)
		case '4':
			return lipgloss.NewStyle().Foreground(colorWarning).Render(m)
		case '5':
			return lipgloss.NewStyle().Foreground(colorError).Render(m)
		}
		return m
	})

	return line
}

func logHelpKeys(previous, wrap bool) string {
	wrapLabel := "w:wrap"
	if wrap {
		wrapLabel = "w:nowrap"
	}
	if previous {
		return fmt.Sprintf("pgup/pgdn:scroll  G:fin  %s  p:logs courants  esc:retour", wrapLabel)
	}
	return fmt.Sprintf("pgup/pgdn:scroll  G:fin  %s  p:logs précédents  esc:retour", wrapLabel)
}
