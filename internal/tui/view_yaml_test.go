package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/domain"
)

func TestYAMLKey_LoadsPodYAML(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "web-1", Status: "Running"},
		},
		PodYAML: "apiVersion: v1\nkind: Pod",
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = mock.Pods
	m.width = 120
	m.height = 30

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	um := updated.(Model)

	if um.view != ViewYAML {
		t.Errorf("view = %v, want ViewYAML", um.view)
	}
	if um.yamlState.resourceType != "pod" {
		t.Errorf("resourceType = %q, want pod", um.yamlState.resourceType)
	}
	if um.yamlState.resourceName != "web-1" {
		t.Errorf("resourceName = %q, want web-1", um.yamlState.resourceName)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd to load YAML")
	}
}

func TestYAMLKey_LoadsDeploymentYAML(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal:   "default",
		DeploymentYAML: "apiVersion: apps/v1\nkind: Deployment",
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewDeployments
	m.deployments = []domain.DeploymentInfo{{Name: "api-deploy"}}
	m.width = 120
	m.height = 30

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	um := updated.(Model)

	if um.view != ViewYAML {
		t.Errorf("view = %v, want ViewYAML", um.view)
	}
	if um.yamlState.resourceType != "deployment" {
		t.Errorf("resourceType = %q, want deployment", um.yamlState.resourceType)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestYAMLLoadedMsg_SetsContent(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewYAML
	m.loading = true

	updated, _ := m.Update(yamlLoadedMsg{content: "kind: Pod\nmetadata:\n  name: web"})
	um := updated.(Model)

	if um.loading {
		t.Error("loading should be false")
	}
	if len(um.yamlState.lines) != 3 {
		t.Errorf("lines = %d, want 3", len(um.yamlState.lines))
	}
}

func TestYAMLView_EscReturns(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewYAML
	m.prevView = ViewPods
	m.yamlState = yamlViewState{content: "test", lines: []string{"test"}}
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(Model)

	if um.view != ViewPods {
		t.Errorf("view = %v, want ViewPods", um.view)
	}
	if um.yamlState.content != "" {
		t.Error("yamlState should be cleared")
	}
}

func TestYAMLView_Scroll(t *testing.T) {
	ys := &yamlViewState{}
	// 30 lines
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line"
	}
	ys.setContent(strings.Join(lines, "\n"))

	ys.scrollDown(20, 10)
	if ys.offset != 20 {
		t.Errorf("after scrollDown: offset = %d, want 20", ys.offset)
	}
	ys.scrollUp(5)
	if ys.offset != 15 {
		t.Errorf("after scrollUp: offset = %d, want 15", ys.offset)
	}
}

func TestYAMLView_JumpToBottom(t *testing.T) {
	ys := &yamlViewState{}
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	ys.setContent(strings.Join(lines, "\n"))

	ys.jumpToBottom(30)
	expected := len(ys.lines) - 30
	if ys.offset != expected {
		t.Errorf("jumpToBottom: offset = %d, want %d", ys.offset, expected)
	}

	// Short content
	ys.setContent("short")
	ys.jumpToBottom(30)
	if ys.offset != 0 {
		t.Errorf("jumpToBottom short: offset = %d, want 0", ys.offset)
	}
}

func TestRenderYAMLView_Header(t *testing.T) {
	ys := &yamlViewState{
		resourceType: "pod",
		resourceName: "web-1",
		content:      "kind: Pod",
		lines:        []string{"kind: Pod"},
	}
	output := renderYAMLView(ys, 120, 20)
	if !strings.Contains(output, "pod/web-1") {
		t.Error("header should contain resource type/name")
	}
}
