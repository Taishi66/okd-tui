package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/config"
	"github.com/jclamy/okd-tui/internal/domain"
	"github.com/jclamy/okd-tui/internal/k8s"
	"github.com/jclamy/okd-tui/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("okd-tui %s\n", version)
		os.Exit(0)
	}

	cfg, _ := config.LoadConfig()

	// ClientFactory wraps k8s.NewClient to return the domain interface.
	factory := func() (domain.KubeGateway, error) {
		return k8s.NewClient()
	}

	client, err := k8s.NewClient()
	if err != nil {
		// Client creation failed -- launch TUI in error mode
		m := tui.NewModelWithError(err, factory)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	m := tui.NewModel(client, factory, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
