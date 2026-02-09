package domain

import "context"

// ClusterInfo provides metadata about the current cluster connection.
type ClusterInfo interface {
	GetContext() string
	GetServerURL() string
	GetNamespace() string
	SetNamespace(ns string)
	Reconnect() error
}

// PodRepository provides access to pod operations.
type PodRepository interface {
	ListPods(ctx context.Context) ([]PodInfo, error)
	WatchPods(ctx context.Context) (<-chan WatchEvent, error)
	GetPodLogs(ctx context.Context, podName string, tailLines int64, previous bool) (string, error)
	DeletePod(ctx context.Context, podName string) error
}

// DeploymentRepository provides access to deployment operations.
type DeploymentRepository interface {
	ListDeployments(ctx context.Context) ([]DeploymentInfo, error)
	WatchDeployments(ctx context.Context) (<-chan WatchEvent, error)
	ScaleDeployment(ctx context.Context, name string, replicas int32) error
}

// NamespaceRepository provides access to namespace operations.
type NamespaceRepository interface {
	ListNamespaces(ctx context.Context) ([]NamespaceInfo, error)
}

// EventRepository provides access to event operations.
type EventRepository interface {
	ListEvents(ctx context.Context) ([]EventInfo, error)
	WatchEvents(ctx context.Context) (<-chan WatchEvent, error)
}

// KubeGateway is the primary port combining all cluster operations.
// The TUI depends on this interface, not on concrete implementations.
type KubeGateway interface {
	ClusterInfo
	PodRepository
	DeploymentRepository
	NamespaceRepository
	EventRepository
}
