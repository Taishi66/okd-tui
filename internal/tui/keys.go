package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Enter    key.Binding
	Escape   key.Binding
	Filter   key.Binding
	Refresh  key.Binding
	Delete   key.Binding
	ScaleUp  key.Binding
	ScaleDn  key.Binding
	ScaleSet key.Binding
	Previous key.Binding
	Copy     key.Binding
	Help     key.Binding
	Tab1     key.Binding
	Tab2     key.Binding
	Tab3     key.Binding
	Tab4     key.Binding
	TabNext  key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "monter")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "descendre")),
	Top:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "début")),
	Bottom:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "fin")),
	PageUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("C-u", "page up")),
	PageDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("C-d", "page dn")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "sélectionner")),
	Escape:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "retour")),
	Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filtre")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "supprimer")),
	ScaleUp:  key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "scale up")),
	ScaleDn:  key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "scale down")),
	ScaleSet: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "scale")),
	Previous: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "logs précédents")),
	Copy:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copier nom")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "aide")),
	Tab1:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "projects")),
	Tab2:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "pods")),
	Tab3:     key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "deploys")),
	Tab4:     key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "events")),
	TabNext:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "vue suivante")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quitter")),
}
