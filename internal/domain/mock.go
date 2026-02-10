package domain

import "context"

// MockGateway implements KubeGateway for testing.
type MockGateway struct {
	ContextVal   string
	ServerURLVal string
	NamespaceVal string

	Pods        []PodInfo
	Deployments []DeploymentInfo
	Namespaces  []NamespaceInfo
	Events      []EventInfo
	LogContent  string

	// Watch channels (inject from tests)
	WatchPodsCh        chan WatchEvent
	WatchDeploymentsCh chan WatchEvent
	WatchEventsCh      chan WatchEvent

	// YAML content
	PodYAML        string
	DeploymentYAML string

	// Error injection
	GetPodYAMLErr        error
	GetDeploymentYAMLErr error
	ListPodsErr          error
	ListDeploymentsErr  error
	ListNamespacesErr   error
	GetPodLogsErr       error
	DeletePodErr        error
	ScaleErr            error
	ReconnectErr        error
	WatchPodsErr        error
	WatchDeploymentsErr error
	ListEventsErr       error
	WatchEventsErr      error

	// Call tracking
	DeletedPod      string
	ScaledDep       string
	ScaledTo        int32
	ReconnectCalls  int
	LoggedContainer string
}

// Compile-time check.
var _ KubeGateway = (*MockGateway)(nil)

func (m *MockGateway) GetContext() string    { return m.ContextVal }
func (m *MockGateway) GetServerURL() string  { return m.ServerURLVal }
func (m *MockGateway) GetNamespace() string  { return m.NamespaceVal }
func (m *MockGateway) SetNamespace(ns string) { m.NamespaceVal = ns }

func (m *MockGateway) Reconnect() error {
	m.ReconnectCalls++
	return m.ReconnectErr
}

func (m *MockGateway) WatchPods(_ context.Context) (<-chan WatchEvent, error) {
	if m.WatchPodsErr != nil {
		return nil, m.WatchPodsErr
	}
	return m.WatchPodsCh, nil
}

func (m *MockGateway) WatchDeployments(_ context.Context) (<-chan WatchEvent, error) {
	if m.WatchDeploymentsErr != nil {
		return nil, m.WatchDeploymentsErr
	}
	return m.WatchDeploymentsCh, nil
}

func (m *MockGateway) ListPods(_ context.Context) ([]PodInfo, error) {
	if m.ListPodsErr != nil {
		return nil, m.ListPodsErr
	}
	return m.Pods, nil
}

func (m *MockGateway) GetPodLogs(_ context.Context, _ string, containerName string, _ int64, _ bool) (string, error) {
	m.LoggedContainer = containerName
	if m.GetPodLogsErr != nil {
		return "", m.GetPodLogsErr
	}
	return m.LogContent, nil
}

func (m *MockGateway) DeletePod(_ context.Context, podName string) error {
	m.DeletedPod = podName
	return m.DeletePodErr
}

func (m *MockGateway) ListDeployments(_ context.Context) ([]DeploymentInfo, error) {
	if m.ListDeploymentsErr != nil {
		return nil, m.ListDeploymentsErr
	}
	return m.Deployments, nil
}

func (m *MockGateway) ScaleDeployment(_ context.Context, name string, replicas int32) error {
	m.ScaledDep = name
	m.ScaledTo = replicas
	return m.ScaleErr
}

func (m *MockGateway) ListEvents(_ context.Context) ([]EventInfo, error) {
	if m.ListEventsErr != nil {
		return nil, m.ListEventsErr
	}
	return m.Events, nil
}

func (m *MockGateway) WatchEvents(_ context.Context) (<-chan WatchEvent, error) {
	if m.WatchEventsErr != nil {
		return nil, m.WatchEventsErr
	}
	return m.WatchEventsCh, nil
}

func (m *MockGateway) ListNamespaces(_ context.Context) ([]NamespaceInfo, error) {
	if m.ListNamespacesErr != nil {
		return nil, m.ListNamespacesErr
	}
	return m.Namespaces, nil
}

func (m *MockGateway) GetPodYAML(_ context.Context, _ string) (string, error) {
	if m.GetPodYAMLErr != nil {
		return "", m.GetPodYAMLErr
	}
	return m.PodYAML, nil
}

func (m *MockGateway) GetDeploymentYAML(_ context.Context, _ string) (string, error) {
	if m.GetDeploymentYAMLErr != nil {
		return "", m.GetDeploymentYAMLErr
	}
	return m.DeploymentYAML, nil
}
