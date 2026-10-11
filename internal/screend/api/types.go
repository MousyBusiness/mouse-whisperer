// Package api holds the JSON wire types for the screend HTTP API, shared by
// the server and the main-app client. Plain JSON by design — the buf/proto
// pipeline is not involved.
package api

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"time"
)

// Provider names shared by the server and its clients.
const (
	ProviderOCR = "ocr"
	ProviderAX  = "ax"
)

// Rect is a target rectangle in global logical (point) coordinates.
// image.Rectangle marshals as nested Min/Max points; this stays flat.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r Rect) ToImageRect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

func FromImageRect(r image.Rectangle) Rect {
	return Rect{X: r.Min.X, Y: r.Min.Y, W: r.Dx(), H: r.Dy()}
}

type TargetsRequest struct {
	Provider          string `json:"provider,omitempty"` // default "ocr"
	Query             string `json:"query"`
	Region            *Rect  `json:"region,omitempty"` // logical coords; nil = full screen
	MaxResults        int    `json:"max_results,omitempty"`
	MaxAgeMs          *int   `json:"max_age_ms,omitempty"` // nil = server TTL, 0 = force fresh capture
	IncludeScreenshot bool   `json:"include_screenshot,omitempty"`
	Display           int    `json:"display,omitempty"`
}

type Target struct {
	ID         int     `json:"id"` // == index == overlay hex label
	Label      string  `json:"label"`
	Rect       Rect    `json:"rect"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
	Match      string  `json:"match"`
	LineText   string  `json:"line_text,omitempty"`
}

type ScreenInfo struct {
	Width   int     `json:"width"` // logical points
	Height  int     `json:"height"`
	Scale   float64 `json:"scale"`
	Display int     `json:"display"`
}

type TargetsResponse struct {
	Provider            string     `json:"provider"`
	SnapshotID          string     `json:"snapshot_id"`
	CapturedAt          time.Time  `json:"captured_at"`
	AgeMs               int64      `json:"age_ms"`
	FromCache           bool       `json:"from_cache"`
	Screen              ScreenInfo `json:"screen"`
	Targets             []Target   `json:"targets"`
	ScreenshotPNGBase64 string     `json:"screenshot_png_base64,omitempty"`
}

// DecodeScreenshot decodes the logical-size PNG captured with the snapshot.
func (r *TargetsResponse) DecodeScreenshot() (image.Image, error) {
	raw, err := base64.StdEncoding.DecodeString(r.ScreenshotPNGBase64)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(raw))
}

type PrewarmRequest struct {
	Provider string `json:"provider,omitempty"` // default "ocr"
	Display  int    `json:"display,omitempty"`
}

type PrewarmResponse struct {
	Status string `json:"status"` // warming | already_warming
}

type HealthResponse struct {
	Status        string   `json:"status"`
	Providers     []string `json:"providers"`
	HelperRunning bool     `json:"helper_running"`
	AXTrusted     *bool    `json:"ax_trusted,omitempty"` // nil until the helper has answered a ping
}

type ErrorResponse struct {
	Error string `json:"error"`
}
