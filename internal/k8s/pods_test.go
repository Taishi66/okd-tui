package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 48 * time.Hour, "2d"},
		{"year+", 400 * 24 * time.Hour, "1y35d"},
		{"zero", 0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Now().Add(-tt.duration)
			got := formatAge(ts)
			if got != tt.want {
				t.Errorf("formatAge(now - %v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestPodStatus(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1.Pod
		want string
	}{
		{
			"running pod",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
				},
			},
			"Running",
		},
		{
			"crashloop",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
					},
				},
			},
			"CrashLoopBackOff",
		},
		{
			"image pull backoff",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}},
					},
				},
			},
			"ImagePullBackOff",
		},
		{
			"init container error",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					InitContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}},
					},
				},
			},
			"Init:Error",
		},
		{
			"completed pod",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"}}},
					},
				},
			},
			"Completed",
		},
		{
			"pending no statuses",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			"Pending",
		},
		{
			"OOMKilled",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}}},
					},
				},
			},
			"OOMKilled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := podStatus(tt.pod)
			if got != tt.want {
				t.Errorf("podStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPodReadyCount(t *testing.T) {
	tests := []struct {
		name      string
		pod       corev1.Pod
		wantReady int
		wantTotal int
	}{
		{
			"all ready",
			corev1.Pod{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}, {Name: "b"}}},
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true}, {Ready: true},
				}},
			},
			2, 2,
		},
		{
			"partial ready",
			corev1.Pod{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}, {Name: "b"}, {Name: "c"}}},
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true}, {Ready: false}, {Ready: true},
				}},
			},
			2, 3,
		},
		{
			"no containers",
			corev1.Pod{},
			0, 0,
		},
		{
			"containers but no statuses yet",
			corev1.Pod{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}}},
			},
			0, 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready, total := podReadyCount(tt.pod)
			if ready != tt.wantReady || total != tt.wantTotal {
				t.Errorf("podReadyCount() = (%d, %d), want (%d, %d)", ready, total, tt.wantReady, tt.wantTotal)
			}
		})
	}
}

func TestPodToPodInfo(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "my-pod-abc123",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
		},
		Spec: corev1.PodSpec{
			NodeName:   "node-1",
			Containers: []corev1.Container{{Name: "main"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready:        true,
					RestartCount: 3,
					State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}

	info := podToPodInfo(pod)

	if info.Name != "my-pod-abc123" {
		t.Errorf("Name = %q, want %q", info.Name, "my-pod-abc123")
	}
	if info.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", info.Namespace, "default")
	}
	if info.Status != "Running" {
		t.Errorf("Status = %q, want %q", info.Status, "Running")
	}
	if info.Ready != "1/1" {
		t.Errorf("Ready = %q, want %q", info.Ready, "1/1")
	}
	if info.Restarts != 3 {
		t.Errorf("Restarts = %d, want %d", info.Restarts, 3)
	}
	if info.Node != "node-1" {
		t.Errorf("Node = %q, want %q", info.Node, "node-1")
	}
	if info.Age != "2h" {
		t.Errorf("Age = %q, want %q", info.Age, "2h")
	}
}
