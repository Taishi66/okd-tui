package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jclamy/okd-tui/internal/domain"
)

func (c *Client) ListPods(ctx context.Context) ([]domain.PodInfo, error) {
	podList, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		Limit: 500,
	})
	if err != nil {
		return nil, classifyError(err, c.serverURL)
	}

	pods := make([]domain.PodInfo, 0, len(podList.Items))
	for _, pod := range podList.Items {
		pods = append(pods, podToPodInfo(pod))
	}
	return pods, nil
}

func (c *Client) WatchPods(ctx context.Context) (<-chan domain.WatchEvent, error) {
	watcher, err := c.clientset.CoreV1().Pods(c.namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, classifyError(err, c.serverURL)
	}
	ch := make(chan domain.WatchEvent)
	go func() {
		defer close(ch)
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}
				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					continue
				}
				info := podToPodInfo(*pod)
				wType := domain.WatchEventType(string(event.Type))
				select {
				case ch <- domain.WatchEvent{Type: wType, Resource: "pod", Pod: &info}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

func (c *Client) GetPodLogs(ctx context.Context, podName, containerName string, tailLines int64, previous bool) (string, error) {
	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
		Previous:  previous,
	}
	if containerName != "" {
		opts.Container = containerName
	}
	result, err := c.clientset.CoreV1().Pods(c.namespace).GetLogs(podName, opts).Do(ctx).Raw()
	if err != nil {
		return "", classifyError(err, c.serverURL)
	}
	return string(result), nil
}

func (c *Client) DeletePod(ctx context.Context, podName string) error {
	err := c.clientset.CoreV1().Pods(c.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	return classifyError(err, c.serverURL)
}

func podToPodInfo(pod corev1.Pod) domain.PodInfo {
	status := podStatus(pod)
	ready, total := podReadyCount(pod)
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}

	// Build container info list
	statusMap := make(map[string]corev1.ContainerStatus)
	for _, cs := range pod.Status.ContainerStatuses {
		statusMap[cs.Name] = cs
	}
	containers := make([]domain.ContainerInfo, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		ci := domain.ContainerInfo{Name: c.Name}
		if cs, ok := statusMap[c.Name]; ok {
			ci.Ready = cs.Ready
			ci.State = containerState(cs)
		}
		containers = append(containers, ci)
	}

	return domain.PodInfo{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		Status:     status,
		Ready:      fmt.Sprintf("%d/%d", ready, total),
		Restarts:   restarts,
		Age:        formatAge(pod.CreationTimestamp.Time),
		Node:       pod.Spec.NodeName,
		Containers: containers,
		CreatedAt:  pod.CreationTimestamp.Time,
	}
}

func containerState(cs corev1.ContainerStatus) string {
	switch {
	case cs.State.Running != nil:
		return "running"
	case cs.State.Waiting != nil:
		return "waiting"
	case cs.State.Terminated != nil:
		return "terminated"
	default:
		return "unknown"
	}
}

func podStatus(pod corev1.Pod) string {
	// Check container statuses for more specific states
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason // CrashLoopBackOff, ImagePullBackOff, etc.
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}
	// Check init container statuses
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return "Init:" + cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return "Init:Error"
		}
	}
	return string(pod.Status.Phase)
}

func podReadyCount(pod corev1.Pod) (int, int) {
	total := len(pod.Spec.Containers)
	ready := 0
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return ready, total
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days > 365 {
			return fmt.Sprintf("%dy%dd", days/365, days%365)
		}
		return fmt.Sprintf("%dd", days)
	}
}
