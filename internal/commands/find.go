package commands

import (
	"strings"

	"github.com/mousybusiness/mouse_whisperer/pkg/symbols"
	"github.com/mousybusiness/mouse_whisperer/types"
	"github.com/lithammer/fuzzysearch/fuzzy"
	log "github.com/sirupsen/logrus"
)

// Find looks up a command by its trigger phrase (ignores state filtering)
func (cl *CommandLookup) Find(trigger TriggerPhrase) *CommandEntry {
	entry, _ := cl.Triggers.Get(TriggerPhrase(trigger))
	return entry
}

// FindWithState looks up a command by its trigger phrase and filters by active states
func (cl *CommandLookup) FindWithState(trigger TriggerPhrase, activeStates map[types.ExecutionState]bool) *CommandEntry {
	entry, _ := cl.Triggers.Get(TriggerPhrase(trigger))
	if entry == nil {
		return nil
	}
	// Check state constraints
	if !entry.Command.IsActiveInState(activeStates) {
		return nil
	}
	return entry
}

// FindSplitWithState looks up a command with fuzzy matching for transcription errors.
// It handles cases where symbols are attached to trigger words (e.g., "put_" instead of "put underscore").
// Returns the matched entry and any extracted suffix (e.g., "_" from "put_").
func (cl *CommandLookup) FindSplitWithState(trigger TriggerPhrase, activeStates map[types.ExecutionState]bool) (*CommandEntry, TrailerText) {

	// First try exact match
	if entry := cl.FindWithState(trigger, activeStates); entry != nil {
		return entry, ""
	}

	// Try stripping trailing symbols/punctuation and matching
	cleaned, suffix := symbols.SplitTriggerAndSymbol(string(trigger))
	if cleaned != string(trigger) && cleaned != "" {
		if entry := cl.FindWithState(TriggerPhrase(cleaned), activeStates); entry != nil {
			return entry, TrailerText(suffix)
		}
	}

	// Try prefix matching - find triggers that the input starts with
	// This handles "put_underscore" matching trigger "put"
	var bestMatch *CommandEntry
	var bestTrigger string
	for t, entry := range cl.Triggers.AllFromFront() {
		if strings.HasPrefix(string(trigger), string(t)) && entry.Command.IsActiveInState(activeStates) {
			// Prefer longer trigger matches
			if len(t) > len(bestTrigger) {
				bestMatch = entry
				bestTrigger = string(t)
			}
		}
	}
	if bestMatch != nil {
		return bestMatch, TrailerText(strings.TrimPrefix(string(trigger), bestTrigger))
	}

	return nil, ""
}

// MinFuzzyConfidence is the minimum similarity ratio (0-1) required for fuzzy matching.
// Higher values = stricter matching. 0.75 means 75% similarity required.
const MinFuzzyConfidence = 0.75

// FindRankedFuzzy performs ranked fuzzy matching against all triggers using Levenshtein distance.
// Returns the best match only if it meets the minimum confidence threshold.
// This is a last-resort fallback for transcription errors like "okay" -> "ok", "top" -> "tap".
func (cl *CommandLookup) FindRankedFuzzy(incorrect IncorrectTriggerPhrase, activeStates map[types.ExecutionState]bool) (*CommandEntry, TriggerPhrase) {
	input := string(incorrect)
	log.Infof("++++++++ FindRankedFuzzy +++++++++: %s", input)
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return nil, ""
	}

	// Collect all active triggers
	triggers := make(map[string]*CommandEntry)

	for t, entry := range cl.Triggers.AllFromFront() {
		if entry.Command.IsActiveInState(activeStates) {
			triggers[string(t)] = entry
		}
	}

	if len(triggers) == 0 {
		return nil, ""
	}

	// Find best match using Levenshtein distance
	var bestTrigger string
	bestDistance := -1

	for t, c := range triggers {
		// Disallow fuzzy for the command
		if c.Command.DisableFuzzy {
			continue
		}

		lct := strings.ToLower(t)

		// First letter must match to avoid conflicts like "hide" matching "show"
		if len(input) > 0 && len(lct) > 0 && input[0] != lct[0] {
			continue
		}

		// Try matching first word against trigger
		dist := fuzzy.LevenshteinDistance(input, lct)
		// log.Debugf("Levenshtein distance between '%s' and '%s': %d", input, triggerLower, dist)

		// Also try full input for multi-word triggers
		if strings.Contains(t, " ") {
			fullDist := fuzzy.LevenshteinDistance(input, lct)
			if fullDist < dist {
				dist = fullDist
			}
		}

		if bestDistance < 0 || dist < bestDistance {
			bestDistance = dist
			bestTrigger = t
		}
	}

	if bestTrigger == "" {
		return nil, ""
	}

	// For short words (3-4 letters), only allow 1 character difference
	triggerLen := len(bestTrigger)
	if triggerLen >= 3 && triggerLen <= 4 {
		if bestDistance > 1 {
			log.Debugf("Best distance is greater than 1 for short words, returning nil")
			return nil, ""
		}
	} else {
		// For longer words, use similarity ratio
		maxLen := max(len(input), len(bestTrigger))
		if maxLen == 0 {
			return nil, ""
		}

		// Similarity = 1 - (distance / maxLen)
		similarity := 1.0 - (float64(bestDistance) / float64(maxLen))

		// Require high confidence
		if similarity < MinFuzzyConfidence {
			log.Debugf("Similarity is less than %f, returning nil", MinFuzzyConfidence)
			return nil, ""
		}
	}

	log.Debugf("Best trigger: %s", bestTrigger)

	entry := triggers[bestTrigger]
	if entry == nil {
		log.Errorf("Entry is nil, returning nil")
		return nil, ""
	}

	return entry, TriggerPhrase(bestTrigger)
}

// HasLongerTrigger reports whether a state-active trigger extends the given
// phrase at a word boundary, e.g. phrase "left" and trigger "left word".
// Used by streaming evaluation to hold a match until the next word decides.
func (cl *CommandLookup) HasLongerTrigger(phrase TriggerPhrase, activeStates map[types.ExecutionState]bool) bool {
	prefix := string(phrase) + " "
	for t, entry := range cl.Triggers.AllFromFront() {
		if strings.HasPrefix(string(t), prefix) && entry.Command.IsActiveInState(activeStates) {
			return true
		}
	}
	return false
}

// FindWithPrefix finds commands that start with the given prefix (for chained commands)
func (cl *CommandLookup) FindWithPrefix(prefix string) []*CommandEntry {
	var matches []*CommandEntry
	for t, entry := range cl.Triggers.AllFromFront() {
		if t.HasPrefix(prefix) {
			matches = append(matches, entry)
		}
	}
	return matches
}
