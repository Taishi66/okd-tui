package cache

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/Taishi66/okd-tui/internal/config"
	"github.com/Taishi66/okd-tui/internal/domain"
)

type cacheEntry[T any] struct {
	data      T
	expiresAt time.Time
}

func (e *cacheEntry[T]) valid() bool {
	return time.Now().Before(e.expiresAt)
}

// CachedGateway decorates a KubeGateway with TTL-based caching for list operations.
type CachedGateway struct {
	delegate domain.KubeGateway
	cfg      config.CacheConfig
	mu       sync.RWMutex

	pods        *cacheEntry[[]domain.PodInfo]
	deployments *cacheEntry[[]domain.DeploymentInfo]
	namespaces  *cacheEntry[[]domain.NamespaceInfo]
	events      *cacheEntry[[]domain.EventInfo]
}

var _ domain.KubeGateway = (*CachedGateway)(nil)

func NewCachedGateway(delegate domain.KubeGateway, cfg config.CacheConfig) *CachedGateway {
	return &CachedGateway{
		delegate: delegate,
		cfg:      cfg,
	}
}

func (c *CachedGateway) invalidateAll() {
	c.pods = nil
	c.deployments = nil
	c.namespaces = nil
	c.events = nil
}

// --- ClusterInfo (pass-through) ---

func (c *CachedGateway) GetContext() string    { return c.delegate.GetContext() }
func (c *CachedGateway) GetServerURL() string  { return c.delegate.GetServerURL() }
func (c *CachedGateway) GetNamespace() string  { return c.delegate.GetNamespace() }

func (c *CachedGateway) SetNamespace(ns string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.delegate.SetNamespace(ns)
	c.invalidateAll()
}

func (c *CachedGateway) Reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.delegate.Reconnect()
	c.invalidateAll()
	return err
}

// --- Cached List operations ---

func (c *CachedGateway) ListPods(ctx context.Context) ([]domain.PodInfo, error) {
	c.mu.RLock()
	if c.pods != nil && c.pods.valid() {
		data := c.pods.data
		c.mu.RUnlock()
		return data, nil
	}
	c.mu.RUnlock()

	result, err := c.delegate.ListPods(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.pods = &cacheEntry[[]domain.PodInfo]{
		data:      result,
		expiresAt: time.Now().Add(c.cfg.PodsTTL),
	}
	c.mu.Unlock()
	return result, nil
}

func (c *CachedGateway) ListDeployments(ctx context.Context) ([]domain.DeploymentInfo, error) {
	c.mu.RLock()
	if c.deployments != nil && c.deployments.valid() {
		data := c.deployments.data
		c.mu.RUnlock()
		return data, nil
	}
	c.mu.RUnlock()

	result, err := c.delegate.ListDeployments(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.deployments = &cacheEntry[[]domain.DeploymentInfo]{
		data:      result,
		expiresAt: time.Now().Add(c.cfg.DeploymentsTTL),
	}
	c.mu.Unlock()
	return result, nil
}

func (c *CachedGateway) ListNamespaces(ctx context.Context) ([]domain.NamespaceInfo, error) {
	c.mu.RLock()
	if c.namespaces != nil && c.namespaces.valid() {
		data := c.namespaces.data
		c.mu.RUnlock()
		return data, nil
	}
	c.mu.RUnlock()

	result, err := c.delegate.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.namespaces = &cacheEntry[[]domain.NamespaceInfo]{
		data:      result,
		expiresAt: time.Now().Add(c.cfg.NamespacesTTL),
	}
	c.mu.Unlock()
	return result, nil
}

func (c *CachedGateway) ListEvents(ctx context.Context) ([]domain.EventInfo, error) {
	c.mu.RLock()
	if c.events != nil && c.events.valid() {
		data := c.events.data
		c.mu.RUnlock()
		return data, nil
	}
	c.mu.RUnlock()

	result, err := c.delegate.ListEvents(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.events = &cacheEntry[[]domain.EventInfo]{
		data:      result,
		expiresAt: time.Now().Add(c.cfg.EventsTTL),
	}
	c.mu.Unlock()
	return result, nil
}

// --- Mutations (pass-through + invalidate) ---

func (c *CachedGateway) DeletePod(ctx context.Context, podName string) error {
	err := c.delegate.DeletePod(ctx, podName)
	if err == nil {
		c.mu.Lock()
		c.pods = nil
		c.mu.Unlock()
	}
	return err
}

func (c *CachedGateway) ScaleDeployment(ctx context.Context, name string, replicas int32) error {
	err := c.delegate.ScaleDeployment(ctx, name, replicas)
	if err == nil {
		c.mu.Lock()
		c.deployments = nil
		c.mu.Unlock()
	}
	return err
}

// --- Pass-through (no caching) ---

func (c *CachedGateway) WatchPods(ctx context.Context) (<-chan domain.WatchEvent, error) {
	return c.delegate.WatchPods(ctx)
}

func (c *CachedGateway) WatchDeployments(ctx context.Context) (<-chan domain.WatchEvent, error) {
	return c.delegate.WatchDeployments(ctx)
}

func (c *CachedGateway) WatchEvents(ctx context.Context) (<-chan domain.WatchEvent, error) {
	return c.delegate.WatchEvents(ctx)
}

func (c *CachedGateway) GetPodLogs(ctx context.Context, podName, containerName string, tailLines int64, previous bool) (string, error) {
	return c.delegate.GetPodLogs(ctx, podName, containerName, tailLines, previous)
}

func (c *CachedGateway) GetPodYAML(ctx context.Context, podName string) (string, error) {
	return c.delegate.GetPodYAML(ctx, podName)
}

func (c *CachedGateway) GetDeploymentYAML(ctx context.Context, name string) (string, error) {
	return c.delegate.GetDeploymentYAML(ctx, name)
}

func (c *CachedGateway) BuildExecCmd(namespace, podName, containerName, shell string) (*exec.Cmd, error) {
	return c.delegate.BuildExecCmd(namespace, podName, containerName, shell)
}
