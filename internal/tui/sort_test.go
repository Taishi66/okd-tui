package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jclamy/okd-tui/internal/domain"
)

func TestSortPods_ByName(t *testing.T) {
	pods := []domain.PodInfo{
		{Name: "charlie"},
		{Name: "alpha"},
		{Name: "bravo"},
	}
	sorted := SortPods(pods, SortState{Column: SortPodName, Ascending: true})
	if sorted[0].Name != "alpha" || sorted[1].Name != "bravo" || sorted[2].Name != "charlie" {
		t.Errorf("sorted = %v", names(sorted))
	}
}

func TestSortPods_ByNameDescending(t *testing.T) {
	pods := []domain.PodInfo{
		{Name: "alpha"},
		{Name: "charlie"},
		{Name: "bravo"},
	}
	sorted := SortPods(pods, SortState{Column: SortPodName, Ascending: false})
	if sorted[0].Name != "charlie" {
		t.Errorf("first = %q, want charlie", sorted[0].Name)
	}
}

func TestSortPods_ByStatus(t *testing.T) {
	pods := []domain.PodInfo{
		{Name: "a", Status: "Running"},
		{Name: "b", Status: "CrashLoopBackOff"},
		{Name: "c", Status: "Pending"},
	}
	sorted := SortPods(pods, SortState{Column: SortPodStatus, Ascending: true})
	if sorted[0].Status != "CrashLoopBackOff" {
		t.Errorf("first status = %q, want CrashLoopBackOff", sorted[0].Status)
	}
}

func TestSortPods_ByRestarts(t *testing.T) {
	pods := []domain.PodInfo{
		{Name: "a", Restarts: 5},
		{Name: "b", Restarts: 0},
		{Name: "c", Restarts: 10},
	}
	sorted := SortPods(pods, SortState{Column: SortPodRestarts, Ascending: true})
	if sorted[0].Restarts != 0 || sorted[2].Restarts != 10 {
		t.Errorf("sorted restarts = %d, %d, %d", sorted[0].Restarts, sorted[1].Restarts, sorted[2].Restarts)
	}
}

func TestSortPods_ByAge(t *testing.T) {
	now := time.Now()
	pods := []domain.PodInfo{
		{Name: "old", CreatedAt: now.Add(-48 * time.Hour)},
		{Name: "new", CreatedAt: now.Add(-1 * time.Hour)},
		{Name: "mid", CreatedAt: now.Add(-24 * time.Hour)},
	}
	// Ascending age = newest first (most recent CreatedAt)
	sorted := SortPods(pods, SortState{Column: SortPodAge, Ascending: true})
	if sorted[0].Name != "new" {
		t.Errorf("first = %q, want new (newest)", sorted[0].Name)
	}
}

func TestSortPods_NoneReturnsOriginal(t *testing.T) {
	pods := []domain.PodInfo{
		{Name: "b"},
		{Name: "a"},
	}
	sorted := SortPods(pods, SortState{Column: SortNone})
	if sorted[0].Name != "b" {
		t.Error("SortNone should preserve original order")
	}
}

func TestSortDeployments_ByName(t *testing.T) {
	deps := []domain.DeploymentInfo{
		{Name: "zulu"},
		{Name: "alpha"},
	}
	sorted := SortDeployments(deps, SortState{Column: SortDepName, Ascending: true})
	if sorted[0].Name != "alpha" {
		t.Errorf("first = %q, want alpha", sorted[0].Name)
	}
}

func TestSortDeployments_ByReady(t *testing.T) {
	deps := []domain.DeploymentInfo{
		{Name: "a", Ready: "3/3", Available: 3},
		{Name: "b", Ready: "0/2", Available: 0},
		{Name: "c", Ready: "1/3", Available: 1},
	}
	sorted := SortDeployments(deps, SortState{Column: SortDepReady, Ascending: true})
	if sorted[0].Available != 0 {
		t.Errorf("first available = %d, want 0", sorted[0].Available)
	}
}

func TestNextPodSort_CyclesCorrectly(t *testing.T) {
	col := SortNone
	col = NextPodSort(col)
	if col != SortPodName {
		t.Errorf("after None: %v, want SortPodName", col)
	}
	col = NextPodSort(col)
	if col != SortPodStatus {
		t.Errorf("after Name: %v, want SortPodStatus", col)
	}
	col = NextPodSort(col)
	if col != SortPodRestarts {
		t.Errorf("after Status: %v, want SortPodRestarts", col)
	}
	col = NextPodSort(col)
	if col != SortPodAge {
		t.Errorf("after Restarts: %v, want SortPodAge", col)
	}
	col = NextPodSort(col)
	if col != SortNone {
		t.Errorf("after Age: %v, want SortNone (cycle)", col)
	}
}

func TestSortKey_ChangesSortColumn(t *testing.T) {
	mock := &domain.MockGateway{NamespaceVal: "default"}
	m := NewModel(mock, nil, nil)
	m.view = ViewPods
	m.pods = []domain.PodInfo{{Name: "a"}, {Name: "b"}}
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	um := updated.(Model)

	state := um.sortState[ViewPods]
	if state.Column != SortPodName {
		t.Errorf("after first 't': column = %v, want SortPodName", state.Column)
	}
}

func names(pods []domain.PodInfo) []string {
	n := make([]string, len(pods))
	for i, p := range pods {
		n[i] = p.Name
	}
	return n
}
