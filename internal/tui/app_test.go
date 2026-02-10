package tui

import (
	"encoding/base64"
	"testing"

	"github.com/jclamy/okd-tui/internal/domain"
)

// --- truncate ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hell…"},
		{"maxLen 1", "hello", 1, "h"},
		{"maxLen 0", "hello", 0, ""},
		{"negative maxLen", "hello", -1, ""},
		{"empty string", "", 5, ""},
		{"unicode string", "héllo", 4, "hél…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("truncate(%q, %d) panicked: %v", tt.input, tt.maxLen, r)
				}
			}()
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// --- encodeBase64 ---

func TestEncodeBase64(t *testing.T) {
	tests := []string{
		"hello",
		"my-pod-name-12345",
		"a",
		"ab",
		"abc",
		"",
		"hello world with spaces",
		"special-chars_123.456",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := encodeBase64(input)
			want := base64.StdEncoding.EncodeToString([]byte(input))
			if got != want {
				t.Errorf("encodeBase64(%q) = %q, want %q (stdlib)", input, got, want)
			}
		})
	}
}

// --- View.String() ---

func TestViewString(t *testing.T) {
	tests := []struct {
		view View
		want string
	}{
		{ViewProjects, "PROJECTS"},
		{ViewPods, "PODS"},
		{ViewDeployments, "DEPLOYS"},
		{ViewLogs, "LOGS"},
		{ViewError, ""},
		{View(99), ""},
	}

	for _, tt := range tests {
		got := tt.view.String()
		if got != tt.want {
			t.Errorf("View(%d).String() = %q, want %q", tt.view, got, tt.want)
		}
	}
}

// --- Filtering ---

func TestFilteredPods(t *testing.T) {
	m := Model{
		pods: []domain.PodInfo{
			{Name: "api-server-abc", Status: "Running"},
			{Name: "worker-def", Status: "CrashLoopBackOff"},
			{Name: "redis-ghi", Status: "Running"},
			{Name: "api-gateway-jkl", Status: "Pending"},
		},
	}

	// No filter
	all := m.filteredPods()
	if len(all) != 4 {
		t.Errorf("no filter: got %d pods, want 4", len(all))
	}

	// Filter by name
	m.filter.SetValue("api")
	filtered := m.filteredPods()
	if len(filtered) != 2 {
		t.Errorf("filter 'api': got %d pods, want 2", len(filtered))
	}

	// Filter by status
	m.filter.SetValue("crash")
	filtered = m.filteredPods()
	if len(filtered) != 1 {
		t.Errorf("filter 'crash': got %d pods, want 1", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].Name != "worker-def" {
		t.Errorf("filter 'crash': got %q, want %q", filtered[0].Name, "worker-def")
	}

	// Filter with no match
	m.filter.SetValue("nonexistent")
	filtered = m.filteredPods()
	if len(filtered) != 0 {
		t.Errorf("filter 'nonexistent': got %d pods, want 0", len(filtered))
	}

	// Case insensitive
	m.filter.SetValue("API")
	filtered = m.filteredPods()
	if len(filtered) != 2 {
		t.Errorf("filter 'API' (case insensitive): got %d pods, want 2", len(filtered))
	}
}

func TestFilteredDeployments(t *testing.T) {
	m := Model{
		deployments: []domain.DeploymentInfo{
			{Name: "frontend"},
			{Name: "backend-api"},
			{Name: "worker"},
		},
	}

	m.filter.SetValue("end")
	filtered := m.filteredDeployments()
	if len(filtered) != 2 {
		t.Errorf("filter 'end': got %d, want 2 (frontend, backend-api)", len(filtered))
	}
}

func TestFilteredNamespaces(t *testing.T) {
	m := Model{
		namespaces: []domain.NamespaceInfo{
			{Name: "default"},
			{Name: "kube-system"},
			{Name: "my-app-prod"},
		},
	}

	m.filter.SetValue("kube")
	filtered := m.filteredNamespaces()
	if len(filtered) != 1 {
		t.Errorf("filter 'kube': got %d, want 1", len(filtered))
	}
}

// --- listLen ---

func TestListLen(t *testing.T) {
	m := Model{
		view: ViewPods,
		pods: []domain.PodInfo{{Name: "a"}, {Name: "b"}, {Name: "c"}},
	}
	if m.listLen() != 3 {
		t.Errorf("listLen (pods) = %d, want 3", m.listLen())
	}

	m.view = ViewDeployments
	m.deployments = []domain.DeploymentInfo{{Name: "x"}}
	if m.listLen() != 1 {
		t.Errorf("listLen (deployments) = %d, want 1", m.listLen())
	}

	m.view = ViewLogs
	if m.listLen() != 0 {
		t.Errorf("listLen (logs) = %d, want 0", m.listLen())
	}

	m.view = ViewError
	if m.listLen() != 0 {
		t.Errorf("listLen (error) = %d, want 0", m.listLen())
	}
}

// --- contentHeight ---

func TestContentHeight(t *testing.T) {
	m := Model{height: 30}
	ch := m.contentHeight()
	if ch != 24 {
		t.Errorf("contentHeight() = %d, want 24 (30 - 6)", ch)
	}

	// Small terminal
	m.height = 10
	ch = m.contentHeight()
	if ch != 4 {
		t.Errorf("contentHeight() = %d, want 4", ch)
	}

	// Very small terminal - should clamp to minimum 1
	m.height = 3
	ch = m.contentHeight()
	if ch != 1 {
		t.Errorf("contentHeight() = %d, want 1 (clamped minimum)", ch)
	}
}

// --- Tab cycling ---

func TestTabCycling(t *testing.T) {
	// ViewProjects=0, ViewPods=1, ViewDeployments=2, ViewEvents=3
	// (view + 1) % 4 should cycle correctly
	tests := []struct {
		current View
		want    View
	}{
		{ViewProjects, ViewPods},
		{ViewPods, ViewDeployments},
		{ViewDeployments, ViewEvents},
		{ViewEvents, ViewProjects},
	}

	for _, tt := range tests {
		next := View((tt.current + 1) % 4)
		if next != tt.want {
			t.Errorf("tab cycle from %d: got %d, want %d", tt.current, next, tt.want)
		}
	}
}

// --- NewModelWithError ---

func TestNewModelWithError(t *testing.T) {
	err := &domain.APIError{Type: domain.ErrNoKubeconfig, Message: "no kubeconfig"}
	m := NewModelWithError(err, nil)

	if m.view != ViewError {
		t.Errorf("view = %d, want ViewError", m.view)
	}
	if m.startupErr != err {
		t.Error("startupErr should be set")
	}
	if m.client != nil {
		t.Error("client should be nil")
	}

	// Init should return nil for error mode
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil for error mode")
	}
}

// --- Cursor boundary checks ---

func TestCursorBoundsEmptyList(t *testing.T) {
	m := Model{
		view: ViewPods,
		pods: []domain.PodInfo{}, // empty list
	}

	// After fix: cursor should stay at 0 on empty list
	maxIdx := m.listLen() - 1
	if maxIdx < 0 {
		maxIdx = 0
	}
	newCursor := min(m.cursor+1, maxIdx)
	if newCursor < 0 {
		t.Errorf("cursor should not go negative on empty list, got %d", newCursor)
	}
	if newCursor != 0 {
		t.Errorf("cursor on empty list should be 0, got %d", newCursor)
	}
}

// --- NewModel with mock ---

func TestNewModelWithMock(t *testing.T) {
	mock := &domain.MockGateway{
		ContextVal:   "test-context",
		ServerURLVal: "https://test:6443",
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "pod-1", Status: "Running"},
			{Name: "pod-2", Status: "Pending"},
		},
	}

	m := NewModel(mock, nil, nil)

	if m.client == nil {
		t.Fatal("client should not be nil")
	}

	// Verify the interface works
	if m.client.GetContext() != "test-context" {
		t.Errorf("GetContext() = %q, want %q", m.client.GetContext(), "test-context")
	}
	if m.client.GetNamespace() != "default" {
		t.Errorf("GetNamespace() = %q, want %q", m.client.GetNamespace(), "default")
	}
}
