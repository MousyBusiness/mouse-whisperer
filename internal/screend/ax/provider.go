// Package ax is the accessibility scan.Provider: it queries the frontmost
// app's UI element tree through the native helper and serves element titles
// as click targets — the fast path in front of OCR for apps with a real AX
// tree. No capture, no recognition: a snapshot is one tree walk.
package ax

import (
	"context"
	"fmt"
	"image"
	"math"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

// doer is the slice of helper.Engine this provider needs; tests stub it.
type doer interface {
	Do(ctx context.Context, req helper.Request) (*helper.Response, error)
}

type Provider struct {
	engine doer
	seq    atomic.Int64
}

func NewProvider(engine *helper.Engine) *Provider {
	return newProvider(engine)
}

func newProvider(engine doer) *Provider {
	return &Provider{engine: engine}
}

func (p *Provider) Name() string { return "ax" }

func (p *Provider) Snapshot(ctx context.Context, display int) (*scan.Snapshot, error) {
	resp, err := p.engine.Do(ctx, helper.Request{Op: helper.OpAXQuery, Display: display})
	if err != nil {
		return nil, err
	}

	var items []scan.Item
	for i, el := range resp.Elements {
		if !el.Enabled {
			// a disabled control is visible but not actionable; matching it
			// would click a dead button while OCR could find a live target
			continue
		}
		items = append(items, splitTitle(el, i)...)
	}

	now := time.Now()
	return &scan.Snapshot{
		ID:         fmt.Sprintf("%d-%d", now.UnixMilli(), p.seq.Add(1)),
		CapturedAt: now,
		Display:    display,
		Screen: scan.ScreenInfo{
			// AX frames are already logical points, so scale is 1 and the
			// screen size comes straight from the display bounds in meta
			Width:   round(resp.Meta.WidthPts),
			Height:  round(resp.Meta.HeightPts),
			Scale:   1,
			Display: display,
		},
		Items: items,
	}, nil
}

func (p *Provider) Query(_ context.Context, snap *scan.Snapshot, query string, opts scan.QueryOptions) ([]scan.Target, error) {
	targets := scan.Rank(snap.Items, query, opts)
	for i := range targets {
		targets[i].Source = p.Name()
	}
	return targets, nil
}

// splitTitle turns one element into per-word scan items, mirroring how the
// helper splits OCR lines into words: the ranker's phrase matching walks
// consecutive single-word items sharing a line index, so a multi-word title
// must arrive as a word run, not one item. Word rects are proportional
// slices of the element frame — approximate, but a phrase match unions them
// back to the full frame, and that union's center is what gets clicked.
// Untitled elements yield nothing: they cannot be spoken.
func splitTitle(el helper.RawElement, line int) []scan.Item {
	runes := []rune(el.Title)
	total := len(runes)
	if total == 0 {
		return nil
	}
	var items []scan.Item
	i := 0
	for i < total {
		for i < total && unicode.IsSpace(runes[i]) {
			i++
		}
		if i >= total {
			break
		}
		start := i
		for i < total && !unicode.IsSpace(runes[i]) {
			i++
		}
		x0 := el.X + el.W*float64(start)/float64(total)
		x1 := el.X + el.W*float64(i)/float64(total)
		items = append(items, scan.Item{
			Text:       string(runes[start:i]),
			Line:       line,
			LineText:   strings.TrimSpace(el.Title),
			Confidence: 1,
			Rect:       image.Rect(round(x0), round(el.Y), round(x1), round(el.Y+el.H)),
		})
	}
	return items
}

func round(f float64) int { return int(math.Round(f)) }
