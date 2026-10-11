// Package vision is the OCR scan.Provider: it drives the native helper's
// ScreenCaptureKit capture + Vision text recognition through
// internal/screend/helper and converts the normalized observations into
// logical-coordinate scan items.
package vision

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

// doer is the slice of helper.Engine this provider needs; tests stub it.
type doer interface {
	Do(ctx context.Context, req helper.Request) (*helper.Response, error)
}

// OCRProvider is the text-matching scan.Provider: one helper round-trip per
// snapshot, pure ranking per query.
type OCRProvider struct {
	engine doer
	level  string
	// includeImage requests the logical-size screenshot PNG with every
	// capture. Off by default: rendering + base64 of a full screen costs
	// hundreds of ms per snapshot and nothing in the scan path consumes it.
	includeImage bool
	seq          atomic.Int64
}

func NewOCRProvider(engine *helper.Engine, level string, includeImage bool) *OCRProvider {
	return newOCRProvider(engine, level, includeImage)
}

func newOCRProvider(engine doer, level string, includeImage bool) *OCRProvider {
	if level == "" {
		// accurate, despite the name, is about recall here: fast mode found
		// 6 of 23 instances of a word on a real dark-theme editor screen and
		// its word count jitters between identical captures. The ~300ms it
		// adds runs during speech (prewarm), so nothing user-visible pays.
		level = "accurate"
	}
	return &OCRProvider{engine: engine, level: level, includeImage: includeImage}
}

func (p *OCRProvider) Name() string { return "ocr" }

func (p *OCRProvider) Snapshot(ctx context.Context, display int) (*scan.Snapshot, error) {
	resp, err := p.engine.Do(ctx, helper.Request{
		Op:           helper.OpCaptureOCR,
		Display:      display,
		Level:        p.level,
		IncludeImage: p.includeImage,
	})
	if err != nil {
		return nil, err
	}

	items := make([]scan.Item, len(resp.Observations))
	for i, ob := range resp.Observations {
		items[i] = scan.Item{
			Text:       ob.Text,
			Line:       ob.Line,
			LineText:   ob.LineText,
			Confidence: ob.Conf,
			Rect:       ToLogical(ob, resp.Meta),
		}
	}

	var pngBytes []byte
	if resp.PNGBase64 != "" {
		if pngBytes, err = base64.StdEncoding.DecodeString(resp.PNGBase64); err != nil {
			return nil, fmt.Errorf("decode helper png: %w", err)
		}
	}

	now := time.Now()
	return &scan.Snapshot{
		ID:         fmt.Sprintf("%d-%d", now.UnixMilli(), p.seq.Add(1)),
		CapturedAt: now,
		Display:    display,
		Screen: scan.ScreenInfo{
			Width:   round(resp.Meta.WidthPts),
			Height:  round(resp.Meta.HeightPts),
			Scale:   float64(resp.Meta.ImgW) / resp.Meta.WidthPts,
			Display: display,
		},
		Items:         items,
		ScreenshotPNG: pngBytes,
	}, nil
}

func (p *OCRProvider) Query(_ context.Context, snap *scan.Snapshot, query string, opts scan.QueryOptions) ([]scan.Target, error) {
	targets := scan.Rank(snap.Items, query, opts)
	for i := range targets {
		targets[i].Source = p.Name()
	}
	return targets, nil
}
