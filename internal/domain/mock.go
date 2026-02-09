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
	LogContent  string

	// Error injection
	ListPodsErr        error
	ListDeploymentsErr error
	ListNamespacesErr  error
	GetPodLogsErr      error
	DeletePodErr       error
	ScaleErr           error
	ReconnectErr       error

	// Call tracking
	DeletedPod    string
	ScaledDep     string
	ScaledTo      int32
	ReconnectCalls int
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

func (m *MockGateway) ListPods(_ context.Context) ([]PodInfo, error) {
	if m.ListPodsErr != nil {
		return nil, m.ListPodsErr
	}
	return m.Pods, nil
}

func (m *MockGateway) GetPodLogs(_ context.Context, _ string, _ int64, _ bool) (string, error) {
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

func (m *MockGateway) ListNamespaces(_ context.Context) ([]NamespaceInfo, error) {
	if m.ListNamespacesErr != nil {
		return nil, m.ListNamespacesErr
	}
	return m.Namespaces, nil
}
