package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/domain"
)

func TestEnterOnSingleContainerPod_DirectlyOpensLogs(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "web-1", Status: "Running", Containers: []domain.ContainerInfo{
				{Name: "app", Ready: true, State: "running"},
			}},
		},
		LogContent: "log line",
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)

	if um.containerSelector {
		t.Error("single container pod should NOT show container selector")
	}
	if um.view != ViewLogs {
		t.Errorf("view = %v, want ViewLogs", um.view)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd to load logs")
	}
}

func TestEnterOnMultiContainerPod_ShowsSelector(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "web-1", Status: "Running", Containers: []domain.ContainerInfo{
				{Name: "app", Ready: true, State: "running"},
				{Name: "sidecar", Ready: true, State: "running"},
			}},
		},
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)

	if !um.containerSelector {
		t.Error("multi-container pod should show container selector")
	}
	if len(um.containerChoices) != 2 {
		t.Errorf("containerChoices len = %d, want 2", len(um.containerChoices))
	}
	if um.containerChoices[0] != "app" || um.containerChoices[1] != "sidecar" {
		t.Errorf("containerChoices = %v, want [app sidecar]", um.containerChoices)
	}
}

func TestContainerSelector_Navigation(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.containerSelector = true
	m.containerChoices = []string{"app", "sidecar", "init"}
	m.containerCursor = 0

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)
	if um.containerCursor != 1 {
		t.Errorf("after j: containerCursor = %d, want 1", um.containerCursor)
	}

	// Move up
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um = updated.(Model)
	if um.containerCursor != 0 {
		t.Errorf("after k: containerCursor = %d, want 0", um.containerCursor)
	}

	// Doesn't go below 0
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um = updated.(Model)
	if um.containerCursor != 0 {
		t.Errorf("k at 0: containerCursor = %d, want 0", um.containerCursor)
	}
}

func TestContainerSelector_SelectOpensLogs(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		LogContent:   "container log",
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.containerSelector = true
	m.containerPodName = "web-1"
	m.containerChoices = []string{"app", "sidecar"}
	m.containerCursor = 1 // select "sidecar"
	m.width = 120
	m.height = 30

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)

	if um.containerSelector {
		t.Error("selector should be dismissed after enter")
	}
	if um.view != ViewLogs {
		t.Errorf("view = %v, want ViewLogs", um.view)
	}
	if um.logState.containerName != "sidecar" {
		t.Errorf("logState.containerName = %q, want sidecar", um.logState.containerName)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd to load logs")
	}
}

func TestContainerSelector_EscCancels(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.containerSelector = true
	m.containerChoices = []string{"app", "sidecar"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(Model)

	if um.containerSelector {
		t.Error("esc should cancel container selector")
	}
}

func TestContainerSelector_Render(t *testing.T) {
	output := renderContainerSelector("web-1", []string{"app", "sidecar"}, 0)
	if output == "" {
		t.Fatal("render should return non-empty string")
	}
	if !strings.Contains(output, "web-1") {
		t.Error("should contain pod name")
	}
	if !strings.Contains(output, "app") || !strings.Contains(output, "sidecar") {
		t.Error("should contain container names")
	}
}

func TestLogHeader_ShowsContainerName(t *testing.T) {
	ls := &logState{
		podName:       "web-1",
		containerName: "sidecar",
		content:       "log",
		lines:         []string{"log"},
	}
	output := renderLogs(ls, 120, 20)
	if !strings.Contains(output, "web-1/sidecar") {
		t.Error("log header should show pod/container format")
	}
}

func TestLogHeader_NoContainerName(t *testing.T) {
	ls := &logState{
		podName: "web-1",
		content: "log",
		lines:   []string{"log"},
	}
	output := renderLogs(ls, 120, 20)
	// Should show "Logs: web-1" not "Logs: web-1/"
	if strings.Contains(output, "web-1/") {
		t.Error("log header should NOT show slash when no container name")
	}
}
