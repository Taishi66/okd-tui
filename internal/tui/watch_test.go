package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/domain"
)

// --- Pod merge tests ---

func TestMergePodEvent_Added(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.pods = []domain.PodInfo{
		{Name: "web-1", Status: "Running"},
	}

	newPod := domain.PodInfo{Name: "web-2", Status: "Running"}
	m.mergePodEvent(domain.WatchEvent{Type: domain.EventAdded, Pod: &newPod})

	if len(m.pods) != 2 {
		t.Fatalf("len(pods) = %d, want 2", len(m.pods))
	}
	if m.pods[1].Name != "web-2" {
		t.Errorf("pods[1].Name = %q, want %q", m.pods[1].Name, "web-2")
	}
}

func TestMergePodEvent_Modified(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.pods = []domain.PodInfo{
		{Name: "web-1", Status: "Running"},
		{Name: "web-2", Status: "Running"},
	}

	updated := domain.PodInfo{Name: "web-1", Status: "CrashLoopBackOff"}
	m.mergePodEvent(domain.WatchEvent{Type: domain.EventModified, Pod: &updated})

	if len(m.pods) != 2 {
		t.Fatalf("len(pods) = %d, want 2", len(m.pods))
	}
	if m.pods[0].Status != "CrashLoopBackOff" {
		t.Errorf("pods[0].Status = %q, want %q", m.pods[0].Status, "CrashLoopBackOff")
	}
}

func TestMergePodEvent_Deleted(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.pods = []domain.PodInfo{
		{Name: "web-1", Status: "Running"},
		{Name: "web-2", Status: "Running"},
		{Name: "web-3", Status: "Running"},
	}

	deleted := domain.PodInfo{Name: "web-2"}
	m.mergePodEvent(domain.WatchEvent{Type: domain.EventDeleted, Pod: &deleted})

	if len(m.pods) != 2 {
		t.Fatalf("len(pods) = %d, want 2", len(m.pods))
	}
	if m.pods[0].Name != "web-1" || m.pods[1].Name != "web-3" {
		t.Errorf("pods = [%s, %s], want [web-1, web-3]", m.pods[0].Name, m.pods[1].Name)
	}
}

func TestMergePodEvent_DeletedAdjustsCursor(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.pods = []domain.PodInfo{
		{Name: "web-1"},
		{Name: "web-2"},
	}
	m.cursor = 1 // pointing at web-2

	deleted := domain.PodInfo{Name: "web-2"}
	m.mergePodEvent(domain.WatchEvent{Type: domain.EventDeleted, Pod: &deleted})

	if len(m.pods) != 1 {
		t.Fatalf("len(pods) = %d, want 1", len(m.pods))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (adjusted after delete)", m.cursor)
	}
}

func TestMergePodEvent_DeletedKeepsCursorWhenNotAtEnd(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.pods = []domain.PodInfo{
		{Name: "web-1"},
		{Name: "web-2"},
		{Name: "web-3"},
	}
	m.cursor = 0 // pointing at web-1

	deleted := domain.PodInfo{Name: "web-3"}
	m.mergePodEvent(domain.WatchEvent{Type: domain.EventDeleted, Pod: &deleted})

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (should not change)", m.cursor)
	}
}

// --- Deployment merge tests ---

func TestMergeDeploymentEvent_Added(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.deployments = []domain.DeploymentInfo{
		{Name: "api", Replicas: 2},
	}

	newDep := domain.DeploymentInfo{Name: "worker", Replicas: 1}
	m.mergeDeploymentEvent(domain.WatchEvent{Type: domain.EventAdded, Deployment: &newDep})

	if len(m.deployments) != 2 {
		t.Fatalf("len(deployments) = %d, want 2", len(m.deployments))
	}
	if m.deployments[1].Name != "worker" {
		t.Errorf("deployments[1].Name = %q, want %q", m.deployments[1].Name, "worker")
	}
}

func TestMergeDeploymentEvent_Modified(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.deployments = []domain.DeploymentInfo{
		{Name: "api", Replicas: 2, Available: 2},
	}

	updated := domain.DeploymentInfo{Name: "api", Replicas: 5, Available: 3}
	m.mergeDeploymentEvent(domain.WatchEvent{Type: domain.EventModified, Deployment: &updated})

	if m.deployments[0].Replicas != 5 {
		t.Errorf("Replicas = %d, want 5", m.deployments[0].Replicas)
	}
	if m.deployments[0].Available != 3 {
		t.Errorf("Available = %d, want 3", m.deployments[0].Available)
	}
}

func TestMergeDeploymentEvent_Deleted(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil)
	m.deployments = []domain.DeploymentInfo{
		{Name: "api"},
		{Name: "worker"},
	}
	m.cursor = 1

	deleted := domain.DeploymentInfo{Name: "worker"}
	m.mergeDeploymentEvent(domain.WatchEvent{Type: domain.EventDeleted, Deployment: &deleted})

	if len(m.deployments) != 1 {
		t.Fatalf("len(deployments) = %d, want 1", len(m.deployments))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

// --- Watch lifecycle tests ---

func TestWatchEventMsg_MergesPodAndReturnsCmd(t *testing.T) {
	watchCh := make(chan domain.WatchEvent, 1)
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		WatchPodsCh:  watchCh,
	}
	m := NewModel(mock, nil)
	m.view = ViewPods
	m.pods = []domain.PodInfo{{Name: "web-1", Status: "Running"}}
	m.watching = true
	m.watchCh = watchCh // set the channel so listenWatch can chain

	newPod := domain.PodInfo{Name: "web-2", Status: "Pending"}
	msg := watchEventMsg{event: domain.WatchEvent{Type: domain.EventAdded, Resource: "pod", Pod: &newPod}}

	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.pods) != 2 {
		t.Fatalf("len(pods) = %d, want 2", len(updated.pods))
	}
	if updated.pods[1].Name != "web-2" {
		t.Errorf("pods[1].Name = %q, want %q", updated.pods[1].Name, "web-2")
	}
	// Should return a cmd to listen for the next event
	if cmd == nil {
		t.Error("expected non-nil cmd (listen for next event)")
	}
}

func TestWatchEventMsg_MergesDeployment(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal:       "default",
		WatchDeploymentsCh: make(chan domain.WatchEvent, 1),
	}
	m := NewModel(mock, nil)
	m.view = ViewDeployments
	m.deployments = []domain.DeploymentInfo{{Name: "api", Replicas: 2}}
	m.watching = true

	updatedDep := domain.DeploymentInfo{Name: "api", Replicas: 5}
	msg := watchEventMsg{event: domain.WatchEvent{Type: domain.EventModified, Resource: "deployment", Deployment: &updatedDep}}

	newModel, _ := m.Update(msg)
	result := newModel.(Model)

	if result.deployments[0].Replicas != 5 {
		t.Errorf("Replicas = %d, want 5", result.deployments[0].Replicas)
	}
}

func TestWatchStoppedMsg_Reconnects(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		WatchPodsCh:  make(chan domain.WatchEvent),
	}
	m := NewModel(mock, nil)
	m.view = ViewPods
	m.watching = true

	msg := watchStoppedMsg{resource: "pod"}

	newModel, cmd := m.Update(msg)
	result := newModel.(Model)

	// startWatch() succeeds (mock has channel), so watching should be true again
	if !result.watching {
		t.Error("watching should be true after successful reconnect")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (restart watch)")
	}
}

func TestWatchStoppedMsg_NoChannelStaysOff(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		// No WatchPodsCh → startWatch returns nil channel
	}
	m := NewModel(mock, nil)
	m.view = ViewPods
	m.watching = true

	msg := watchStoppedMsg{resource: "pod"}

	newModel, _ := m.Update(msg)
	result := newModel.(Model)

	if result.watching {
		t.Error("watching should be false when no channel available")
	}
}

func TestSwitchView_CancelsWatch(t *testing.T) {
	cancelled := false
	mock := &domain.MockGateway{
		NamespaceVal:   "default",
		Pods:           []domain.PodInfo{{Name: "web-1"}},
		Deployments:    []domain.DeploymentInfo{{Name: "api"}},
		WatchPodsCh:    make(chan domain.WatchEvent),
		WatchDeploymentsCh: make(chan domain.WatchEvent),
	}
	m := NewModel(mock, nil)
	m.view = ViewPods
	m.watching = true
	m.watchCancel = func() { cancelled = true }

	// Switch to deployments via tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	result := newModel.(Model)

	if !cancelled {
		t.Error("watchCancel should have been called on view switch")
	}
	if result.view != ViewDeployments {
		t.Errorf("view = %v, want ViewDeployments", result.view)
	}
}

func TestStatusBar_ShowsLiveIndicator(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "test-ns"}
	m := NewModel(mock, nil)
	m.view = ViewPods
	m.width = 120
	m.height = 30
	m.watching = true

	output := m.View()

	if !strings.Contains(output, "LIVE") {
		t.Error("status bar should contain LIVE indicator when watching")
	}
}

func TestStatusBar_NoLiveWhenNotWatching(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "test-ns"}
	m := NewModel(mock, nil)
	m.view = ViewPods
	m.width = 120
	m.height = 30
	m.watching = false

	output := m.View()

	if strings.Contains(output, "LIVE") {
		t.Error("status bar should NOT contain LIVE when not watching")
	}
}

func TestPodsLoadedMsg_StartsWatch(t *testing.T) {
	ch := make(chan domain.WatchEvent, 1)
	mock := &domain.MockGateway{
		NamespaceVal: "default",
		WatchPodsCh:  ch,
	}
	m := NewModel(mock, nil)
	m.view = ViewPods

	msg := podsLoadedMsg{items: []domain.PodInfo{{Name: "web-1"}}}
	newModel, cmd := m.Update(msg)
	result := newModel.(Model)

	if !result.watching {
		t.Error("watching should be true after podsLoadedMsg")
	}
	if result.watchCancel == nil {
		t.Error("watchCancel should be set after podsLoadedMsg")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (start listening)")
	}
}

func TestDeploymentsLoadedMsg_StartsWatch(t *testing.T) {
	ch := make(chan domain.WatchEvent, 1)
	mock := &domain.MockGateway{
		NamespaceVal:       "default",
		WatchDeploymentsCh: ch,
	}
	m := NewModel(mock, nil)
	m.view = ViewDeployments

	msg := deploymentsLoadedMsg{items: []domain.DeploymentInfo{{Name: "api"}}}
	newModel, cmd := m.Update(msg)
	result := newModel.(Model)

	if !result.watching {
		t.Error("watching should be true after deploymentsLoadedMsg")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (start listening)")
	}
}
