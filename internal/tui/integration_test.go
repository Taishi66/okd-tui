package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/domain"
)

// --- helpers ---

func newTestModel() Model {
	mock := &domain.MockGateway{
		ContextVal:   "test-ctx",
		ServerURLVal: "https://api.test:6443",
		NamespaceVal: "default",
		Pods: []domain.PodInfo{
			{Name: "api-pod-1", Status: "Running", Ready: "1/1", Restarts: 0, Age: "2h"},
			{Name: "worker-pod-2", Status: "CrashLoopBackOff", Ready: "0/1", Restarts: 5, Age: "1d"},
			{Name: "redis-pod-3", Status: "Running", Ready: "1/1", Restarts: 0, Age: "5d"},
		},
		Deployments: []domain.DeploymentInfo{
			{Name: "api-deploy", Ready: "3/3", Replicas: 3, Available: 3, Age: "10d", Image: "api:v1"},
			{Name: "worker-deploy", Ready: "0/2", Replicas: 2, Available: 0, Age: "5d", Image: "worker:v1"},
		},
		Namespaces: []domain.NamespaceInfo{
			{Name: "default", Status: "Active", Age: "30d"},
			{Name: "kube-system", Status: "Active", Age: "30d"},
			{Name: "my-app-prod", Status: "Active", Age: "10d"},
		},
		LogContent: "2024-01-01 INFO starting\n2024-01-01 INFO ready",
	}

	factory := func() (domain.KubeGateway, error) {
		return mock, nil
	}

	m := NewModel(mock, factory, nil)
	m.width = 120
	m.height = 30
	return m
}

func mockOf(m Model) *domain.MockGateway {
	return m.client.(*domain.MockGateway)
}

// --- Update: WindowSizeMsg ---

func TestUpdateWindowSize(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	um := updated.(Model)
	if um.width != 200 || um.height != 50 {
		t.Errorf("WindowSizeMsg: got %dx%d, want 200x50", um.width, um.height)
	}
}

// --- Update: data loaded messages ---

func TestUpdatePodsLoaded(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.cursor = 5 // should reset to 0

	pods := []domain.PodInfo{{Name: "new-pod", Status: "Running"}}
	updated, _ := m.Update(podsLoadedMsg{items: pods})
	um := updated.(Model)

	if um.loading {
		t.Error("loading should be false after podsLoadedMsg")
	}
	if um.cursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", um.cursor)
	}
	if len(um.pods) != 1 || um.pods[0].Name != "new-pod" {
		t.Errorf("pods not updated correctly")
	}
	if um.disconnected {
		t.Error("disconnected should be false after successful load")
	}
}

func TestUpdateDeploymentsLoaded(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.view = ViewDeployments

	deps := []domain.DeploymentInfo{{Name: "dep-1"}, {Name: "dep-2"}}
	updated, _ := m.Update(deploymentsLoadedMsg{items: deps})
	um := updated.(Model)

	if um.loading {
		t.Error("loading should be false")
	}
	if len(um.deployments) != 2 {
		t.Errorf("deployments count = %d, want 2", len(um.deployments))
	}
}

func TestUpdateNamespacesLoaded(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.view = ViewProjects
	m.disconnected = true // should clear on load

	ns := []domain.NamespaceInfo{{Name: "ns-1"}}
	updated, _ := m.Update(namespacesLoadedMsg{items: ns})
	um := updated.(Model)

	if um.loading {
		t.Error("loading should be false")
	}
	if um.disconnected {
		t.Error("disconnected should clear after successful load")
	}
	if len(um.namespaces) != 1 {
		t.Errorf("namespaces count = %d, want 1", len(um.namespaces))
	}
}

func TestUpdateLogsLoaded(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.view = ViewLogs

	updated, _ := m.Update(logsLoadedMsg{content: "line1\nline2\nline3"})
	um := updated.(Model)

	if um.loading {
		t.Error("loading should be false")
	}
	if len(um.logState.lines) != 3 {
		t.Errorf("log lines = %d, want 3", len(um.logState.lines))
	}
}

func TestUpdateActionDone(t *testing.T) {
	m := newTestModel()
	m.loading = true

	updated, cmd := m.Update(actionDoneMsg{message: "Pod deleted"})
	um := updated.(Model)

	if um.loading {
		t.Error("loading should be false")
	}
	if um.toast.message != "Pod deleted" {
		t.Errorf("toast message = %q, want %q", um.toast.message, "Pod deleted")
	}
	if um.toast.level != toastSuccess {
		t.Errorf("toast level = %v, want toastSuccess", um.toast.level)
	}
	if cmd == nil {
		t.Error("cmd should not be nil (should schedule toast clear + reload)")
	}
}

func TestUpdateToastExpired(t *testing.T) {
	m := newTestModel()
	m.toast = newToast("hello", toastInfo)

	updated, _ := m.Update(toastExpiredMsg{})
	um := updated.(Model)

	if um.toast.message != "" {
		t.Error("toast should be cleared after toastExpiredMsg")
	}
}

// --- handleAPIError ---

func TestHandleAPIError_TokenExpired(t *testing.T) {
	m := newTestModel()
	err := &domain.APIError{Type: domain.ErrTokenExpired, Message: "Session expired"}

	updated, cmd := m.Update(apiErrMsg{err: err})
	um := updated.(Model)

	if !um.disconnected {
		t.Error("should be disconnected on ErrTokenExpired")
	}
	if um.toast.level != toastError {
		t.Error("toast should be error level")
	}
	if cmd != nil {
		t.Error("cmd should be nil (no auto-clear for token expired)")
	}
}

func TestHandleAPIError_Unreachable(t *testing.T) {
	m := newTestModel()
	err := &domain.APIError{Type: domain.ErrUnreachable, Message: "unreachable"}

	updated, _ := m.Update(apiErrMsg{err: err})
	um := updated.(Model)

	if !um.disconnected {
		t.Error("should be disconnected on ErrUnreachable")
	}
}

func TestHandleAPIError_Forbidden(t *testing.T) {
	m := newTestModel()
	err := &domain.APIError{Type: domain.ErrForbidden, Message: "forbidden"}

	updated, cmd := m.Update(apiErrMsg{err: err})
	um := updated.(Model)

	if um.disconnected {
		t.Error("should not be disconnected on ErrForbidden")
	}
	if um.toast.level != toastError {
		t.Error("toast should be error level")
	}
	if cmd == nil {
		t.Error("should schedule toast clear")
	}
}

func TestHandleAPIError_NotFound(t *testing.T) {
	m := newTestModel()
	err := &domain.APIError{Type: domain.ErrNotFound, Message: "not found"}

	updated, cmd := m.Update(apiErrMsg{err: err})
	um := updated.(Model)

	if um.toast.message != "not found" {
		t.Errorf("toast = %q, want 'not found'", um.toast.message)
	}
	if cmd == nil {
		t.Error("should schedule toast clear + reload")
	}
}

func TestHandleAPIError_NonAPIError(t *testing.T) {
	m := newTestModel()
	err := errors.New("some random error")

	updated, cmd := m.Update(apiErrMsg{err: err})
	um := updated.(Model)

	if um.toast.message != "some random error" {
		t.Errorf("toast = %q, want 'some random error'", um.toast.message)
	}
	if cmd == nil {
		t.Error("should schedule toast clear")
	}
}

// --- handleKey: navigation ---

func TestHandleKeyNavigation(t *testing.T) {
	m := newTestModel()
	m.pods = mockOf(m).Pods

	// Down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)
	if um.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", um.cursor)
	}

	// Down again
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um = updated.(Model)
	if um.cursor != 2 {
		t.Errorf("after jj: cursor = %d, want 2", um.cursor)
	}

	// Down past end (3 pods, max index = 2)
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um = updated.(Model)
	if um.cursor != 2 {
		t.Errorf("after jjj: cursor = %d, want 2 (clamped)", um.cursor)
	}

	// Up
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um = updated.(Model)
	if um.cursor != 1 {
		t.Errorf("after k: cursor = %d, want 1", um.cursor)
	}

	// Top
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	um = updated.(Model)
	if um.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", um.cursor)
	}

	// Bottom
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	um = updated.(Model)
	if um.cursor != 2 {
		t.Errorf("after G: cursor = %d, want 2", um.cursor)
	}
}

// --- handleKey: tab switching ---

func TestHandleKeyTabSwitch(t *testing.T) {
	m := newTestModel()

	// Switch to projects with '1'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	um := updated.(Model)
	if um.view != ViewProjects {
		t.Errorf("after '1': view = %d, want ViewProjects", um.view)
	}
	if !um.loading {
		t.Error("should be loading after tab switch")
	}
	if cmd == nil {
		t.Error("should return a load command")
	}

	// Switch to deployments with '3'
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	um = updated.(Model)
	if um.view != ViewDeployments {
		t.Errorf("after '3': view = %d, want ViewDeployments", um.view)
	}
}

// --- handleKey: quit ---

func TestHandleKeyQuit(t *testing.T) {
	m := newTestModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("q should return a quit command")
	}
}

func TestHandleKeyQuitFromLogs(t *testing.T) {
	m := newTestModel()
	m.view = ViewLogs
	m.prevView = ViewPods
	m.logState = logState{podName: "test", content: "data"}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	um := updated.(Model)

	if um.view != ViewPods {
		t.Errorf("q from logs should go to prevView, got %d", um.view)
	}
	if cmd != nil {
		t.Error("should not quit, just go back")
	}
}

// --- handleKey: escape ---

func TestHandleKeyEscape(t *testing.T) {
	m := newTestModel()
	m.toast = newToast("test", toastInfo)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um := updated.(Model)

	if um.toast.message != "" {
		t.Error("escape should clear toast")
	}
}

func TestHandleKeyEscapeFromLogs(t *testing.T) {
	m := newTestModel()
	m.view = ViewLogs
	m.prevView = ViewPods

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um := updated.(Model)

	if um.view != ViewPods {
		t.Errorf("esc from logs should go to prevView, got %d", um.view)
	}
}

// --- handleKey: filter ---

func TestHandleKeyFilter(t *testing.T) {
	m := newTestModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	um := updated.(Model)

	if !um.filtering {
		t.Error("/ should activate filtering")
	}
	if cmd == nil {
		t.Error("should return blink command")
	}
}

func TestHandleFilterInputEnter(t *testing.T) {
	m := newTestModel()
	m.filtering = true
	m.filter.SetValue("api")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)

	if um.filtering {
		t.Error("enter should deactivate filtering")
	}
	// Value should be preserved
	if um.filter.Value() != "api" {
		t.Errorf("filter value = %q, want 'api' (preserved on enter)", um.filter.Value())
	}
}

func TestHandleFilterInputEsc(t *testing.T) {
	m := newTestModel()
	m.filtering = true
	m.filter.SetValue("api")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um := updated.(Model)

	if um.filtering {
		t.Error("esc should deactivate filtering")
	}
	if um.filter.Value() != "" {
		t.Errorf("filter value = %q, want '' (cleared on esc)", um.filter.Value())
	}
}

// --- handleKey: error screen ---

func TestErrorScreenReconnect(t *testing.T) {
	called := false
	factory := func() (domain.KubeGateway, error) {
		called = true
		return &domain.MockGateway{
			ContextVal:   "new-ctx",
			NamespaceVal: "default",
		}, nil
	}

	m := NewModelWithError(errors.New("connection failed"), factory)
	m.width = 80
	m.height = 30

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	um := updated.(Model)

	if !called {
		t.Error("factory should be called on 'r'")
	}
	if um.view != ViewPods {
		t.Errorf("view should be ViewPods after reconnect, got %d", um.view)
	}
	if um.client == nil {
		t.Error("client should be set after reconnect")
	}
	if um.startupErr != nil {
		t.Error("startupErr should be nil after reconnect")
	}
	if cmd == nil {
		t.Error("should return load command")
	}
}

func TestErrorScreenReconnectFails(t *testing.T) {
	factory := func() (domain.KubeGateway, error) {
		return nil, errors.New("still broken")
	}

	m := NewModelWithError(errors.New("initial error"), factory)
	m.width = 80
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	um := updated.(Model)

	if um.view != ViewError {
		t.Error("should stay on error screen")
	}
	if um.startupErr.Error() != "still broken" {
		t.Errorf("startupErr = %q, want 'still broken'", um.startupErr.Error())
	}
}

func TestErrorScreenNoFactory(t *testing.T) {
	m := NewModelWithError(errors.New("error"), nil)
	m.width = 80
	m.height = 30

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	um := updated.(Model)

	if um.view != ViewError {
		t.Error("should stay on error screen when no factory")
	}
	if cmd != nil {
		t.Error("should return nil cmd")
	}
}

// --- switchView ---

func TestSwitchViewFromLogs(t *testing.T) {
	m := newTestModel()
	m.view = ViewLogs
	m.logState = logState{podName: "test", content: "data", lines: []string{"data"}}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	um := updated.(Model)

	if um.view != ViewPods {
		t.Errorf("view = %d, want ViewPods", um.view)
	}
	if um.logState.podName != "" {
		t.Error("logState should be cleared on view switch")
	}
	if cmd == nil {
		t.Error("should return load command")
	}
}

// --- loadCurrentView ---

func TestLoadCurrentViewPods(t *testing.T) {
	m := newTestModel()
	m.view = ViewPods

	cmd := m.loadCurrentView()
	if cmd == nil {
		t.Fatal("loadCurrentView should return a command for ViewPods")
	}

	// Execute the command to verify it calls the mock
	msg := cmd()
	loaded, ok := msg.(podsLoadedMsg)
	if !ok {
		// Could be apiErrMsg if error
		t.Fatalf("expected podsLoadedMsg, got %T", msg)
	}
	if len(loaded.items) != 3 {
		t.Errorf("loaded %d pods, want 3", len(loaded.items))
	}
}

func TestLoadCurrentViewDeployments(t *testing.T) {
	m := newTestModel()
	m.view = ViewDeployments

	cmd := m.loadCurrentView()
	msg := cmd()
	loaded, ok := msg.(deploymentsLoadedMsg)
	if !ok {
		t.Fatalf("expected deploymentsLoadedMsg, got %T", msg)
	}
	if len(loaded.items) != 2 {
		t.Errorf("loaded %d deployments, want 2", len(loaded.items))
	}
}

func TestLoadCurrentViewNamespaces(t *testing.T) {
	m := newTestModel()
	m.view = ViewProjects

	cmd := m.loadCurrentView()
	msg := cmd()
	loaded, ok := msg.(namespacesLoadedMsg)
	if !ok {
		t.Fatalf("expected namespacesLoadedMsg, got %T", msg)
	}
	if len(loaded.items) != 3 {
		t.Errorf("loaded %d namespaces, want 3", len(loaded.items))
	}
}

func TestLoadCurrentViewReturnsErrorMsg(t *testing.T) {
	m := newTestModel()
	m.view = ViewPods
	mockOf(m).ListPodsErr = &domain.APIError{Type: domain.ErrForbidden, Message: "forbidden"}

	cmd := m.loadCurrentView()
	msg := cmd()
	errMsg, ok := msg.(apiErrMsg)
	if !ok {
		t.Fatalf("expected apiErrMsg, got %T", msg)
	}
	var apiErr *domain.APIError
	if !errors.As(errMsg.err, &apiErr) {
		t.Fatal("expected domain.APIError")
	}
	if apiErr.Type != domain.ErrForbidden {
		t.Errorf("error type = %v, want ErrForbidden", apiErr.Type)
	}
}

func TestLoadCurrentViewLogs(t *testing.T) {
	m := newTestModel()
	m.view = ViewLogs

	cmd := m.loadCurrentView()
	if cmd != nil {
		t.Error("loadCurrentView should return nil for ViewLogs")
	}
}

// --- View rendering ---

func TestViewRendering(t *testing.T) {
	m := newTestModel()
	m.pods = mockOf(m).Pods

	output := m.View()
	if output == "" {
		t.Fatal("View() should return non-empty output")
	}

	// Should contain context bar
	if !containsStr(output, "OKD TUI") {
		t.Error("View should contain 'OKD TUI'")
	}
	if !containsStr(output, "test-ctx") {
		t.Error("View should contain context name")
	}

	// Should contain tabs
	if !containsStr(output, "Projects") {
		t.Error("View should contain Projects tab")
	}
	if !containsStr(output, "Pods") {
		t.Error("View should contain Pods tab")
	}
}

func TestViewRenderingZeroWidth(t *testing.T) {
	m := newTestModel()
	m.width = 0

	output := m.View()
	if output != "Chargement..." {
		t.Errorf("View with width=0 should return 'Chargement...', got %q", output)
	}
}

func TestViewRenderingErrorScreen(t *testing.T) {
	m := NewModelWithError(errors.New("test error"), nil)
	m.width = 80
	m.height = 30

	output := m.View()
	if !containsStr(output, "Erreur de connexion") {
		t.Error("error screen should contain 'Erreur de connexion'")
	}
	if !containsStr(output, "test error") {
		t.Error("error screen should contain the error message")
	}
	if !containsStr(output, "Réessayer") {
		t.Error("error screen should contain retry option")
	}
}

func TestViewRenderingDisconnected(t *testing.T) {
	m := newTestModel()
	m.pods = mockOf(m).Pods
	m.disconnected = true

	output := m.View()
	if !containsStr(output, "Connexion perdue") {
		t.Error("should show disconnected banner")
	}
}

func TestViewRenderingLoading(t *testing.T) {
	m := newTestModel()
	m.loading = true

	output := m.View()
	if !containsStr(output, "Chargement") {
		t.Error("should show loading message")
	}
}

// --- renderPodList ---

func TestRenderPodList(t *testing.T) {
	pods := []domain.PodInfo{
		{Name: "pod-a", Status: "Running", Ready: "1/1", Restarts: 0, Age: "2h"},
		{Name: "pod-b", Status: "Pending", Ready: "0/1", Restarts: 3, Age: "5m"},
	}

	// Wide terminal
	output := renderPodList(pods, 0, 120, 10)
	if !containsStr(output, "pod-a") {
		t.Error("should contain pod-a")
	}
	if !containsStr(output, "pod-b") {
		t.Error("should contain pod-b")
	}
	if !containsStr(output, "RESTARTS") {
		t.Error("wide terminal should show RESTARTS column")
	}

	// Narrow terminal
	output = renderPodList(pods, 0, 60, 10)
	if !containsStr(output, "pod-a") {
		t.Error("narrow: should contain pod-a")
	}
}

func TestRenderPodListEmpty(t *testing.T) {
	output := renderPodList(nil, 0, 80, 10)
	if !containsStr(output, "Aucun pod") {
		t.Error("empty list should show 'Aucun pod'")
	}
}

// --- renderDeploymentList ---

func TestRenderDeploymentList(t *testing.T) {
	deps := []domain.DeploymentInfo{
		{Name: "api", Ready: "3/3", Available: 3, Age: "10d", Image: "img:v1"},
	}

	// Wide
	output := renderDeploymentList(deps, 0, 130, 10)
	if !containsStr(output, "api") {
		t.Error("should contain deployment name")
	}
	if !containsStr(output, "IMAGE") {
		t.Error("very wide should show IMAGE column")
	}

	// Medium
	output = renderDeploymentList(deps, 0, 90, 10)
	if !containsStr(output, "AVAIL") {
		t.Error("medium width should show AVAIL column")
	}

	// Narrow
	output = renderDeploymentList(deps, 0, 60, 10)
	if !containsStr(output, "api") {
		t.Error("narrow: should still show name")
	}
}

func TestRenderDeploymentListEmpty(t *testing.T) {
	output := renderDeploymentList(nil, 0, 80, 10)
	if !containsStr(output, "Aucun deployment") {
		t.Error("empty list should show 'Aucun deployment'")
	}
}

// --- renderProjectList ---

func TestRenderProjectList(t *testing.T) {
	ns := []domain.NamespaceInfo{
		{Name: "default", Status: "Active", Age: "30d"},
		{Name: "staging", Status: "Active", Age: "10d"},
	}

	output := renderProjectList(ns, 0, 80, 10, "default")
	if !containsStr(output, ">") {
		t.Error("active namespace should have '>' marker")
	}
	if !containsStr(output, "staging") {
		t.Error("should contain staging namespace")
	}
}

func TestRenderProjectListEmpty(t *testing.T) {
	output := renderProjectList(nil, 0, 80, 10, "")
	if !containsStr(output, "Aucun projet") {
		t.Error("empty list should show 'Aucun projet'")
	}
}

// --- colorizeReady ---

func TestColorizeReady(t *testing.T) {
	tests := []struct {
		ready string
	}{
		{"3/3"}, // all ready
		{"0/3"}, // none ready
		{"1/3"}, // partial
		{"0/0"}, // zero
	}

	for _, tt := range tests {
		t.Run(tt.ready, func(t *testing.T) {
			result := colorizeReady(tt.ready)
			if result == "" {
				t.Errorf("colorizeReady(%q) returned empty string", tt.ready)
			}
		})
	}
}

// --- toast ---

func TestToast(t *testing.T) {
	to := newToast("test message", toastSuccess)
	if !to.isActive() {
		t.Error("new toast should be active")
	}
	if to.message != "test message" {
		t.Errorf("message = %q, want 'test message'", to.message)
	}

	rendered := to.render()
	if rendered == "" {
		t.Error("active toast should render non-empty")
	}
}

func TestToastInactive(t *testing.T) {
	to := toast{}
	if to.isActive() {
		t.Error("empty toast should be inactive")
	}
	if to.render() != "" {
		t.Error("inactive toast should render empty")
	}
}

func TestToastError(t *testing.T) {
	to := newToast("error!", toastError)
	rendered := to.render()
	if rendered == "" {
		t.Error("error toast should render non-empty")
	}
}

func TestToastInfo(t *testing.T) {
	to := newToast("info", toastInfo)
	rendered := to.render()
	// Info toast uses plain text (no style wrapper returns the message itself)
	if rendered == "" {
		t.Error("info toast should render non-empty")
	}
}

func TestScheduleToastClear(t *testing.T) {
	cmd := scheduleToastClear()
	if cmd == nil {
		t.Error("scheduleToastClear should return a command")
	}
}

// --- logHelpKeys ---

func TestLogHelpKeys(t *testing.T) {
	current := logHelpKeys(false)
	if !containsStr(current, "précédents") {
		t.Error("current logs help should mention 'précédents'")
	}

	previous := logHelpKeys(true)
	if !containsStr(previous, "courants") {
		t.Error("previous logs help should mention 'courants'")
	}
}

// --- helpKeys ---

func TestHelpKeys(t *testing.T) {
	if podHelpKeys() == "" {
		t.Error("podHelpKeys should not be empty")
	}
	if deploymentHelpKeys() == "" {
		t.Error("deploymentHelpKeys should not be empty")
	}
	if projectHelpKeys() == "" {
		t.Error("projectHelpKeys should not be empty")
	}
}

// --- renderStatusBar ---

func TestRenderStatusBar(t *testing.T) {
	m := newTestModel()
	m.pods = mockOf(m).Pods

	bar := m.renderStatusBar()
	if bar == "" {
		t.Error("status bar should not be empty")
	}
	if !containsStr(bar, "PODS") {
		t.Error("status bar should contain view name")
	}
	if !containsStr(bar, "3 items") {
		t.Error("status bar should contain item count")
	}
}

func TestRenderStatusBarNilClient(t *testing.T) {
	m := Model{view: ViewPods, width: 80}
	bar := m.renderStatusBar()
	// Should not panic with nil client
	if bar == "" {
		t.Error("status bar should not be empty even with nil client")
	}
}

// --- renderContextBar ---

func TestRenderContextBar(t *testing.T) {
	m := newTestModel()
	bar := m.renderContextBar()
	if !containsStr(bar, "test-ctx") {
		t.Error("context bar should contain context")
	}
	if !containsStr(bar, "default") {
		t.Error("context bar should contain namespace")
	}
}

func TestRenderContextBarNilClient(t *testing.T) {
	m := Model{width: 80}
	bar := m.renderContextBar()
	if !containsStr(bar, "OKD TUI") {
		t.Error("should still contain title with nil client")
	}
}

// --- renderTabs ---

func TestRenderTabs(t *testing.T) {
	m := newTestModel()
	m.view = ViewPods

	tabs := m.renderTabs()
	if !containsStr(tabs, "Pods") {
		t.Error("tabs should contain 'Pods'")
	}
	if !containsStr(tabs, "Projects") {
		t.Error("tabs should contain 'Projects'")
	}
	if !containsStr(tabs, "Deploys") {
		t.Error("tabs should contain 'Deploys'")
	}
}

// --- Init ---

func TestInitNormal(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a load command for normal mode")
	}
}
