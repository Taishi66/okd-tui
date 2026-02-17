package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taishi66/okd-tui/internal/config"
	"github.com/Taishi66/okd-tui/internal/domain"
)

func TestShellKey_SingleContainer_CallsBuildExec(t *testing.T) {
	cmd := exec.Command("echo", "test")
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "web-1", Status: "Running", Containers: []domain.ContainerInfo{{Name: "main"}}},
		},
		ExecCmd: cmd,
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	_, resultCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if mock.ExecPod != "web-1" {
		t.Errorf("ExecPod = %q, want web-1", mock.ExecPod)
	}
	if resultCmd == nil {
		t.Error("expected non-nil cmd for exec")
	}
}

func TestShellKey_MultiContainer_ShowsSelector(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{
				Name:   "web-1",
				Status: "Running",
				Containers: []domain.ContainerInfo{
					{Name: "app"},
					{Name: "sidecar"},
				},
			},
		},
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	um := updated.(Model)

	if !um.containerSelector {
		t.Error("containerSelector should be true")
	}
	if um.containerSelectorAction != "exec" {
		t.Errorf("containerSelectorAction = %q, want exec", um.containerSelectorAction)
	}
	if len(um.containerChoices) != 2 {
		t.Errorf("containerChoices = %d, want 2", len(um.containerChoices))
	}
}

func TestShellKey_ReadonlyNamespace_Blocked(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "kube-system",
		Pods: []domain.PodInfo{
			{Name: "coredns-1", Status: "Running"},
		},
	}
	cfg := config.DefaultConfig()
	cfg.ReadonlyNamespaces = []string{"kube-*"}
	m := NewModel(mock, nil, cfg)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	um := updated.(Model)

	if !um.toast.isActive() {
		t.Error("expected toast for readonly namespace")
	}
	if !strings.Contains(um.toast.message, "lecture seule") {
		t.Errorf("toast = %q, want readonly message", um.toast.message)
	}
}

func TestShellKey_BuildExecError_ShowsToast(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "web-1", Status: "Running"},
		},
		BuildExecErr: fmt.Errorf("ni 'oc' ni 'kubectl' trouvé dans le PATH"),
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	um := updated.(Model)

	if !um.toast.isActive() {
		t.Error("expected toast for exec error")
	}
	if !strings.Contains(um.toast.message, "oc") {
		t.Errorf("toast = %q, want message mentioning oc", um.toast.message)
	}
}

func TestContainerSelector_ExecAction_CallsBuildExec(t *testing.T) {
	cmd := exec.Command("echo", "test")
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		ExecCmd:      cmd,
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.containerSelector = true
	m.containerSelectorAction = "exec"
	m.containerPodName = "web-1"
	m.containerChoices = []string{"app", "sidecar"}
	m.containerCursor = 1
	m.width = 120
	m.height = 30

	_, resultCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if mock.ExecContainer != "sidecar" {
		t.Errorf("ExecContainer = %q, want sidecar", mock.ExecContainer)
	}
	if resultCmd == nil {
		t.Error("expected non-nil cmd for exec")
	}
}

func TestExecDoneMsg_ShowsToast(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.width = 120
	m.height = 30

	updated, _ := m.Update(execDoneMsg{err: nil})
	um := updated.(Model)

	if !um.toast.isActive() {
		t.Error("expected toast after exec done")
	}
	if !strings.Contains(um.toast.message, "Shell terminé") {
		t.Errorf("toast = %q, want 'Shell terminé'", um.toast.message)
	}
}

func TestExecDoneMsg_WithError_ShowsErrorToast(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.width = 120
	m.height = 30

	updated, _ := m.Update(execDoneMsg{err: fmt.Errorf("signal: killed")})
	um := updated.(Model)

	if !um.toast.isActive() {
		t.Error("expected error toast")
	}
	if um.toast.level != toastError {
		t.Error("toast should be error level")
	}
}
