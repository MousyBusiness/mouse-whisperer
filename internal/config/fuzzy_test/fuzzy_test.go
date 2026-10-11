package fuzzy

import (
	"testing"

	"github.com/elliotchance/orderedmap/v3"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	_ "github.com/mousybusiness/mouse_whisperer/internal/state" // import for side effects (state constants)
	"github.com/mousybusiness/mouse_whisperer/pkg/symbols"
	"github.com/mousybusiness/mouse_whisperer/types"
)

// createTestLookup creates a CommandLookup with the given triggers for testing
func createTestLookup(triggers []string) *commands.CommandLookup {
	lookup := &commands.CommandLookup{
		Triggers: orderedmap.NewOrderedMap[commands.TriggerPhrase, *commands.CommandEntry](),
	}

	for i, trigger := range triggers {
		entry := &commands.CommandEntry{
			Category: "test",
			Name:     commands.CommandName(trigger),
			Command: &commands.Command{
				Triggers: []string{trigger},
			},
		}
		// Use index to ensure unique keys
		_ = i
		lookup.Triggers.Set(commands.TriggerPhrase(trigger), entry)
	}

	return lookup
}

func TestFindRankedFuzzy_CommonTranscriptionErrors(t *testing.T) {
	// Common triggers that exist in the system
	triggers := []string{
		"tap",
		"click",
		"scroll",
		"keep all",
		"put",
		"comma",
		"ok",
		"up",
		"down",
		"left",
		"right",
		"copy",
		"paste",
		"undo",
		"enter",
		"escape",
		"delete",
		"select",
		"save",
	}

	lookup := createTestLookup(triggers)

	tests := []struct {
		name          string
		input         string
		wantTrigger   string
		shouldMatch   bool
		minConfidence float64 // for documentation
	}{
		// Common transcription errors that SHOULD match
		{
			name:        "top -> tap (common vowel swap)",
			input:       "top",
			wantTrigger: "tap",
			shouldMatch: true,
		},
		{
			name:        "tapped -> tap (extra suffix)",
			input:       "tapped",
			wantTrigger: "tap",
			shouldMatch: false, // distance 3, too far
		},
		{
			name:        "commas -> comma (extra s)",
			input:       "commas",
			wantTrigger: "comma",
			shouldMatch: true,
		},
		{
			name:        "put",
			input:       "put_",
			wantTrigger: "put",
			shouldMatch: true,
		},
		{
			name:        "delet -> delete (missing e)",
			input:       "delet",
			wantTrigger: "delete",
			shouldMatch: true,
		},
	}

	// Default state map (no special states active)
	defaultStates := make(map[types.ExecutionState]bool)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, matchedTrigger := lookup.FindRankedFuzzy(commands.IncorrectTriggerPhrase(tt.input), defaultStates)
			matchedTriggerStr := string(matchedTrigger)
			if tt.shouldMatch {
				if entry == nil {
					t.Errorf("FindRankedFuzzy(%q) returned nil, want match for %q", tt.input, tt.wantTrigger)
					return
				}
				if matchedTriggerStr != tt.wantTrigger {
					t.Errorf("FindRankedFuzzy(%q) = %q, want %q", tt.input, matchedTrigger, tt.wantTrigger)
				}
			} else {
				if entry != nil && tt.wantTrigger != "" && matchedTriggerStr == tt.wantTrigger {
					t.Errorf("FindRankedFuzzy(%q) matched %q, but should not have", tt.input, matchedTrigger)
				}
				// If wantTrigger is empty, we just want no good match
				if tt.wantTrigger == "" && entry != nil {
					// This is okay if it matched something else with low confidence
					// but we expected no match at all
				}
			}
		})
	}
}

func TestFindFuzzyWithState_SymbolAttached(t *testing.T) {
	triggers := []string{
		"put",
		"tap",
		"click",
		"scroll",
		"type",
	}

	lookup := createTestLookup(triggers)

	tests := []struct {
		name        string
		input       string
		wantTrigger string
		wantSuffix  string
		shouldMatch bool
	}{
		{
			name:        "put_ -> put with _ suffix",
			input:       "put_",
			wantTrigger: "put",
			wantSuffix:  "_",
			shouldMatch: true,
		},
		{
			name:        "put, -> put with , suffix",
			input:       "put,",
			wantTrigger: "put",
			wantSuffix:  ",",
			shouldMatch: true,
		},
		{
			name:        "put/ -> put with / suffix",
			input:       "put/",
			wantTrigger: "put",
			wantSuffix:  "/",
			shouldMatch: true,
		},
		{
			name:        "tap. -> tap with . suffix",
			input:       "tap.",
			wantTrigger: "tap",
			wantSuffix:  ".",
			shouldMatch: true,
		},
		{
			name:        "click! -> click with ! suffix",
			input:       "click!",
			wantTrigger: "click",
			wantSuffix:  "!",
			shouldMatch: true,
		},
		{
			name:        "scroll -> scroll (no suffix)",
			input:       "scroll",
			wantTrigger: "scroll",
			wantSuffix:  "",
			shouldMatch: true,
		},
		{
			name:        "type; -> type with ; suffix",
			input:       "type;",
			wantTrigger: "type",
			wantSuffix:  ";",
			shouldMatch: true,
		},
		{
			name:        "put_underscore -> put with _underscore suffix (prefix match)",
			input:       "put_underscore",
			wantTrigger: "put",
			wantSuffix:  "_underscore",
			shouldMatch: true,
		},
	}

	// Default state map (no special states active)
	defaultStates := make(map[types.ExecutionState]bool)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, suffix := lookup.FindSplitWithState(commands.TriggerPhrase(tt.input), defaultStates)

			if tt.shouldMatch {
				if entry == nil {
					t.Errorf("FindFuzzyWithState(%q) returned nil, want match", tt.input)
					return
				}
				if entry.Command.Triggers[0] != tt.wantTrigger {
					t.Errorf("FindFuzzyWithState(%q) matched %q, want %q", tt.input, entry.Command.Triggers[0], tt.wantTrigger)
				}
				if string(suffix) != tt.wantSuffix {
					t.Errorf("FindFuzzyWithState(%q) suffix = %q, want %q", tt.input, suffix, tt.wantSuffix)
				}
			} else {
				if entry != nil {
					t.Errorf("FindFuzzyWithState(%q) matched %q, but should not have", tt.input, entry.Command.Triggers[0])
				}
			}
		})
	}
}

func TestSplitTriggerAndSymbol(t *testing.T) {
	tests := []struct {
		input      string
		wantWord   string
		wantSymbol string
	}{
		{"put_", "put", "_"},
		{"put,", "put", ","},
		{"put.", "put", "."},
		{"put/", "put", "/"},
		{"scroll", "scroll", ""},
		{"tap!", "tap", "!"},
		{"type;:", "type", ";:"},
		{"_", "", "_"},
		{"___", "", "___"},
		{"a_b_c", "a_b_c", ""},
		{"test123", "test123", ""},
		{"test123_", "test123", "_"},
		{"hello how are you.", "hello how are you", "."},
		{"", "", ""},
		{"!!!", "", "!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			word, symbol := symbols.SplitTriggerAndSymbol(tt.input)
			if word != tt.wantWord {
				t.Errorf("SplitTriggerAndSymbol(%q) word = %q, want %q", tt.input, word, tt.wantWord)
			}
			if symbol != tt.wantSymbol {
				t.Errorf("SplitTriggerAndSymbol(%q) symbol = %q, want %q", tt.input, symbol, tt.wantSymbol)
			}
		})
	}
}

func TestNormalizeSymbolInput(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Literal symbols -> word names
		{",", "comma"},
		{"_", "underscore"},
		{".", "dot"},
		{";", "semicolon"},
		{"!", "exclamation"},
		{"@", "at"},
		{"#", "hash"},
		{"/", "slash"},
		{"\\", "backslash"},

		// Word with trailing symbol (transcription artifact)
		{"comma,", "comma"},
		{"dot.", "dot"},

		// Already correct
		{"comma", "comma"},
		{"underscore", "underscore"},
		{"dot", "dot"},

		// Unknown should pass through
		{"foobar", "foobar"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := symbols.NormalizeSymbolInput(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeSymbolInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
