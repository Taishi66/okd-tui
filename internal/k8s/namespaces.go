package k8s

import (
	"context"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jclamy/okd-tui/internal/domain"
)

func (c *Client) ListNamespaces(ctx context.Context) ([]domain.NamespaceInfo, error) {
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		Limit: 500,
	})
	if err != nil {
		return nil, classifyError(err, c.serverURL)
	}

	namespaces := make([]domain.NamespaceInfo, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, domain.NamespaceInfo{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
			Age:    formatAge(ns.CreationTimestamp.Time),
		})
	}
	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})
	return namespaces, nil
}
