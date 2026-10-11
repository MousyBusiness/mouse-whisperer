package scan

import (
	"image"
	"testing"
)

func item(text string, line int, lineText string, x, y, w, h int) Item {
	return Item{Text: text, Line: line, LineText: lineText, Confidence: 0.9, Rect: image.Rect(x, y, x+w, y+h)}
}

type wantTarget struct {
	label string
	match string
	rect  image.Rectangle
}

func TestRank(t *testing.T) {
	cases := []struct {
		name  string
		items []Item
		query string
		opts  QueryOptions
		want  []wantTarget
	}{
		{
			name: "exact multi-hit in reading order",
			items: []Item{
				item("foo", 0, "foo handler", 10, 10, 30, 14),
				item("handler", 0, "foo handler", 300, 10, 60, 14),
				item("handler", 0, "handler bar", 10, 12, 60, 14),
				item("Handler", 1, "Handler baz", 10, 50, 60, 14),
			},
			query: "handler",
			want: []wantTarget{
				{"handler", "exact", image.Rect(10, 12, 70, 26)},
				{"handler", "exact", image.Rect(300, 10, 360, 24)},
				{"Handler", "exact", image.Rect(10, 50, 70, 64)},
			},
		},
		{
			// prefix rects narrow to the queried span: OCR sees dotted
			// identifiers as one word, and the click must land on the part
			// the user actually named
			name: "exact ranks before prefix and prefix rects narrow to the query",
			items: []Item{
				item("handlers", 0, "handlers", 10, 0, 80, 14),
				item("handlerFunc", 1, "handlerFunc", 10, 30, 110, 14),
				item("handler_test", 2, "handler_test", 10, 60, 120, 14),
				item("handler", 3, "handler", 10, 100, 70, 14),
			},
			query: "handler",
			want: []wantTarget{
				{"handler", "exact", image.Rect(10, 100, 80, 114)},
				{"handlers", "prefix", image.Rect(10, 0, 80, 14)},
				{"handlerFunc", "prefix", image.Rect(10, 30, 80, 44)},
				{"handler_test", "prefix", image.Rect(10, 60, 80, 74)},
			},
		},
		{
			name: "substring narrows rect proportionally",
			items: []Item{
				// 19 runes over 190px = 10px/rune; "handler" starts at rune 11
				item("RegisterOCRHandlers", 0, "func RegisterOCRHandlers(", 100, 40, 190, 14),
			},
			query: "handler",
			want: []wantTarget{
				{"RegisterOCRHandlers", "substring", image.Rect(210, 40, 280, 54)},
			},
		},
		{
			name: "phrase joins consecutive words on one line",
			items: []Item{
				item("mouse", 3, "mouse whisperer app", 100, 200, 50, 14),
				item("whisperer", 3, "mouse whisperer app", 160, 200, 80, 14),
				item("app", 3, "mouse whisperer app", 250, 200, 30, 14),
			},
			query: "mouse whisperer",
			want: []wantTarget{
				{"mouse whisperer", "phrase", image.Rect(100, 200, 240, 214)},
			},
		},
		{
			name: "phrase does not span lines",
			items: []Item{
				item("mouse", 0, "mouse", 10, 10, 50, 14),
				item("whisperer", 1, "whisperer", 10, 40, 80, 14),
			},
			query: "mouse whisperer",
			want:  nil,
		},
		{
			name: "fuzzy fires only when no other tier matches",
			items: []Item{
				item("handler", 0, "handler", 10, 10, 60, 14),
			},
			query: "hdlr",
			want: []wantTarget{
				{"handler", "fuzzy", image.Rect(10, 10, 70, 24)},
			},
		},
		{
			name: "fuzzy suppressed by an exact hit",
			items: []Item{
				item("handler", 0, "handler", 10, 10, 60, 14),
				item("hzzandler", 1, "hzzandler", 10, 40, 80, 14), // fuzzy-matches "handler"
			},
			query: "handler",
			want: []wantTarget{
				{"handler", "exact", image.Rect(10, 10, 70, 24)},
			},
		},
		{
			name: "overlapping duplicates dedup to the first",
			items: []Item{
				item("handler", 0, "handler", 10, 10, 60, 14),
				item("handler", 0, "handler", 12, 10, 60, 14),
			},
			query: "handler",
			want: []wantTarget{
				{"handler", "exact", image.Rect(10, 10, 70, 24)},
			},
		},
		{
			name: "region filters items before matching",
			items: []Item{
				item("handler", 0, "handler", 10, 10, 60, 14),
				item("handler", 5, "handler", 500, 500, 60, 14),
			},
			query: "handler",
			opts:  QueryOptions{Region: rectPtr(image.Rect(0, 0, 100, 100))},
			want: []wantTarget{
				{"handler", "exact", image.Rect(10, 10, 70, 24)},
			},
		},
		{
			name: "max results caps in rank order",
			items: []Item{
				item("handler", 0, "handler", 10, 10, 60, 14),
				item("handler", 1, "handler", 10, 40, 60, 14),
				item("handler", 2, "handler", 10, 70, 60, 14),
			},
			query: "handler",
			opts:  QueryOptions{MaxResults: 2},
			want: []wantTarget{
				{"handler", "exact", image.Rect(10, 10, 70, 24)},
				{"handler", "exact", image.Rect(10, 40, 70, 54)},
			},
		},
		{
			name: "empty query returns everything in reading order",
			items: []Item{
				item("beta", 1, "beta", 10, 40, 40, 14),
				item("alpha", 0, "alpha", 10, 10, 50, 14),
				item("gamma", 0, "alpha gamma", 100, 10, 50, 14),
			},
			query: "",
			want: []wantTarget{
				{"alpha", "all", image.Rect(10, 10, 60, 24)},
				{"gamma", "all", image.Rect(100, 10, 150, 24)},
				{"beta", "all", image.Rect(10, 40, 50, 54)},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Rank(c.items, c.query, c.opts)
			if len(got) != len(c.want) {
				t.Fatalf("got %d targets, want %d: %+v", len(got), len(c.want), got)
			}
			for i, w := range c.want {
				if got[i].Label != w.label {
					t.Errorf("target %d label = %q, want %q", i, got[i].Label, w.label)
				}
				if got[i].Match != w.match {
					t.Errorf("target %d match = %q, want %q", i, got[i].Match, w.match)
				}
				if got[i].Rect != w.rect {
					t.Errorf("target %d rect = %v, want %v", i, got[i].Rect, w.rect)
				}
			}
		})
	}
}

func TestRankDefaultCap(t *testing.T) {
	var items []Item
	for i := range 70 {
		items = append(items, item("handler", i, "handler", 10, 10+i*20, 60, 14))
	}
	got := Rank(items, "handler", QueryOptions{})
	if len(got) != DefaultMaxResults {
		t.Fatalf("got %d targets, want %d", len(got), DefaultMaxResults)
	}
}

func rectPtr(r image.Rectangle) *image.Rectangle { return &r }
