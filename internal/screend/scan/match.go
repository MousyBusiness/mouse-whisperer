package scan

import (
	"image"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

var tierRank = map[string]int{"exact": 0, "prefix": 1, "substring": 2, "phrase": 3, "fuzzy": 4, "all": 5}

// Rank finds every instance of query among items, ordered for hex labeling:
// better match tier first, then reading order (top to bottom, left to right).
// Multi-word queries match phrases of consecutive items on the same line, so
// providers must supply a line's words adjacently and in visual order. Fuzzy
// matching is a fallback only — it never dilutes results from other tiers.
func Rank(items []Item, query string, opts QueryOptions) []Target {
	max := opts.MaxResults
	if max <= 0 {
		max = DefaultMaxResults
	}
	if opts.Region != nil {
		var filtered []Item
		for _, it := range items {
			if it.Rect.Overlaps(*opts.Region) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	words := strings.Fields(strings.ToLower(query))

	var targets []Target
	switch {
	case len(words) == 0:
		for _, it := range items {
			targets = append(targets, target(it, "all", it.Rect))
		}
	case len(words) == 1:
		targets = matchWord(items, words[0])
	default:
		targets = matchPhrase(items, words)
	}

	sort.SliceStable(targets, func(i, j int) bool {
		ti, tj := tierRank[targets[i].Match], tierRank[targets[j].Match]
		if ti != tj {
			return ti < tj
		}
		if !sameRow(targets[i].Rect, targets[j].Rect) {
			return centerY(targets[i].Rect) < centerY(targets[j].Rect)
		}
		if targets[i].Rect.Min.X != targets[j].Rect.Min.X {
			return targets[i].Rect.Min.X < targets[j].Rect.Min.X
		}
		return targets[i].Confidence > targets[j].Confidence
	})

	targets = dedup(targets)
	if len(targets) > max {
		targets = targets[:max]
	}
	return targets
}

func matchWord(items []Item, q string) []Target {
	qRunes := len([]rune(q))
	var targets []Target
	for _, it := range items {
		lower := strings.ToLower(it.Text)
		switch idx := strings.Index(lower, q); {
		case lower == q:
			targets = append(targets, target(it, "exact", it.Rect))
		case idx >= 0:
			// OCR sees dotted identifiers as one word; the box (and the
			// click center) must cover only the part the user named
			kind := "substring"
			if idx == 0 {
				kind = "prefix"
			}
			start := len([]rune(lower[:idx]))
			narrowed := narrowRect(it.Rect, start, qRunes, len([]rune(lower)))
			targets = append(targets, target(it, kind, narrowed))
		}
	}
	if len(targets) > 0 {
		return targets
	}
	for _, it := range items {
		if fuzzy.Match(q, strings.ToLower(it.Text)) {
			targets = append(targets, target(it, "fuzzy", it.Rect))
		}
	}
	return targets
}

func matchPhrase(items []Item, words []string) []Target {
	var targets []Target
	n := len(words)
	for i := 0; i+n <= len(items); i++ {
		if !phraseAt(items, i, words) {
			continue
		}
		run := items[i : i+n]
		rect := run[0].Rect
		conf := run[0].Confidence
		labels := make([]string, n)
		for j, it := range run {
			rect = rect.Union(it.Rect)
			conf = min(conf, it.Confidence)
			labels[j] = it.Text
		}
		t := target(run[0], "phrase", rect)
		t.Label = strings.Join(labels, " ")
		t.Confidence = conf
		targets = append(targets, t)
	}
	return targets
}

func phraseAt(items []Item, i int, words []string) bool {
	for j, w := range words {
		it := items[i+j]
		if it.Line != items[i].Line || strings.ToLower(it.Text) != w {
			return false
		}
	}
	return true
}

// narrowRect slices the horizontal span covered by a substring match,
// assuming uniform glyph width across the word.
func narrowRect(r image.Rectangle, start, count, total int) image.Rectangle {
	if total <= 0 || count <= 0 {
		return r
	}
	x0 := r.Min.X + start*r.Dx()/total
	x1 := x0 + count*r.Dx()/total
	return image.Rect(x0, r.Min.Y, x1, r.Max.Y)
}

// dedup drops a target when an already-accepted one with the same label
// covers >=60% of the smaller rect — Vision occasionally emits near-duplicate
// observations of the same word.
func dedup(ts []Target) []Target {
	var out []Target
	for _, t := range ts {
		dup := false
		for _, kept := range out {
			if !strings.EqualFold(kept.Label, t.Label) {
				continue
			}
			inter := kept.Rect.Intersect(t.Rect)
			smaller := min(area(kept.Rect), area(t.Rect))
			if smaller > 0 && area(inter)*10 >= smaller*6 {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, t)
		}
	}
	return out
}

// sameRow tolerates the few pixels of vertical jitter between word boxes of
// one visual line so reading order stays left-to-right within a row.
func sameRow(a, b image.Rectangle) bool {
	tol := min(a.Dy(), b.Dy()) / 2
	diff := centerY(a) - centerY(b)
	return diff <= tol && diff >= -tol
}

func centerY(r image.Rectangle) int { return r.Min.Y + r.Dy()/2 }

func area(r image.Rectangle) int { return r.Dx() * r.Dy() }

func target(it Item, match string, rect image.Rectangle) Target {
	return Target{Label: it.Text, Rect: rect, Confidence: it.Confidence, Match: match, LineText: it.LineText}
}
