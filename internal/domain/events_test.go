package domain

import (
	"context"
	"testing"
)

func TestEventInfoFields(t *testing.T) {
	evt := EventInfo{
		Type:      "Warning",
		Reason:    "FailedScheduling",
		Message:   "0/3 nodes are available",
		Object:    "pod/web-1",
		Namespace: "default",
		Age:       "2m",
		Count:     5,
	}
	if evt.Type != "Warning" {
		t.Errorf("Type = %q, want %q", evt.Type, "Warning")
	}
	if evt.Reason != "FailedScheduling" {
		t.Errorf("Reason = %q, want %q", evt.Reason, "FailedScheduling")
	}
	if evt.Message != "0/3 nodes are available" {
		t.Errorf("Message = %q", evt.Message)
	}
	if evt.Object != "pod/web-1" {
		t.Errorf("Object = %q, want %q", evt.Object, "pod/web-1")
	}
	if evt.Count != 5 {
		t.Errorf("Count = %d, want 5", evt.Count)
	}
}

func TestWatchEventHasEventField(t *testing.T) {
	info := EventInfo{Type: "Normal", Reason: "Scheduled"}
	evt := WatchEvent{
		Type:     EventAdded,
		Resource: "event",
		Event:    &info,
	}
	if evt.Event == nil {
		t.Fatal("Event field should not be nil")
	}
	if evt.Event.Reason != "Scheduled" {
		t.Errorf("Event.Reason = %q, want %q", evt.Event.Reason, "Scheduled")
	}
}

func TestMockGatewayListEvents(t *testing.T) {
	mock := &MockGateway{
		Events: []EventInfo{
			{Type: "Warning", Reason: "BackOff", Object: "pod/web-1"},
			{Type: "Normal", Reason: "Scheduled", Object: "pod/web-2"},
		},
	}

	events, err := mock.ListEvents(context.Background())
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Reason != "BackOff" {
		t.Errorf("events[0].Reason = %q, want %q", events[0].Reason, "BackOff")
	}
}

func TestMockGatewayListEventsError(t *testing.T) {
	mock := &MockGateway{
		ListEventsErr: &APIError{Type: ErrForbidden, Message: "forbidden"},
	}
	_, err := mock.ListEvents(context.Background())
	if err == nil {
		t.Fatal("ListEvents() should return error")
	}
}

func TestMockGatewayWatchEvents(t *testing.T) {
	ch := make(chan WatchEvent, 1)
	mock := &MockGateway{WatchEventsCh: ch}

	gotCh, err := mock.WatchEvents(context.Background())
	if err != nil {
		t.Fatalf("WatchEvents() error = %v", err)
	}

	info := EventInfo{Type: "Warning", Reason: "Killing"}
	ch <- WatchEvent{Type: EventAdded, Resource: "event", Event: &info}

	evt := <-gotCh
	if evt.Event.Reason != "Killing" {
		t.Errorf("Event.Reason = %q, want %q", evt.Event.Reason, "Killing")
	}
}

func TestMockGatewayWatchEventsError(t *testing.T) {
	mock := &MockGateway{
		WatchEventsErr: &APIError{Type: ErrUnreachable, Message: "unreachable"},
	}
	_, err := mock.WatchEvents(context.Background())
	if err == nil {
		t.Fatal("WatchEvents() should return error")
	}
}
