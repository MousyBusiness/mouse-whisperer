package spaces

import "testing"

// Sample captured from a real space-helper run: IDs are not sequential and
// do not match Mission Control positions.
const sample = `{"active_id":6,"spaces":[{"type":0,"id":217},{"id":4,"type":0},{"id":6,"type":0},{"id":3,"type":0},{"id":194,"type":0},{"id":5,"type":0}]}`

func TestParse(t *testing.T) {
	snap, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(snap.Spaces) != 6 {
		t.Fatalf("expected 6 spaces, got %d", len(snap.Spaces))
	}
	if got := snap.ActiveIndex(); got != 3 {
		t.Errorf("ActiveIndex = %d, want 3", got)
	}
	if got := snap.IndexOf(217); got != 1 {
		t.Errorf("IndexOf(217) = %d, want 1", got)
	}
	if got := snap.IndexOf(5); got != 6 {
		t.Errorf("IndexOf(5) = %d, want 6", got)
	}
	if got := snap.IndexOf(999); got != 0 {
		t.Errorf("IndexOf(999) = %d, want 0 for missing space", got)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte("not json")); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if _, err := Parse([]byte(`{"active_id":1,"spaces":[]}`)); err == nil {
		t.Error("expected error for empty space list")
	}
}
