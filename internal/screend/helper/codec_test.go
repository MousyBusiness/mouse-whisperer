package helper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteRequestSingleLine(t *testing.T) {
	var buf bytes.Buffer
	err := writeRequest(&buf, Request{ID: 7, Op: OpCaptureOCR, Level: "accurate", IncludeImage: true})
	if err != nil {
		t.Fatalf("writeRequest: %v", err)
	}
	s := buf.String()
	if !strings.HasSuffix(s, "\n") {
		t.Fatalf("request not newline-terminated: %q", s)
	}
	if strings.Count(s, "\n") != 1 {
		t.Fatalf("request spans multiple lines: %q", s)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		t.Fatalf("request line is not valid JSON: %v", err)
	}
	if decoded["op"] != "capture_ocr" || decoded["id"] != float64(7) {
		t.Fatalf("unexpected request payload: %v", decoded)
	}
}

func TestWriteRequestAXFields(t *testing.T) {
	var buf bytes.Buffer
	if err := writeRequest(&buf, Request{ID: 2, Op: OpAXQuery, Pid: 123, MaxElements: 500, MaxDepth: 10}); err != nil {
		t.Fatalf("writeRequest: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("request line is not valid JSON: %v", err)
	}
	if decoded["op"] != "ax_query" || decoded["pid"] != float64(123) ||
		decoded["max_elements"] != float64(500) || decoded["max_depth"] != float64(10) {
		t.Fatalf("unexpected request payload: %v", decoded)
	}
	// omitempty: an OCR request must not leak ax fields onto the wire
	buf.Reset()
	if err := writeRequest(&buf, Request{ID: 3, Op: OpCaptureOCR}); err != nil {
		t.Fatalf("writeRequest: %v", err)
	}
	for _, key := range []string{"pid", "max_elements", "max_depth"} {
		if strings.Contains(buf.String(), key) {
			t.Fatalf("ocr request leaks %q: %s", key, buf.String())
		}
	}
}

func TestReadResponseLargeLine(t *testing.T) {
	// a base64 screenshot line runs to several MB — must survive well past
	// bufio.Scanner's 64KB default
	resp := Response{ID: 3, OK: true, PNGBase64: strings.Repeat("A", 5<<20)}
	line, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	r := bufio.NewReader(bytes.NewReader(append(line, '\n')))
	got, err := readResponse(r)
	if err != nil {
		t.Fatalf("readResponse: %v", err)
	}
	if got.ID != 3 || !got.OK || len(got.PNGBase64) != 5<<20 {
		t.Fatalf("round-trip mismatch: id=%d ok=%v len=%d", got.ID, got.OK, len(got.PNGBase64))
	}
}

func TestReadResponseFields(t *testing.T) {
	line := `{"id":9,"ok":true,"meta":{"img_w":3000,"img_h":2000,"origin_x_pts":0,"origin_y_pts":0,"width_pts":1500,"height_pts":1000},"observations":[{"text":"handler","line":2,"line_text":"func handler()","conf":0.97,"x":0.1,"y":0.2,"w":0.05,"h":0.01}]}` + "\n"
	got, err := readResponse(bufio.NewReader(strings.NewReader(line)))
	if err != nil {
		t.Fatalf("readResponse: %v", err)
	}
	if got.Meta.ImgW != 3000 || got.Meta.WidthPts != 1500 {
		t.Fatalf("meta mismatch: %+v", got.Meta)
	}
	if len(got.Observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(got.Observations))
	}
	ob := got.Observations[0]
	if ob.Text != "handler" || ob.Line != 2 || ob.LineText != "func handler()" || ob.Conf != 0.97 || ob.X != 0.1 {
		t.Fatalf("observation mismatch: %+v", ob)
	}
}

func TestReadResponseElements(t *testing.T) {
	line := `{"id":4,"ok":true,"meta":{"img_w":1512,"img_h":982,"origin_x_pts":0,"origin_y_pts":0,"width_pts":1512,"height_pts":982},"ax_trusted":true,"elements":[{"role":"AXButton","subrole":"","title":"Save As","enabled":true,"x":100.5,"y":200,"w":80,"h":24,"actions":["AXPress"]}]}` + "\n"
	got, err := readResponse(bufio.NewReader(strings.NewReader(line)))
	if err != nil {
		t.Fatalf("readResponse: %v", err)
	}
	if got.AXTrusted == nil || !*got.AXTrusted {
		t.Fatalf("ax_trusted mismatch: %+v", got.AXTrusted)
	}
	if len(got.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(got.Elements))
	}
	el := got.Elements[0]
	if el.Role != "AXButton" || el.Title != "Save As" || !el.Enabled ||
		el.X != 100.5 || el.W != 80 || len(el.Actions) != 1 || el.Actions[0] != "AXPress" {
		t.Fatalf("element mismatch: %+v", el)
	}
}

func TestReadResponseErrors(t *testing.T) {
	if _, err := readResponse(bufio.NewReader(strings.NewReader("not json\n"))); err == nil {
		t.Fatal("malformed line should error")
	}
	if _, err := readResponse(bufio.NewReader(strings.NewReader(""))); err == nil {
		t.Fatal("EOF should error")
	}
	got, err := readResponse(bufio.NewReader(strings.NewReader(`{"id":1,"ok":false,"error":"no permission"}` + "\n")))
	if err != nil {
		t.Fatalf("ok:false is a valid protocol response, got err %v", err)
	}
	if got.OK || got.Error != "no permission" {
		t.Fatalf("error response mismatch: %+v", got)
	}
}
