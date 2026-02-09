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

// WatchEventType represents the type of a Kubernetes watch event.
type WatchEventType string

const (
	EventAdded    WatchEventType = "ADDED"
	EventModified WatchEventType = "MODIFIED"
	EventDeleted  WatchEventType = "DELETED"
)

// EventInfo represents a Kubernetes event for display in the TUI.
type EventInfo struct {
	Type      string // "Normal" or "Warning"
	Reason    string
	Message   string
	Object    string // e.g. "pod/web-1"
	Namespace string
	Age       string
	Count     int32
}

// WatchEvent carries a single watch event for the TUI to merge into its state.
type WatchEvent struct {
	Type       WatchEventType
	Resource   string // "pod", "deployment", "event"
	Pod        *PodInfo
	Deployment *DeploymentInfo
	Event      *EventInfo
}
