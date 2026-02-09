package domain

// PodInfo represents a Kubernetes pod for display in the TUI.
type PodInfo struct {
	Name      string
	Namespace string
	Status    string
	Ready     string
	Restarts  int32
	Age       string
	Node      string
}

// DeploymentInfo represents a Kubernetes deployment for display in the TUI.
type DeploymentInfo struct {
	Name      string
	Namespace string
	Ready     string
	Replicas  int32
	Available int32
	Age       string
	Image     string
}

// NamespaceInfo represents a Kubernetes namespace for display in the TUI.
type NamespaceInfo struct {
	Name   string
	Status string
	Age    string
}
