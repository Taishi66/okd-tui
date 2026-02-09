package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmState_InitiallyInactive(t *testing.T) {
	cs := newConfirmState()
	if cs.isActive() {
		t.Error("new confirmState should be inactive")
	}
}

func TestConfirmState_ActivateSimple(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete pod", "my-pod", "dev", false, func() tea.Msg {
		return nil
	})

	if !cs.isActive() {
		t.Error("should be active after activate()")
	}
	if cs.mode != confirmSimple {
		t.Errorf("mode = %v, want confirmSimple", cs.mode)
	}

	// Confirm with 'y'
	cmd, handled := cs.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled {
		t.Error("should handle 'y' key")
	}
	if cmd == nil {
		t.Error("cmd should not be nil after confirm")
	}
	if cs.isActive() {
		t.Error("should be inactive after confirm")
	}
}

func TestConfirmState_CancelSimple(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete pod", "my-pod", "dev", false, func() tea.Msg { return nil })

	// Cancel with 'n'
	_, handled := cs.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !handled {
		t.Error("should handle 'n' key")
	}
	if cs.isActive() {
		t.Error("should be inactive after cancel")
	}
}

func TestConfirmState_CancelWithEsc(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete pod", "my-pod", "dev", false, func() tea.Msg { return nil })

	_, handled := cs.update(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("should handle esc key")
	}
	if cs.isActive() {
		t.Error("should be inactive after esc")
	}
}

func TestConfirmState_AbsorbsOtherKeys(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete pod", "my-pod", "dev", false, func() tea.Msg { return nil })

	// Random key should be absorbed but not trigger action
	_, handled := cs.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !handled {
		t.Error("should absorb all keys in simple mode")
	}
	if !cs.isActive() {
		t.Error("should still be active after random key")
	}
}

func TestConfirmState_ProdMode(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete pod", "my-pod-abc123", "production", true, func() tea.Msg {
		return nil
	})

	if cs.mode != confirmProd {
		t.Errorf("mode = %v, want confirmProd", cs.mode)
	}

	// Wrong name should not trigger
	cs.input.SetValue("wrong-name")
	cmd, handled := cs.update(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("should handle enter")
	}
	if cmd != nil {
		t.Error("wrong name should not produce cmd")
	}
	if !cs.isActive() {
		t.Error("should still be active after wrong name")
	}

	// Correct name should trigger
	cs.input.SetValue("my-pod-abc123")
	cmd, handled = cs.update(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("should handle enter")
	}
	if cmd == nil {
		t.Error("correct name should produce cmd")
	}
	if cs.isActive() {
		t.Error("should be inactive after correct name")
	}
}

func TestConfirmState_ProdEscCancels(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete", "pod-x", "production", true, func() tea.Msg { return nil })

	_, handled := cs.update(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("esc should be handled")
	}
	if cs.isActive() {
		t.Error("esc should cancel prod confirm")
	}
}

func TestConfirmState_Reset(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Delete", "pod-x", "prod", true, func() tea.Msg { return nil })
	cs.reset()

	if cs.isActive() {
		t.Error("should be inactive after reset")
	}
	if cs.action != "" || cs.resourceName != "" || cs.namespace != "" {
		t.Error("fields should be cleared after reset")
	}
	if cs.callback != nil {
		t.Error("callback should be nil after reset")
	}
}

func TestConfirmState_ViewSimple(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Supprimer pod", "my-pod", "dev", false, nil)

	view := cs.view(80)
	if view == "" {
		t.Error("view should not be empty for simple mode")
	}
	// Should contain the resource name
	if !containsStr(view, "my-pod") {
		t.Error("view should contain resource name")
	}
	// Should contain y/N
	if !containsStr(view, "[y/N]") {
		t.Error("view should contain [y/N]")
	}
}

func TestConfirmState_ViewProd(t *testing.T) {
	cs := newConfirmState()
	cs.activate("Supprimer pod", "my-pod", "production", true, nil)

	view := cs.view(80)
	if view == "" {
		t.Error("view should not be empty for prod mode")
	}
	if !containsStr(view, "PRODUCTION") {
		t.Error("view should contain PRODUCTION")
	}
	if !containsStr(view, "my-pod") {
		t.Error("view should contain resource name")
	}
}

func TestConfirmState_ViewInactive(t *testing.T) {
	cs := newConfirmState()
	view := cs.view(80)
	if view != "" {
		t.Error("inactive view should be empty")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
