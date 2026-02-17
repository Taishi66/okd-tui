package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"

	"github.com/Taishi66/okd-tui/internal/domain"
)

func newFakeClient(objects ...runtime.Object) (*Client, *fakeK8s.Clientset) {
	cs := fakeK8s.NewSimpleClientset(objects...)
	return &Client{
		clientset: cs,
		namespace: "default",
		serverURL: "https://fake:6443",
	}, cs
}

func TestWatchPods_ReceivesAddedEvent(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("pods", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.WatchPods(ctx)
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	go fakeWatcher.Add(pod)

	select {
	case evt := <-ch:
		if evt.Type != domain.EventAdded {
			t.Errorf("Type = %q, want %q", evt.Type, domain.EventAdded)
		}
		if evt.Resource != "pod" {
			t.Errorf("Resource = %q, want %q", evt.Resource, "pod")
		}
		if evt.Pod == nil || evt.Pod.Name != "web-1" {
			t.Errorf("Pod.Name = %v, want web-1", evt.Pod)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestWatchPods_ReceivesModifiedEvent(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("pods", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.WatchPods(ctx)
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodFailed},
	}
	go fakeWatcher.Modify(pod)

	select {
	case evt := <-ch:
		if evt.Type != domain.EventModified {
			t.Errorf("Type = %q, want %q", evt.Type, domain.EventModified)
		}
		if evt.Pod.Status != "Failed" {
			t.Errorf("Pod.Status = %q, want %q", evt.Pod.Status, "Failed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestWatchPods_ReceivesDeletedEvent(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("pods", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.WatchPods(ctx)
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default"},
	}
	go fakeWatcher.Delete(pod)

	select {
	case evt := <-ch:
		if evt.Type != domain.EventDeleted {
			t.Errorf("Type = %q, want %q", evt.Type, domain.EventDeleted)
		}
		if evt.Pod.Name != "web-1" {
			t.Errorf("Pod.Name = %q, want %q", evt.Pod.Name, "web-1")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestWatchPods_ContextCancelClosesChannel(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("pods", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.WatchPods(ctx)
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
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

func TestWatchDeployments_ReceivesAddedEvent(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("deployments", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.WatchDeployments(ctx)
	if err != nil {
		t.Fatalf("WatchDeployments() error = %v", err)
	}

	replicas := int32(3)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     2,
			AvailableReplicas: 2,
		},
	}
	go fakeWatcher.Add(dep)

	select {
	case evt := <-ch:
		if evt.Type != domain.EventAdded {
			t.Errorf("Type = %q, want %q", evt.Type, domain.EventAdded)
		}
		if evt.Resource != "deployment" {
			t.Errorf("Resource = %q, want %q", evt.Resource, "deployment")
		}
		if evt.Deployment == nil || evt.Deployment.Name != "api" {
			t.Errorf("Deployment.Name = %v, want api", evt.Deployment)
		}
		if evt.Deployment.Replicas != 3 {
			t.Errorf("Deployment.Replicas = %d, want 3", evt.Deployment.Replicas)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestWatchDeployments_ContextCancelClosesChannel(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("deployments", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.WatchDeployments(ctx)
	if err != nil {
		t.Fatalf("WatchDeployments() error = %v", err)
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

func TestWatchPods_ChannelClosedWhenWatcherStops(t *testing.T) {
	c, cs := newFakeClient()
	fakeWatcher := watch.NewFake()
	cs.PrependWatchReactor("pods", k8sTesting.DefaultWatchReactor(fakeWatcher, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.WatchPods(ctx)
	if err != nil {
		t.Fatalf("WatchPods() error = %v", err)
	}

	fakeWatcher.Stop()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed when watcher stops")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}
