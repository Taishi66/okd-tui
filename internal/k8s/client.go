package k8s

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jclamy/okd-tui/internal/domain"
)

// Client wraps the Kubernetes clientset and connection metadata.
// It implements domain.KubeGateway.
type Client struct {
	clientset      kubernetes.Interface
	config         *rest.Config
	kubeconfigPath string
	context        string
	serverURL      string
	namespace      string
}

// Compile-time check that Client implements domain.KubeGateway.
var _ domain.KubeGateway = (*Client)(nil)

// --- ClusterInfo implementation ---

func (c *Client) GetContext() string    { return c.context }
func (c *Client) GetServerURL() string  { return c.serverURL }
func (c *Client) GetNamespace() string  { return c.namespace }
func (c *Client) SetNamespace(ns string) { c.namespace = ns }

// NewClient creates a K8s client from kubeconfig.
func NewClient() (*Client, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, _ := os.UserHomeDir()
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, &domain.APIError{
			Type:    domain.ErrNoKubeconfig,
			Message: fmt.Sprintf("Aucun kubeconfig trouvé.\nConfigurez votre accès avec : oc login <cluster-url>\n\nCherché dans : %s", kubeconfigPath),
			Err:     err,
		}
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, &domain.APIError{
			Type:    domain.ErrBadKubeconfig,
			Message: fmt.Sprintf("Kubeconfig invalide : %v", err),
			Err:     err,
		}
	}

	if rawConfig.CurrentContext == "" {
		return nil, &domain.APIError{
			Type:    domain.ErrNoContext,
			Message: "Aucun contexte actif dans le kubeconfig.\nUtilisez : kubectl config use-context <ctx>",
		}
	}

	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, &domain.APIError{
			Type:    domain.ErrBadKubeconfig,
			Message: fmt.Sprintf("Impossible de créer la config client : %v", err),
			Err:     err,
		}
	}

	// Optimize for snappy TUI
	restConfig.QPS = 50
	restConfig.Burst = 100
	restConfig.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, &domain.APIError{
			Type:    domain.ErrUnknown,
			Message: fmt.Sprintf("Impossible de créer le client K8s : %v", err),
			Err:     err,
		}
	}

	namespace, _, _ := kubeConfig.Namespace()
	if namespace == "" {
		namespace = "default"
	}

	serverURL := ""
	if clusterInfo, ok := rawConfig.Clusters[rawConfig.Contexts[rawConfig.CurrentContext].Cluster]; ok {
		serverURL = clusterInfo.Server
	}

	return &Client{
		clientset:      clientset,
		config:         restConfig,
		kubeconfigPath: kubeconfigPath,
		context:        rawConfig.CurrentContext,
		serverURL:      serverURL,
		namespace:      namespace,
	}, nil
}

// Reconnect reloads the kubeconfig from disk and recreates the clientset.
func (c *Client) Reconnect() error {
	newClient, err := NewClient()
	if err != nil {
		return err
	}
	c.clientset = newClient.clientset
	c.config = newClient.config
	c.context = newClient.context
	c.serverURL = newClient.serverURL
	return nil
}

// TestConnection makes a lightweight API call to verify connectivity.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	return classifyError(err, c.serverURL)
}

// classifyError converts a raw K8s error into a domain.APIError.
func classifyError(err error, serverURL string) error {
	if err == nil {
		return nil
	}

	var statusErr *k8serrors.StatusError
	if errors.As(err, &statusErr) {
		code := statusErr.Status().Code
		switch {
		case code == http.StatusUnauthorized:
			loginCmd := "oc login"
			if serverURL != "" {
				loginCmd = fmt.Sprintf("oc login %s", serverURL)
			}
			return &domain.APIError{
				Type:    domain.ErrTokenExpired,
				Message: fmt.Sprintf("Session expirée. Reconnectez-vous :\n  %s\nPuis appuyez sur 'r' pour reconnecter", loginCmd),
				Err:     err,
			}
		case code == http.StatusForbidden:
			return &domain.APIError{
				Type:    domain.ErrForbidden,
				Message: statusErr.Status().Message,
				Err:     err,
			}
		case code == http.StatusNotFound:
			return &domain.APIError{
				Type:    domain.ErrNotFound,
				Message: statusErr.Status().Message,
				Err:     err,
			}
		case code == http.StatusConflict:
			return &domain.APIError{
				Type:    domain.ErrConflict,
				Message: "Conflit : la ressource a été modifiée. Réessayez.",
				Err:     err,
			}
		case code == http.StatusTooManyRequests:
			return &domain.APIError{
				Type:    domain.ErrRateLimited,
				Message: "Trop de requêtes. Pause 2s...",
				Err:     err,
			}
		case code >= 500:
			return &domain.APIError{
				Type:    domain.ErrServerError,
				Message: fmt.Sprintf("Erreur serveur (%d). Réessayez avec 'r'.", code),
				Err:     err,
			}
		}
	}

	errStr := err.Error()
	if strings.Contains(errStr, "x509") || strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") {
		return &domain.APIError{
			Type:    domain.ErrTLS,
			Message: fmt.Sprintf("Certificat TLS invalide pour %s.\nVérifiez votre kubeconfig.", serverURL),
			Err:     err,
		}
	}

	if strings.Contains(errStr, "dial tcp") || strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "i/o timeout") {
		return &domain.APIError{
			Type:    domain.ErrUnreachable,
			Message: fmt.Sprintf("Cluster injoignable : %s\n%v", serverURL, err),
			Err:     err,
		}
	}

	return &domain.APIError{
		Type:    domain.ErrUnknown,
		Message: err.Error(),
		Err:     err,
	}
}
