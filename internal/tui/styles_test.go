package tui

import "testing"

func TestColorizeStatus(t *testing.T) {
	// Just verify it doesn't panic and returns non-empty for all known statuses
	statuses := []string{
		"Running", "Active",
		"Succeeded", "Completed",
		"Pending", "ContainerCreating", "Terminating",
		"Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff",
		"ErrImagePull", "OOMKilled", "Init:Error", "Init:CrashLoopBackOff",
		"UnknownStatus",
		"",
	}

	for _, s := range statuses {
		t.Run(s, func(t *testing.T) {
			result := colorizeStatus(s)
			if s != "" && result == "" {
				t.Errorf("colorizeStatus(%q) returned empty string", s)
			}
		})
	}
}
