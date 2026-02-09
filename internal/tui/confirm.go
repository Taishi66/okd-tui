package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type confirmMode int

const (
	confirmNone confirmMode = iota
	confirmSimple   // y/N prompt
	confirmProd     // type the full resource name
)

type confirmState struct {
	mode         confirmMode
	action       string // "Supprimer pod", "Scale deployment"
	resourceName string
	namespace    string
	isProd       bool
	input        textinput.Model
	callback     func() tea.Msg // action to execute on confirm
}

func newConfirmState() confirmState {
	ti := textinput.New()
	ti.CharLimit = 128
	ti.Width = 50
	return confirmState{
		mode:  confirmNone,
		input: ti,
	}
}

func (cs *confirmState) activate(action, resourceName, namespace string, isProd bool, callback func() tea.Msg) {
	cs.action = action
	cs.resourceName = resourceName
	cs.namespace = namespace
	cs.isProd = isProd
	cs.callback = callback
	if isProd {
		cs.mode = confirmProd
		cs.input.Placeholder = resourceName
		cs.input.SetValue("")
		cs.input.Focus()
	} else {
		cs.mode = confirmSimple
	}
}

func (cs *confirmState) reset() {
	cs.mode = confirmNone
	cs.action = ""
	cs.resourceName = ""
	cs.namespace = ""
	cs.isProd = false
	cs.input.SetValue("")
	cs.input.Blur()
	cs.callback = nil
}

func (cs *confirmState) isActive() bool {
	return cs.mode != confirmNone
}

func (cs *confirmState) update(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch cs.mode {
	case confirmSimple:
		switch msg.String() {
		case "y", "Y":
			cb := cs.callback
			cs.reset()
			if cb != nil {
				return cb, true
			}
			return nil, true
		case "n", "N", "esc":
			cs.reset()
			return nil, true
		}
		return nil, true // absorb all other keys

	case confirmProd:
		switch msg.String() {
		case "esc":
			cs.reset()
			return nil, true
		case "enter":
			if strings.TrimSpace(cs.input.Value()) == cs.resourceName {
				cb := cs.callback
				cs.reset()
				if cb != nil {
					return cb, true
				}
			}
			return nil, true // wrong name, stay
		default:
			var cmd tea.Cmd
			cs.input, cmd = cs.input.Update(msg)
			return cmd, true
		}
	}
	return nil, false
}

func (cs *confirmState) view(width int) string {
	switch cs.mode {
	case confirmSimple:
		prompt := fmt.Sprintf("  %s %s ? [y/N] ", cs.action, cs.resourceName)
		return "\n" + prompt
	case confirmProd:
		box := fmt.Sprintf(
			"  NAMESPACE PRODUCTION\n\n"+
				"  Action : %s\n"+
				"  Cible  : %s\n"+
				"  NS     : %s\n\n"+
				"  Tapez \"%s\" pour confirmer :\n"+
				"  > %s\n\n"+
				"  [Esc] Annuler",
			cs.action, cs.resourceName, cs.namespace,
			cs.resourceName, cs.input.View(),
		)
		return "\n" + bannerProdStyle.Width(min(width-4, 60)).Render(box) + "\n"
	}
	return ""
}
