// Package helper supervises the native screend-helper process and speaks its
// wire protocol. The helper does screen capture (ScreenCaptureKit), OCR
// (Vision framework), and accessibility queries (AXUIElement), reporting raw
// facts; all coordinate conversion happens in the Go providers.
//
// Protocol: NDJSON over stdin/stdout — one request line in, one response
// line out, correlated by id, strictly sequential. Helper stderr is logs.
package helper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const (
	OpPing       = "ping"
	OpCaptureOCR = "capture_ocr"
	OpOCRFile    = "ocr_file"
	OpAXQuery    = "ax_query"
)

type Request struct {
	ID           int64  `json:"id"`
	Op           string `json:"op"`
	Display      int    `json:"display,omitempty"`
	Level        string `json:"level,omitempty"` // "accurate" (default) | "fast"
	IncludeImage bool   `json:"include_image,omitempty"`
	Path         string `json:"path,omitempty"`         // ocr_file only
	Pid          int    `json:"pid,omitempty"`          // ax_query only; 0 = frontmost app
	MaxElements  int    `json:"max_elements,omitempty"` // ax_query walk cap; 0 = helper default
	MaxDepth     int    `json:"max_depth,omitempty"`    // ax_query depth cap; 0 = helper default
}

type Response struct {
	ID           int64        `json:"id"`
	OK           bool         `json:"ok"`
	Error        string       `json:"error,omitempty"`
	Meta         CaptureMeta  `json:"meta"`
	Observations []RawBox     `json:"observations"`
	Elements     []RawElement `json:"elements,omitempty"`   // ax_query only
	AXTrusted    *bool        `json:"ax_trusted,omitempty"` // ping only
	PNGBase64    string       `json:"png_base64,omitempty"` // logical-size PNG
}

// CaptureMeta reports facts about the captured frame (or, for ax_query, the
// queried display); the helper does no coordinate math beyond this.
type CaptureMeta struct {
	ImgW       int     `json:"img_w"` // physical pixels
	ImgH       int     `json:"img_h"`
	OriginXPts float64 `json:"origin_x_pts"` // display origin, global points, top-left origin
	OriginYPts float64 `json:"origin_y_pts"`
	WidthPts   float64 `json:"width_pts"` // display size in points
	HeightPts  float64 `json:"height_pts"`
}

// RawBox is one recognized word in Vision-normalized coordinates
// (bottom-left origin, relative to the captured image).
type RawBox struct {
	Text     string  `json:"text"`
	Line     int     `json:"line"` // observation index, groups words into lines
	LineText string  `json:"line_text"`
	Conf     float64 `json:"conf"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
}

// RawElement is one actionable UI element from an ax_query. Unlike RawBox,
// its frame is already global logical points with a top-left origin — the
// space robotgo and the overlay consume — so it must NOT go through the
// Vision coordinate conversion.
type RawElement struct {
	Role    string   `json:"role"`
	Subrole string   `json:"subrole,omitempty"`
	Title   string   `json:"title"`
	Enabled bool     `json:"enabled"`
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	W       float64  `json:"w"`
	H       float64  `json:"h"`
	Actions []string `json:"actions,omitempty"`
}

func writeRequest(w io.Writer, req Request) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// readResponse reads one NDJSON line. It must use bufio.Reader, not
// bufio.Scanner: base64 screenshot lines run to several MB, past Scanner's
// 64KB default limit.
func readResponse(r *bufio.Reader) (*Response, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read helper response: %w", err)
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("decode helper response: %w", err)
	}
	return &resp, nil
}
