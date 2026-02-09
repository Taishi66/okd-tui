package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type toastLevel int

const (
	toastInfo toastLevel = iota
	toastSuccess
	toastError
)

type toast struct {
	message string
	level   toastLevel
	expires time.Time
}

type toastExpiredMsg struct{}

func (t toast) isActive() bool {
	return t.message != "" && time.Now().Before(t.expires)
}

func (t toast) render() string {
	if !t.isActive() {
		return ""
	}
	switch t.level {
	case toastSuccess:
		return toastSuccessStyle.Render(t.message)
	case toastError:
		return toastErrorStyle.Render(t.message)
	default:
		return t.message
	}
}

func newToast(msg string, level toastLevel) toast {
	return toast{
		message: msg,
		level:   level,
		expires: time.Now().Add(5 * time.Second),
	}
}

func scheduleToastClear() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return toastExpiredMsg{}
	})
}
