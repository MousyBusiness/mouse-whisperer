package voice

import "testing"

func TestIsAbortPhrase(t *testing.T) {
	aborts := []string{"abort", "Abort.", "cancel", "Cancel!", "nevermind", "Never mind."}
	for _, s := range aborts {
		if !isAbortPhrase(s) {
			t.Errorf("expected %q to abort", s)
		}
	}
	inputs := []string{"abort the mission", "I want to cancel my subscription", "hello there", ""}
	for _, s := range inputs {
		if isAbortPhrase(s) {
			t.Errorf("%q must be treated as input, not abort", s)
		}
	}
}
