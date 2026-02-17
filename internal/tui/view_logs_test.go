package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taishi66/okd-tui/internal/domain"
)

func TestLogState_SetContent(t *testing.T) {
	ls := logState{podName: "my-pod"}
	ls.setContent("line1\nline2\nline3")

	if len(ls.lines) != 3 {
		t.Errorf("lines count = %d, want 3", len(ls.lines))
	}
	if ls.offset != 0 {
		t.Errorf("offset after setContent = %d, want 0", ls.offset)
	}
}

func TestLogState_SetContentEmpty(t *testing.T) {
	ls := logState{}
	ls.setContent("")

	if len(ls.lines) != 1 {
		// "" split by \n gives [""], which is 1 element
		t.Errorf("empty content: lines count = %d, want 1", len(ls.lines))
	}
}

func TestLogState_ScrollDown(t *testing.T) {
	ls := logState{podName: "test"}
	// 50 lines of content
	content := ""
	for i := 0; i < 50; i++ {
		if i > 0 {
			content += "\n"
		}
		content += "line"
	}
	ls.setContent(content)

	viewHeight := 20

	// Scroll down
	ls.scrollDown(10, viewHeight)
	if ls.offset != 10 {
		t.Errorf("offset after scrollDown(10) = %d, want 10", ls.offset)
	}

	// Scroll down beyond max
	ls.scrollDown(100, viewHeight)
	maxOffset := len(ls.lines) - viewHeight
	if ls.offset != maxOffset {
		t.Errorf("offset after overscroll = %d, want %d", ls.offset, maxOffset)
	}

	// Scroll down with viewHeight > lines should clamp to 0
	ls.offset = 0
	ls.setContent("one\ntwo\nthree")
	ls.scrollDown(10, 100) // viewHeight(100) > lines(3)
	if ls.offset != 0 {
		t.Errorf("offset when viewHeight > lines = %d, want 0", ls.offset)
	}
}

func TestLogState_ScrollUp(t *testing.T) {
	ls := logState{offset: 20}

	ls.scrollUp(5)
	if ls.offset != 15 {
		t.Errorf("offset after scrollUp(5) = %d, want 15", ls.offset)
	}

	// Scroll up beyond 0
	ls.scrollUp(100)
	if ls.offset != 0 {
		t.Errorf("offset after overscroll up = %d, want 0", ls.offset)
	}
}

func TestLogState_JumpToBottom(t *testing.T) {
	ls := logState{podName: "test"}
	content := ""
	for i := 0; i < 100; i++ {
		if i > 0 {
			content += "\n"
		}
		content += "line"
	}
	ls.setContent(content)

	ls.jumpToBottom(30)
	expected := len(ls.lines) - 30
	if ls.offset != expected {
		t.Errorf("offset after jumpToBottom = %d, want %d", ls.offset, expected)
	}

	// Jump to bottom when content smaller than view
	ls.setContent("short")
	ls.jumpToBottom(30)
	if ls.offset != 0 {
		t.Errorf("offset jumpToBottom (short content) = %d, want 0", ls.offset)
	}
}

func TestRenderLogs_EmptyContent(t *testing.T) {
	ls := logState{podName: "test", content: ""}
	output := renderLogs(&ls, 80, 20)
	if output == "" {
		t.Error("renderLogs should return something even with empty content")
	}
}

func TestRenderLogs_LineTruncation(t *testing.T) {
	ls := logState{podName: "test"}
	longLine := ""
	for i := 0; i < 200; i++ {
		longLine += "x"
	}
	ls.setContent(longLine)

	output := renderLogs(&ls, 80, 20)
	// Each rendered line should be <= width
	if len(output) == 0 {
		t.Error("renderLogs produced empty output")
	}
}

// --- wrap key integration test ---

func TestWrapKey_TogglesWrap(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewLogs
	m.logState = logState{podName: "test", content: "line", lines: []string{"line"}}
	m.width = 120
	m.height = 30

	if m.logState.wrap {
		t.Error("wrap should be off by default")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	um := updated.(Model)
	if !um.logState.wrap {
		t.Error("wrap should be on after pressing w")
	}

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	um = updated.(Model)
	if um.logState.wrap {
		t.Error("wrap should be off after pressing w again")
	}
}

// --- wrap/truncate tests ---

func TestRenderLogs_TruncateWithEllipsis(t *testing.T) {
	ls := logState{podName: "test"}
	ls.setContent("abcdefghijklmnopqrstuvwxyz0123456789")
	// width=20 → usable=18 → truncate to 17 + "…"
	output := renderLogs(&ls, 20, 10)
	if !strings.Contains(output, "…") {
		t.Error("truncated line should end with …")
	}
}

func TestRenderLogs_WrapMode(t *testing.T) {
	ls := logState{podName: "test", wrap: true}
	// Line of 30 chars, width=20 → usable=18 → should wrap into 2 visual lines
	ls.setContent("123456789012345678901234567890")
	output := renderLogs(&ls, 20, 10)
	// Count non-header content lines: should be 2 (18 + 12)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	// lines[0] = header, lines[1..] = content
	contentLines := lines[1:]
	if len(contentLines) < 2 {
		t.Errorf("wrap mode: expected >= 2 visual lines, got %d", len(contentLines))
	}
}

func TestRenderLogs_WrapMode_ShortLine(t *testing.T) {
	ls := logState{podName: "test", wrap: true}
	ls.setContent("short")
	output := renderLogs(&ls, 80, 10)
	if strings.Contains(output, "…") {
		t.Error("short line in wrap mode should not have …")
	}
	if !strings.Contains(output, "short") {
		t.Error("should contain the full short line")
	}
}

func TestRenderLogs_TruncateMode_ShortLine(t *testing.T) {
	ls := logState{podName: "test"}
	ls.setContent("short line")
	output := renderLogs(&ls, 80, 10)
	if strings.Contains(output, "…") {
		t.Error("short line should not have …")
	}
	if !strings.Contains(output, "short line") {
		t.Error("should contain the full short line")
	}
}

func TestRenderLogs_WrapRespectsViewHeight(t *testing.T) {
	ls := logState{podName: "test", wrap: true}
	// 1 very long line that would wrap into 5 visual lines at width 12 (usable=10)
	ls.setContent("12345678901234567890123456789012345678901234567890")
	output := renderLogs(&ls, 12, 3)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	// header(1) + max 3 content lines = 4
	if len(lines) > 4 {
		t.Errorf("should respect viewHeight, got %d lines", len(lines))
	}
}

func TestLogHelpKeys_WrapLabel(t *testing.T) {
	help := logHelpKeys(false, false)
	if !strings.Contains(help, "w:wrap") {
		t.Errorf("should show w:wrap, got %q", help)
	}
	help = logHelpKeys(false, true)
	if !strings.Contains(help, "w:nowrap") {
		t.Errorf("should show w:nowrap when wrap is on, got %q", help)
	}
}

// --- colorizeLine tests ---

func TestColorizeLine_EmptyLine(t *testing.T) {
	result := colorizeLine("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestColorizeLine_PlainText(t *testing.T) {
	line := "just some plain text without patterns"
	result := colorizeLine(line)
	if result != line {
		t.Errorf("plain text should be unchanged, got %q", result)
	}
}

func TestColorizeLine_HTTPStatus_2xx(t *testing.T) {
	result := colorizeLine("Outcome: Success(200 OK)")
	green := lipgloss.NewStyle().Foreground(colorSuccess).Render("200")
	if !strings.Contains(result, green) {
		t.Errorf("200 should be green, got %q", result)
	}
}

func TestColorizeLine_HTTPStatus_3xx(t *testing.T) {
	result := colorizeLine("301 Moved Permanently")
	blue := lipgloss.NewStyle().Foreground(colorPrimary).Render("301")
	if !strings.Contains(result, blue) {
		t.Errorf("301 should be blue, got %q", result)
	}
}

func TestColorizeLine_HTTPStatus_4xx(t *testing.T) {
	result := colorizeLine("404 Not Found")
	yellow := lipgloss.NewStyle().Foreground(colorWarning).Render("404")
	if !strings.Contains(result, yellow) {
		t.Errorf("404 should be yellow, got %q", result)
	}
}

func TestColorizeLine_HTTPStatus_5xx(t *testing.T) {
	result := colorizeLine("500 Internal Server Error")
	red := lipgloss.NewStyle().Foreground(colorError).Render("500")
	if !strings.Contains(result, red) {
		t.Errorf("500 should be red, got %q", result)
	}
}

func TestColorizeLine_LogLevel_INFO(t *testing.T) {
	result := colorizeLine("INFO some message")
	blue := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("INFO")
	if !strings.Contains(result, blue) {
		t.Errorf("INFO should be blue bold, got %q", result)
	}
}

func TestColorizeLine_LogLevel_ERROR(t *testing.T) {
	result := colorizeLine("ERROR something failed")
	red := lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("ERROR")
	if !strings.Contains(result, red) {
		t.Errorf("ERROR should be red bold, got %q", result)
	}
}

func TestColorizeLine_LogLevel_WARN(t *testing.T) {
	result := colorizeLine("WARN slow query")
	yellow := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("WARN")
	if !strings.Contains(result, yellow) {
		t.Errorf("WARN should be yellow bold, got %q", result)
	}
}

func TestColorizeLine_LogLevel_DEBUG(t *testing.T) {
	result := colorizeLine("DEBUG verbose")
	gray := lipgloss.NewStyle().Foreground(colorMuted).Render("DEBUG")
	if !strings.Contains(result, gray) {
		t.Errorf("DEBUG should be gray, got %q", result)
	}
}

func TestColorizeLine_HTTPMethod_GET(t *testing.T) {
	result := colorizeLine("GET /api/test")
	green := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("GET")
	if !strings.Contains(result, green) {
		t.Errorf("GET should be green bold, got %q", result)
	}
}

func TestColorizeLine_HTTPMethod_DELETE(t *testing.T) {
	result := colorizeLine("DELETE /api/staff/17738")
	red := lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("DELETE")
	if !strings.Contains(result, red) {
		t.Errorf("DELETE should be red bold, got %q", result)
	}
}

func TestColorizeLine_HTTPMethod_POST(t *testing.T) {
	result := colorizeLine("POST /api/data")
	yellow := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("POST")
	if !strings.Contains(result, yellow) {
		t.Errorf("POST should be yellow bold, got %q", result)
	}
}

func TestColorizeLine_Timestamp(t *testing.T) {
	result := colorizeLine("2026-02-10 10:40:17.098 INFO message")
	gray := lipgloss.NewStyle().Foreground(colorMuted).Render("2026-02-10 10:40:17.098")
	if !strings.Contains(result, gray) {
		t.Errorf("timestamp should be gray, got %q", result)
	}
}

func TestColorizeLine_FullLine(t *testing.T) {
	line := "2026-02-10 10:40:17.098 INFO Outcome: Success(200 OK)"
	result := colorizeLine(line)

	if !strings.Contains(result, lipgloss.NewStyle().Foreground(colorMuted).Render("2026-02-10 10:40:17.098")) {
		t.Error("full line: timestamp not colorized")
	}
	if !strings.Contains(result, lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("INFO")) {
		t.Error("full line: INFO not colorized")
	}
	if !strings.Contains(result, lipgloss.NewStyle().Foreground(colorSuccess).Render("200")) {
		t.Error("full line: 200 not colorized")
	}
}
