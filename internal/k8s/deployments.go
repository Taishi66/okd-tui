package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Taishi66/okd-tui/internal/domain"
)

func (c *Client) ListDeployments(ctx context.Context) ([]domain.DeploymentInfo, error) {
	depList, err := c.clientset.AppsV1().Deployments(c.namespace).List(ctx, metav1.ListOptions{
		Limit: 500,
	})
	if err != nil {
		return nil, classifyError(err, c.serverURL)
	}

	deps := make([]domain.DeploymentInfo, 0, len(depList.Items))
	for _, dep := range depList.Items {
		var replicas int32
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		image := ""
		if len(dep.Spec.Template.Spec.Containers) > 0 {
			image = dep.Spec.Template.Spec.Containers[0].Image
		}
		deps = append(deps, domain.DeploymentInfo{
			Name:      dep.Name,
			Namespace: dep.Namespace,
			Ready:     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, replicas),
			Replicas:  replicas,
			Available: dep.Status.AvailableReplicas,
			Age:       formatAge(dep.CreationTimestamp.Time),
			Image:     image,
			CreatedAt: dep.CreationTimestamp.Time,
		})
	}
	return deps, nil
}

func (c *Client) WatchDeployments(ctx context.Context) (<-chan domain.WatchEvent, error) {
	watcher, err := c.clientset.AppsV1().Deployments(c.namespace).Watch(ctx, metav1.ListOptions{})
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
				dep, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					continue
				}
				var replicas int32
				if dep.Spec.Replicas != nil {
					replicas = *dep.Spec.Replicas
				}
				image := ""
				if len(dep.Spec.Template.Spec.Containers) > 0 {
					image = dep.Spec.Template.Spec.Containers[0].Image
				}
				info := domain.DeploymentInfo{
					Name:      dep.Name,
					Namespace: dep.Namespace,
					Ready:     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, replicas),
					Replicas:  replicas,
					Available: dep.Status.AvailableReplicas,
					Age:       formatAge(dep.CreationTimestamp.Time),
					Image:     image,
					CreatedAt: dep.CreationTimestamp.Time,
				}
				wType := domain.WatchEventType(string(event.Type))
				select {
				case ch <- domain.WatchEvent{Type: wType, Resource: "deployment", Deployment: &info}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

func (c *Client) ScaleDeployment(ctx context.Context, name string, replicas int32) error {
	if replicas < 0 {
		replicas = 0
	}
	scale, err := c.clientset.AppsV1().Deployments(c.namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return classifyError(err, c.serverURL)
	}
	scale.Spec.Replicas = replicas
	_, err = c.clientset.AppsV1().Deployments(c.namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return classifyError(err, c.serverURL)
}
