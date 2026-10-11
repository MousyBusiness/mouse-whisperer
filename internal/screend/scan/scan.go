package scan

import (
	"context"
	"image"
	"time"
)

// Provider turns a display into a Snapshot and resolves queries against it.
// "ocr" matches literal text; future providers (e.g. a vision LLM) may
// interpret the query semantically but return the same Target shape.
type Provider interface {
	Name() string
	Snapshot(ctx context.Context, display int) (*Snapshot, error)
	Query(ctx context.Context, snap *Snapshot, query string, opts QueryOptions) ([]Target, error)
}

// Snapshot is one captured + indexed frame of a display.
type Snapshot struct {
	ID            string
	CapturedAt    time.Time
	Display       int
	Screen        ScreenInfo
	Items         []Item // word-level items in logical coords; may be empty for non-OCR providers
	ScreenshotPNG []byte // logical-size PNG of the captured frame
}

// ScreenInfo describes the captured display in logical points.
type ScreenInfo struct {
	Width   int
	Height  int
	Scale   float64
	Display int
}

// Item is a single recognized word in global logical (point) coordinates.
type Item struct {
	Text       string
	Line       int    // groups words belonging to the same recognized line
	LineText   string // full text of that line
	Confidence float64
	Rect       image.Rectangle
}

// Target is a clickable match returned to callers.
type Target struct {
	Label      string
	Rect       image.Rectangle
	Confidence float64
	Source     string // provider name, stamped by the provider
	Match      string // exact | prefix | substring | phrase | fuzzy | all
	LineText   string
}

// QueryOptions narrows and bounds a query.
type QueryOptions struct {
	Region     *image.Rectangle // logical coords; nil = whole snapshot
	MaxResults int              // 0 = DefaultMaxResults
}

// DefaultMaxResults caps targets so overlay hex labels stay short.
const DefaultMaxResults = 64
