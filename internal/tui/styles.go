package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary   = lipgloss.Color("#326CE5") // Kubernetes blue
	colorSecondary = lipgloss.Color("#EE0000") // OKD red
	colorSuccess   = lipgloss.Color("#04B575")
	colorWarning   = lipgloss.Color("#FFBD2E")
	colorError     = lipgloss.Color("#FF6B6B")
	colorMuted     = lipgloss.Color("#626262")
	colorHighlight = lipgloss.Color("#7D56F4")
	colorProdBg    = lipgloss.Color("#8B0000")
	colorWarnBg    = lipgloss.Color("#CC7700")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	contextStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	namespaceStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			PaddingLeft(1).
			PaddingRight(1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMuted).
			Underline(true)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	toastSuccessStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	toastErrorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	bannerWarnStyle = lipgloss.NewStyle().
			Background(colorWarnBg).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	bannerProdStyle = lipgloss.NewStyle().
			Background(colorProdBg).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorWarning).
			Padding(1, 2)

	errorScreenStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true).
				PaddingLeft(2).
				PaddingTop(1)

	liveStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)
)

func colorizeStatus(status string) string {
	switch status {
	case "Running", "Active":
		return lipgloss.NewStyle().Foreground(colorSuccess).Render(status)
	case "Succeeded", "Completed":
		return lipgloss.NewStyle().Foreground(colorSuccess).Render(status)
	case "Pending", "ContainerCreating", "Terminating":
		return lipgloss.NewStyle().Foreground(colorWarning).Render(status)
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff",
		"ErrImagePull", "OOMKilled", "Init:Error", "Init:CrashLoopBackOff":
		return lipgloss.NewStyle().Foreground(colorError).Render(status)
	default:
		return lipgloss.NewStyle().Foreground(colorMuted).Render(status)
	}
}
