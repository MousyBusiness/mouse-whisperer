// Package spaces reads the Mission Control space layout via the native
// space-helper binary (build/space-helper, see native/space-helper).
package spaces

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Space is one Mission Control space in left-to-right order. Type follows
// SkyLight's convention: 0 = user desktop, 4 = fullscreen app.
type Space struct {
	ID   uint64 `json:"id"`
	Type int    `json:"type"`
}

// Snapshot is the space layout at one moment. IDs are stable while a space
// exists; positions shift as spaces are added, removed, or reordered.
type Snapshot struct {
	ActiveID uint64  `json:"active_id"`
	Spaces   []Space `json:"spaces"`
}

// Query runs the helper binary and parses its output.
func Query(helperPath string) (*Snapshot, error) {
	out, err := exec.Command(helperPath).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("space-helper: %s", ee.Stderr)
		}
		return nil, fmt.Errorf("space-helper: %w", err)
	}
	return Parse(out)
}

// Parse decodes helper output; separated from Query for testability.
func Parse(b []byte) (*Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("space-helper output: %w", err)
	}
	if len(s.Spaces) == 0 {
		return nil, fmt.Errorf("space-helper reported no spaces")
	}
	return &s, nil
}

// IndexOf returns the 1-based Mission Control position of the space with the
// given ID, or 0 if the space no longer exists.
func (s *Snapshot) IndexOf(id uint64) int {
	for i, sp := range s.Spaces {
		if sp.ID == id {
			return i + 1
		}
	}
	return 0
}

// ActiveIndex returns the 1-based position of the current space, or 0 if it
// could not be determined.
func (s *Snapshot) ActiveIndex() int {
	return s.IndexOf(s.ActiveID)
}
