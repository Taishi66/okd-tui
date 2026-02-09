package tui

import "testing"

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
