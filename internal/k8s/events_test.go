package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8sTesting "k8s.io/client-go/testing"

	"github.com/jclamy/okd-tui/internal/domain"
)

func TestListEvents(t *testing.T) {
	now := metav1.Now()
	events := []corev1.Event{
		{
			ObjectMeta:     metav1.ObjectMeta{Name: "evt-1", Namespace: "default"},
			Type:           "Warning",
			Reason:         "FailedScheduling",
			Message:        "0/3 nodes are available",
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "web-1"},
			LastTimestamp:   now,
			Count:          3,
		},
		{
			ObjectMeta:     metav1.ObjectMeta{Name: "evt-2", Namespace: "default"},
			Type:           "Normal",
			Reason:         "Scheduled",
			Message:        "Successfully assigned",
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "web-2"},
			LastTimestamp:   now,
			Count:          1,
		},
	}

	c, _ := newFakeClient(&corev1.EventList{Items: events})

	result, err := c.ListEvents(context.Background())
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].Type != "Warning" {
		t.Errorf("result[0].Type = %q, want %q", result[0].Type, "Warning")
	}
	if result[0].Reason != "FailedScheduling" {
		t.Errorf("result[0].Reason = %q, want %q", result[0].Reason, "FailedScheduling")
	}
	if result[0].Object != "Pod/web-1" {
		t.Errorf("result[0].Object = %q, want %q", result[0].Object, "Pod/web-1")
	}
	if result[0].Count != 3 {
		t.Errorf("result[0].Count = %d, want 3", result[0].Count)
	}
	if result[1].Reason != "Scheduled" {
		t.Errorf("result[1].Reason = %q, want %q", result[1].Reason, "Scheduled")
	}
}

func TestWatchEvents_ReceivesAddedEvent(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("events", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.WatchEvents(ctx)
	if err != nil {
		t.Fatalf("WatchEvents() error = %v", err)
	}

	evt := &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: "evt-1", Namespace: "default"},
		Type:           "Warning",
		Reason:         "BackOff",
		Message:        "Back-off restarting failed container",
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "web-1"},
		Count:          12,
	}
	go fakeWatcher.Add(evt)

	select {
	case got := <-ch:
		if got.Type != domain.EventAdded {
			t.Errorf("Type = %q, want %q", got.Type, domain.EventAdded)
		}
		if got.Resource != "event" {
			t.Errorf("Resource = %q, want %q", got.Resource, "event")
		}
		if got.Event == nil {
			t.Fatal("Event should not be nil")
		}
		if got.Event.Reason != "BackOff" {
			t.Errorf("Event.Reason = %q, want %q", got.Event.Reason, "BackOff")
		}
		if got.Event.Count != 12 {
			t.Errorf("Event.Count = %d, want 12", got.Event.Count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestWatchEvents_ContextCancelClosesChannel(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("events", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.WatchEvents(ctx)
	if err != nil {
		t.Fatalf("WatchEvents() error = %v", err)
	}

	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			_, ok = <-ch
		}
		if ok {
			t.Error("channel should be closed after context cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}
