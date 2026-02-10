package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/domain"
)

// --- ViewEvents basics ---

func TestViewEvents_String(t *testing.T) {
	if ViewEvents.String() != "EVENTS" {
		t.Errorf("ViewEvents.String() = %q, want %q", ViewEvents.String(), "EVENTS")
	}
}

func TestViewEvents_IotaValue(t *testing.T) {
	// ViewEvents should be between ViewDeployments and ViewLogs
	if ViewEvents != 3 {
		t.Errorf("ViewEvents = %d, want 3", ViewEvents)
	}
	if ViewLogs != 4 {
		t.Errorf("ViewLogs = %d, want 4 (shifted by ViewEvents insertion)", ViewLogs)
	}
}

// --- Tab navigation ---

func TestTab4_SwitchesToEvents(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal:  "default",
		Events:        []domain.EventInfo{{Type: "Warning", Reason: "BackOff"}},
		WatchEventsCh: make(chan domain.WatchEvent),
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.width = 120
	m.height = 30

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	result := newModel.(Model)

	if result.view != ViewEvents {
		t.Errorf("view = %v, want ViewEvents", result.view)
	}
}

func TestTabNext_CyclesThrough4Views(t *testing.T) {
	mock := &domain.MockGateway{
		NamespaceVal:       "default",
		WatchPodsCh:        make(chan domain.WatchEvent),
		WatchDeploymentsCh: make(chan domain.WatchEvent),
		WatchEventsCh:      make(chan domain.WatchEvent),
	}

	views := []View{ViewProjects, ViewPods, ViewDeployments, ViewEvents}
	expected := []View{ViewPods, ViewDeployments, ViewEvents, ViewProjects}

	for i, startView := range views {
		m := NewModel(mock, nil, nil)
		m.view = startView
		m.width = 120
		m.height = 30

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		result := newModel.(Model)

		if result.view != expected[i] {
			t.Errorf("from %v: tab → %v, want %v", startView, result.view, expected[i])
		}
	}
}

// --- Events loaded ---

func TestEventsLoadedMsg_PopulatesEvents(t *testing.T) {
	ch := make(chan domain.WatchEvent, 1)
	mock := &domain.MockGateway{
		NamespaceVal:  "default",
		WatchEventsCh: ch,
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewEvents

	events := []domain.EventInfo{
		{Type: "Warning", Reason: "FailedScheduling", Message: "0/3 nodes available", Object: "Pod/web-1", Count: 3},
		{Type: "Normal", Reason: "Scheduled", Message: "Successfully assigned", Object: "Pod/web-2", Count: 1},
	}
	msg := eventsLoadedMsg{items: events}

	newModel, cmd := m.Update(msg)
	result := newModel.(Model)

	if len(result.events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(result.events))
	}
	if result.events[0].Reason != "FailedScheduling" {
		t.Errorf("events[0].Reason = %q, want %q", result.events[0].Reason, "FailedScheduling")
	}
	if !result.watching {
		t.Error("watching should be true after eventsLoadedMsg")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (start watch)")
	}
}

// --- Event merge ---

func TestMergeEventEvent_Added(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.events = []domain.EventInfo{
		{Reason: "Scheduled", Object: "Pod/web-1"},
	}

	newEvt := domain.EventInfo{Reason: "Pulled", Object: "Pod/web-1", Message: "Pulled image"}
	m.mergeEventEvent(domain.WatchEvent{Type: domain.EventAdded, Event: &newEvt})

	if len(m.events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(m.events))
	}
	if m.events[1].Reason != "Pulled" {
		t.Errorf("events[1].Reason = %q, want %q", m.events[1].Reason, "Pulled")
	}
}

func TestMergeEventEvent_Modified(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.events = []domain.EventInfo{
		{Reason: "BackOff", Object: "Pod/web-1", Count: 3},
		{Reason: "Scheduled", Object: "Pod/web-2", Count: 1},
	}

	updated := domain.EventInfo{Reason: "BackOff", Object: "Pod/web-1", Count: 5}
	m.mergeEventEvent(domain.WatchEvent{Type: domain.EventModified, Event: &updated})

	if m.events[0].Count != 5 {
		t.Errorf("events[0].Count = %d, want 5", m.events[0].Count)
	}
}

func TestMergeEventEvent_Deleted(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.events = []domain.EventInfo{
		{Reason: "Scheduled", Object: "Pod/web-1"},
		{Reason: "BackOff", Object: "Pod/web-2"},
		{Reason: "Pulled", Object: "Pod/web-3"},
	}
	m.cursor = 2

	deleted := domain.EventInfo{Reason: "Pulled", Object: "Pod/web-3"}
	m.mergeEventEvent(domain.WatchEvent{Type: domain.EventDeleted, Event: &deleted})

	if len(m.events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(m.events))
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (adjusted after delete)", m.cursor)
	}
}

// --- Watch event msg for events ---

func TestWatchEventMsg_MergesEvent(t *testing.T) {
	watchCh := make(chan domain.WatchEvent, 1)
	mock := &domain.MockGateway{
		NamespaceVal:  "default",
		WatchEventsCh: watchCh,
	}
	m := NewModel(mock, nil, nil)
	m.view = ViewEvents
	m.events = []domain.EventInfo{{Reason: "Scheduled", Object: "Pod/web-1"}}
	m.watching = true
	m.watchCh = watchCh

	newEvt := domain.EventInfo{Reason: "Pulled", Object: "Pod/web-1", Message: "Pulled image"}
	msg := watchEventMsg{event: domain.WatchEvent{Type: domain.EventAdded, Resource: "event", Event: &newEvt}}

	newModel, cmd := m.Update(msg)
	result := newModel.(Model)

	if len(result.events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(result.events))
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (listen for next event)")
	}
}

// --- Filtering ---

func TestFilteredEvents_NoFilter(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.events = []domain.EventInfo{
		{Reason: "BackOff", Message: "Back-off restarting"},
		{Reason: "Scheduled", Message: "Successfully assigned"},
	}

	result := m.filteredEvents()
	if len(result) != 2 {
		t.Errorf("len(filteredEvents) = %d, want 2", len(result))
	}
}

func TestFilteredEvents_ByReason(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.events = []domain.EventInfo{
		{Reason: "BackOff", Message: "Back-off restarting"},
		{Reason: "Scheduled", Message: "Successfully assigned"},
		{Reason: "BackOff", Message: "Back-off pulling"},
	}
	m.filter.SetValue("backoff")

	result := m.filteredEvents()
	if len(result) != 2 {
		t.Errorf("len(filteredEvents) = %d, want 2", len(result))
	}
}

func TestFilteredEvents_ByMessage(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.events = []domain.EventInfo{
		{Reason: "BackOff", Message: "Back-off restarting"},
		{Reason: "Scheduled", Message: "Successfully assigned"},
	}
	m.filter.SetValue("assigned")

	result := m.filteredEvents()
	if len(result) != 1 {
		t.Errorf("len(filteredEvents) = %d, want 1", len(result))
	}
	if result[0].Reason != "Scheduled" {
		t.Errorf("result[0].Reason = %q, want %q", result[0].Reason, "Scheduled")
	}
}

// --- listLen ---

func TestListLen_ViewEvents(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewEvents
	m.events = []domain.EventInfo{
		{Reason: "BackOff"},
		{Reason: "Scheduled"},
		{Reason: "Pulled"},
	}

	if m.listLen() != 3 {
		t.Errorf("listLen() = %d, want 3", m.listLen())
	}
}

// --- Render ---

func TestRenderEventList_Columns(t *testing.T) {
	events := []domain.EventInfo{
		{Type: "Warning", Reason: "BackOff", Object: "Pod/web-1", Message: "Back-off restarting", Age: "2m", Count: 5},
		{Type: "Normal", Reason: "Scheduled", Object: "Pod/web-2", Message: "Successfully assigned", Age: "5m", Count: 1},
	}

	output := renderEventList(events, 0, 120, 20)

	// Header columns
	if !strings.Contains(output, "TYPE") {
		t.Error("output should contain TYPE column header")
	}
	if !strings.Contains(output, "REASON") {
		t.Error("output should contain REASON column header")
	}
	if !strings.Contains(output, "OBJECT") {
		t.Error("output should contain OBJECT column header")
	}
	if !strings.Contains(output, "MESSAGE") {
		t.Error("output should contain MESSAGE column header")
	}
	if !strings.Contains(output, "AGE") {
		t.Error("output should contain AGE column header")
	}
	if !strings.Contains(output, "COUNT") {
		t.Error("output should contain COUNT column header")
	}

	// Data
	if !strings.Contains(output, "BackOff") {
		t.Error("output should contain event reason 'BackOff'")
	}
	if !strings.Contains(output, "Pod/web-1") {
		t.Error("output should contain event object 'Pod/web-1'")
	}
}

func TestRenderEventList_Empty(t *testing.T) {
	output := renderEventList(nil, 0, 120, 20)

	if !strings.Contains(output, "Aucun") {
		t.Error("empty event list should show 'Aucun' message")
	}
}

// --- Status bar ---

func TestStatusBar_ShowsEventsView(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "test-ns"}
	m := NewModel(mock, nil, nil)
	m.view = ViewEvents
	m.width = 120
	m.height = 30

	output := m.View()

	if !strings.Contains(output, "EVENTS") {
		t.Error("status bar should contain 'EVENTS' for events view")
	}
}

func TestStatusBar_ShowsLiveOnEventsView(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "test-ns"}
	m := NewModel(mock, nil, nil)
	m.view = ViewEvents
	m.width = 120
	m.height = 30
	m.watching = true

	output := m.View()

	if !strings.Contains(output, "LIVE") {
		t.Error("status bar should contain LIVE indicator when watching events")
	}
}

// --- Tabs render ---

func TestRenderTabs_IncludesEvents(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewEvents
	m.width = 120
	m.height = 30

	output := m.View()

	if !strings.Contains(output, "[4] Events") {
		t.Error("tabs should contain '[4] Events'")
	}
}
