package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (c *Client) GetPodYAML(ctx context.Context, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(c.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", classifyError(err, c.serverURL)
	}
	pod.ManagedFields = nil
	data, err := yaml.Marshal(pod)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) GetDeploymentYAML(ctx context.Context, name string) (string, error) {
	dep, err := c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", classifyError(err, c.serverURL)
	}
	dep.ManagedFields = nil
	data, err := yaml.Marshal(dep)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
