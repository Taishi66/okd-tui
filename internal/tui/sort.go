package tui

import (
	"sort"
	"strings"

	"github.com/jclamy/okd-tui/internal/domain"
)

// SortColumn identifies a column for sorting.
type SortColumn int

const (
	SortNone SortColumn = iota
	// Pods
	SortPodName
	SortPodStatus
	SortPodRestarts
	SortPodAge
	// Deployments
	SortDepName
	SortDepReady
	SortDepAge
	// Events
	SortEvtType
	SortEvtAge
	SortEvtCount
)

// SortState holds the current sort configuration for a view.
type SortState struct {
	Column    SortColumn
	Ascending bool
}

// SortColumnLabel returns a display label for the sort column.
func (s SortState) Label() string {
	switch s.Column {
	case SortPodName, SortDepName:
		return "NAME"
	case SortPodStatus:
		return "STATUS"
	case SortPodRestarts:
		return "RESTARTS"
	case SortPodAge, SortDepAge, SortEvtAge:
		return "AGE"
	case SortDepReady:
		return "READY"
	case SortEvtType:
		return "TYPE"
	case SortEvtCount:
		return "COUNT"
	default:
		return ""
	}
}

// SortIndicator returns ▲ or ▼ for the active sort column header.
func SortIndicator(header string, state SortState) string {
	label := state.Label()
	if label == "" || !strings.EqualFold(header, label) {
		return header
	}
	if state.Ascending {
		return header + " ▲"
	}
	return header + " ▼"
}

// --- Pod sorting ---

func SortPods(pods []domain.PodInfo, state SortState) []domain.PodInfo {
	if state.Column == SortNone || len(pods) == 0 {
		return pods
	}
	sorted := make([]domain.PodInfo, len(pods))
	copy(sorted, pods)
	sort.SliceStable(sorted, func(i, j int) bool {
		var less bool
		switch state.Column {
		case SortPodName:
			less = strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		case SortPodStatus:
			less = strings.ToLower(sorted[i].Status) < strings.ToLower(sorted[j].Status)
		case SortPodRestarts:
			less = sorted[i].Restarts < sorted[j].Restarts
		case SortPodAge:
			less = sorted[i].CreatedAt.After(sorted[j].CreatedAt) // newest first for ascending
		default:
			return false
		}
		if !state.Ascending {
			return !less
		}
		return less
	})
	return sorted
}

func NextPodSort(current SortColumn) SortColumn {
	switch current {
	case SortNone:
		return SortPodName
	case SortPodName:
		return SortPodStatus
	case SortPodStatus:
		return SortPodRestarts
	case SortPodRestarts:
		return SortPodAge
	default:
		return SortNone
	}
}

// --- Deployment sorting ---

func SortDeployments(deps []domain.DeploymentInfo, state SortState) []domain.DeploymentInfo {
	if state.Column == SortNone || len(deps) == 0 {
		return deps
	}
	sorted := make([]domain.DeploymentInfo, len(deps))
	copy(sorted, deps)
	sort.SliceStable(sorted, func(i, j int) bool {
		var less bool
		switch state.Column {
		case SortDepName:
			less = strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		case SortDepReady:
			less = sorted[i].Available < sorted[j].Available
		case SortDepAge:
			less = sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		default:
			return false
		}
		if !state.Ascending {
			return !less
		}
		return less
	})
	return sorted
}

func NextDeploymentSort(current SortColumn) SortColumn {
	switch current {
	case SortNone:
		return SortDepName
	case SortDepName:
		return SortDepReady
	case SortDepReady:
		return SortDepAge
	default:
		return SortNone
	}
}

// --- Event sorting ---

func SortEvents(events []domain.EventInfo, state SortState) []domain.EventInfo {
	if state.Column == SortNone || len(events) == 0 {
		return events
	}
	sorted := make([]domain.EventInfo, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		var less bool
		switch state.Column {
		case SortEvtType:
			less = sorted[i].Type < sorted[j].Type
		case SortEvtAge:
			less = sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		case SortEvtCount:
			less = sorted[i].Count < sorted[j].Count
		default:
			return false
		}
		if !state.Ascending {
			return !less
		}
		return less
	})
	return sorted
}

func NextEventSort(current SortColumn) SortColumn {
	switch current {
	case SortNone:
		return SortEvtType
	case SortEvtType:
		return SortEvtAge
	case SortEvtAge:
		return SortEvtCount
	default:
		return SortNone
	}
}
