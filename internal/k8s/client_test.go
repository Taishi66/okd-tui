package k8s

import (
	"errors"
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/jclamy/okd-tui/internal/domain"
)

func TestClassifyError_Nil(t *testing.T) {
	err := classifyError(nil, "https://api.cluster:6443")
	if err != nil {
		t.Errorf("classifyError(nil) = %v, want nil", err)
	}
}

func TestClassifyError_401(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: http.StatusUnauthorized},
	}
	err := classifyError(k8sErr, "https://api.cluster:6443")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrTokenExpired {
		t.Errorf("Type = %v, want ErrTokenExpired", apiErr.Type)
	}
	if apiErr.Unwrap() != k8sErr {
		t.Error("Unwrap should return original error")
	}
}

func TestClassifyError_403(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Code:    http.StatusForbidden,
			Message: "pods is forbidden",
		},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrForbidden {
		t.Errorf("Type = %v, want ErrForbidden", apiErr.Type)
	}
}

func TestClassifyError_404(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Code:    http.StatusNotFound,
			Message: "pod not found",
		},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrNotFound {
		t.Errorf("Type = %v, want ErrNotFound", apiErr.Type)
	}
}

func TestClassifyError_409(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: http.StatusConflict},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrConflict {
		t.Errorf("Type = %v, want ErrConflict", apiErr.Type)
	}
}

func TestClassifyError_429(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: http.StatusTooManyRequests},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrRateLimited {
		t.Errorf("Type = %v, want ErrRateLimited", apiErr.Type)
	}
}

func TestClassifyError_500(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: 500},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrServerError {
		t.Errorf("Type = %v, want ErrServerError", apiErr.Type)
	}
}

func TestClassifyError_502(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: 502},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrServerError {
		t.Errorf("Type = %v, want ErrServerError (got %v for 502)", apiErr.Type, apiErr.Type)
	}
}

func TestClassifyError_TLS(t *testing.T) {
	tests := []string{
		"x509: certificate signed by unknown authority",
		"tls: handshake failure",
		"certificate is not valid",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			err := classifyError(errors.New(msg), "https://api:6443")
			var apiErr *domain.APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("expected APIError")
			}
			if apiErr.Type != domain.ErrTLS {
				t.Errorf("Type = %v, want ErrTLS for %q", apiErr.Type, msg)
			}
		})
	}
}

func TestClassifyError_Unreachable(t *testing.T) {
	tests := []string{
		"dial tcp 10.0.0.1:6443: i/o timeout",
		"dial tcp: lookup api.cluster: no such host",
		"connection refused",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			err := classifyError(errors.New(msg), "https://api:6443")
			var apiErr *domain.APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("expected APIError")
			}
			if apiErr.Type != domain.ErrUnreachable {
				t.Errorf("Type = %v, want ErrUnreachable for %q", apiErr.Type, msg)
			}
		})
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	err := classifyError(errors.New("some random error"), "")
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Type != domain.ErrUnknown {
		t.Errorf("Type = %v, want ErrUnknown", apiErr.Type)
	}
}

func TestClassifyError_401WithServerURL(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: http.StatusUnauthorized},
	}
	err := classifyError(k8sErr, "https://api.my-cluster.com:6443")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	// Message should contain the server URL for re-login
	if !containsSubstring(apiErr.Message, "api.my-cluster.com") {
		t.Errorf("401 message should contain server URL, got: %s", apiErr.Message)
	}
}

func TestClassifyError_401WithoutServerURL(t *testing.T) {
	k8sErr := &k8serrors.StatusError{
		ErrStatus: metav1.Status{Code: http.StatusUnauthorized},
	}
	err := classifyError(k8sErr, "")

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	// Should still have a usable message without URL
	if !containsSubstring(apiErr.Message, "oc login") {
		t.Errorf("401 message should contain 'oc login', got: %s", apiErr.Message)
	}
}

func TestAPIError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	apiErr := &domain.APIError{
		Type:    domain.ErrForbidden,
		Message: "access denied",
		Err:     inner,
	}

	if apiErr.Error() != "access denied" {
		t.Errorf("Error() = %q, want %q", apiErr.Error(), "access denied")
	}
	if apiErr.Unwrap() != inner {
		t.Error("Unwrap() should return inner error")
	}
}

func TestAPIError_UnwrapNil(t *testing.T) {
	apiErr := &domain.APIError{
		Type:    domain.ErrNoContext,
		Message: "no context",
	}
	if apiErr.Unwrap() != nil {
		t.Error("Unwrap() should return nil when Err is nil")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
