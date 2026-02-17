package cache

import (
	"context"
	"testing"
	"time"

	"github.com/Taishi66/okd-tui/internal/config"
	"github.com/Taishi66/okd-tui/internal/domain"
)

func newTestCache() (*CachedGateway, *domain.MockGateway) {
	mock := &domain.MockGateway{
		ContextVal:   "test",
		ServerURLVal: "https://test:6443",
		NamespaceVal: "default",
		Pods:         []domain.PodInfo{{Name: "web-1"}},
		Deployments:  []domain.DeploymentInfo{{Name: "api"}},
		Namespaces:   []domain.NamespaceInfo{{Name: "default"}},
		Events:       []domain.EventInfo{{Reason: "Pulled"}},
	}
	cfg := config.CacheConfig{
		PodsTTL:        100 * time.Millisecond,
		DeploymentsTTL: 100 * time.Millisecond,
		NamespacesTTL:  100 * time.Millisecond,
		EventsTTL:      100 * time.Millisecond,
	}
	return NewCachedGateway(mock, cfg), mock
}

func TestCachedGateway_CachesListPods(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListPods(ctx)
	_, _ = c.ListPods(ctx)

	if mock.ListPodsCalls != 1 {
		t.Errorf("ListPodsCalls = %d, want 1 (should cache)", mock.ListPodsCalls)
	}
}

func TestCachedGateway_ExpiresAfterTTL(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListPods(ctx)
	time.Sleep(150 * time.Millisecond)
	_, _ = c.ListPods(ctx)

	if mock.ListPodsCalls != 2 {
		t.Errorf("ListPodsCalls = %d, want 2 (TTL expired)", mock.ListPodsCalls)
	}
}

func TestCachedGateway_DeletePod_InvalidatesPodCache(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListPods(ctx)
	_ = c.DeletePod(ctx, "web-1")
	_, _ = c.ListPods(ctx)

	if mock.ListPodsCalls != 2 {
		t.Errorf("ListPodsCalls = %d, want 2 (cache invalidated by delete)", mock.ListPodsCalls)
	}
}

func TestCachedGateway_ScaleDeployment_InvalidatesCache(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListDeployments(ctx)
	_ = c.ScaleDeployment(ctx, "api", 3)
	_, _ = c.ListDeployments(ctx)

	if mock.ListDeploymentsCalls != 2 {
		t.Errorf("ListDeploymentsCalls = %d, want 2", mock.ListDeploymentsCalls)
	}
}

func TestCachedGateway_SetNamespace_InvalidatesAll(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListPods(ctx)
	_, _ = c.ListDeployments(ctx)
	c.SetNamespace("other")
	_, _ = c.ListPods(ctx)
	_, _ = c.ListDeployments(ctx)

	if mock.ListPodsCalls != 2 {
		t.Errorf("ListPodsCalls = %d, want 2", mock.ListPodsCalls)
	}
	if mock.ListDeploymentsCalls != 2 {
		t.Errorf("ListDeploymentsCalls = %d, want 2", mock.ListDeploymentsCalls)
	}
}

func TestCachedGateway_WatchPassesThrough(t *testing.T) {
	c, mock := newTestCache()
	ch := make(chan domain.WatchEvent)
	mock.WatchPodsCh = ch

	got, err := c.WatchPods(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != ch {
		t.Error("WatchPods should pass through to delegate")
	}
}

func TestCachedGateway_Reconnect_InvalidatesAll(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListPods(ctx)
	_ = c.Reconnect()
	_, _ = c.ListPods(ctx)

	if mock.ListPodsCalls != 2 {
		t.Errorf("ListPodsCalls = %d, want 2", mock.ListPodsCalls)
	}
	if mock.ReconnectCalls != 1 {
		t.Errorf("ReconnectCalls = %d, want 1", mock.ReconnectCalls)
	}
}

func TestCachedGateway_CachesNamespaces(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListNamespaces(ctx)
	_, _ = c.ListNamespaces(ctx)

	if mock.ListNamespacesCalls != 1 {
		t.Errorf("ListNamespacesCalls = %d, want 1", mock.ListNamespacesCalls)
	}
}

func TestCachedGateway_CachesEvents(t *testing.T) {
	c, mock := newTestCache()
	ctx := context.Background()

	_, _ = c.ListEvents(ctx)
	_, _ = c.ListEvents(ctx)

	if mock.ListEventsCalls != 1 {
		t.Errorf("ListEventsCalls = %d, want 1", mock.ListEventsCalls)
	}
}
