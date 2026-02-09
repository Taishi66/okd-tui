package domain

import (
	"context"
	"testing"
)

func TestWatchEventTypeConstants(t *testing.T) {
	// Verify the three event type constants exist and have correct values
	tests := []struct {
		got  WatchEventType
		want string
	}{
		{EventAdded, "ADDED"},
		{EventModified, "MODIFIED"},
		{EventDeleted, "DELETED"},
	}
	for _, tt := range tests {
		if string(tt.got) != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestWatchEventPodField(t *testing.T) {
	pod := PodInfo{Name: "web-1", Status: "Running"}
	evt := WatchEvent{
		Type:     EventAdded,
		Resource: "pod",
		Pod:      &pod,
	}
	if evt.Type != EventAdded {
		t.Errorf("Type = %q, want %q", evt.Type, EventAdded)
	}
	if evt.Resource != "pod" {
		t.Errorf("Resource = %q, want %q", evt.Resource, "pod")
	}
	if evt.Pod.Name != "web-1" {
		t.Errorf("Pod.Name = %q, want %q", evt.Pod.Name, "web-1")
	}
	if evt.Deployment != nil {
		t.Error("Deployment should be nil for pod event")
	}
}

func TestWatchEventDeploymentField(t *testing.T) {
	dep := DeploymentInfo{Name: "api", Replicas: 3}
	evt := WatchEvent{
		Type:       EventModified,
		Resource:   "deployment",
		Deployment: &dep,
	}
	if evt.Type != EventModified {
		t.Errorf("Type = %q, want %q", evt.Type, EventModified)
	}
	if evt.Resource != "deployment" {
		t.Errorf("Resource = %q, want %q", evt.Resource, "deployment")
	}
	if evt.Deployment.Name != "api" {
		t.Errorf("Deployment.Name = %q, want %q", evt.Deployment.Name, "api")
	}
	if evt.Deployment.Replicas != 3 {
		t.Errorf("Deployment.Replicas = %d, want %d", evt.Deployment.Replicas, 3)
	}
	if evt.Pod != nil {
		t.Error("Pod should be nil for deployment event")
	}
}

func TestWatchEventDeletedType(t *testing.T) {
	pod := PodInfo{Name: "worker-5"}
	evt := WatchEvent{
		Type:     EventDeleted,
		Resource: "pod",
		Pod:      &pod,
	}
	if evt.Type != EventDeleted {
		t.Errorf("Type = %q, want %q", evt.Type, EventDeleted)
	}
}

// --- MockGateway Watch tests ---

func TestMockGatewayImplementsKubeGateway(t *testing.T) {
	// Compile-time check already exists, but verify at runtime too
	var gw KubeGateway = &MockGateway{}
	if gw == nil {
		t.Fatal("MockGateway should implement KubeGateway")
	}
}

func TestMockGatewayWatchPodsReturnsChannel(t *testing.T) {
	ch := make(chan WatchEvent, 1)
	mock := &MockGateway{WatchPodsCh: ch}

	gotCh, err := mock.WatchPods(context.Background())
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
	}

	pod := PodInfo{Name: "new-pod", Status: "Running"}
	ch <- WatchEvent{Type: EventAdded, Resource: "pod", Pod: &pod}

	evt := <-gotCh
	if evt.Type != EventAdded {
		t.Errorf("event Type = %q, want %q", evt.Type, EventAdded)
	}
	if evt.Pod.Name != "new-pod" {
		t.Errorf("event Pod.Name = %q, want %q", evt.Pod.Name, "new-pod")
	}
}

func TestMockGatewayWatchPodsReturnsError(t *testing.T) {
	mock := &MockGateway{
		WatchPodsErr: &APIError{Type: ErrForbidden, Message: "forbidden"},
	}

	_, err := mock.WatchPods(context.Background())
	if err == nil {
		t.Fatal("WatchPods() should return error")
	}
}

func TestMockGatewayWatchDeploymentsReturnsChannel(t *testing.T) {
	ch := make(chan WatchEvent, 1)
	mock := &MockGateway{WatchDeploymentsCh: ch}

	gotCh, err := mock.WatchDeployments(context.Background())
	if err != nil {
		t.Fatalf("WatchDeployments() error = %v", err)
	}

	dep := DeploymentInfo{Name: "api", Replicas: 2}
	ch <- WatchEvent{Type: EventModified, Resource: "deployment", Deployment: &dep}

	evt := <-gotCh
	if evt.Type != EventModified {
		t.Errorf("event Type = %q, want %q", evt.Type, EventModified)
	}
	if evt.Deployment.Name != "api" {
		t.Errorf("event Deployment.Name = %q, want %q", evt.Deployment.Name, "api")
	}
}

func TestMockGatewayWatchDeploymentsReturnsError(t *testing.T) {
	mock := &MockGateway{
		WatchDeploymentsErr: &APIError{Type: ErrUnreachable, Message: "unreachable"},
	}

	_, err := mock.WatchDeployments(context.Background())
	if err == nil {
		t.Fatal("WatchDeployments() should return error")
	}
}

func TestMockGatewayWatchPodsNilChannelReturnsNil(t *testing.T) {
	mock := &MockGateway{} // no channel set

	ch, err := mock.WatchPods(context.Background())
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
	}
	if ch != nil {
		t.Error("WatchPods() should return nil channel when none set")
	}
}
