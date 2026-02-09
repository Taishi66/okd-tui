package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jclamy/okd-tui/internal/domain"
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
		})
	}
	return deps, nil
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
