package domain

// ErrType classifies errors for the TUI to display appropriate messages.
type ErrType int

const (
	ErrUnknown       ErrType = iota
	ErrNoKubeconfig          // kubeconfig file not found
	ErrBadKubeconfig         // kubeconfig is malformed
	ErrNoContext             // no current context set
	ErrUnreachable           // cluster not reachable (timeout/DNS)
	ErrTokenExpired          // 401 Unauthorized
	ErrForbidden             // 403 Forbidden
	ErrNotFound              // 404 Not Found
	ErrConflict              // 409 Conflict
	ErrRateLimited           // 429 Too Many Requests
	ErrServerError           // 500+
	ErrTLS                   // TLS/cert error
)

// APIError wraps a K8s API error with classification.
type APIError struct {
	Type    ErrType
	Message string
	Err     error
}

func (e *APIError) Error() string {
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.Err
}
