package debuglog

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	// DefaultRoot is the default directory for debug logs and frames.
	DefaultRoot = "/tmp/mousewhisperer"
	// Window is the rolling time window for CSV rows.
	Window = 60 * time.Second
	// MaxPOIFrameDirs is how many poi_frames batch directories to keep (oldest removed).
	MaxPOIFrameDirs = 5
)

var (
	enabled bool
	root    string
	mu      sync.Mutex

	eyeOCR *rollingCSV
	cmds   *rollingCSV
)

// Enabled reports whether debug logging was initialized (-debug).
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// Root returns the debug output directory (empty if not initialized).
func Root() string {
	mu.Lock()
	defer mu.Unlock()
	return root
}

// Init creates the output directory and enables rolling CSV writers.
func Init(path string) error {
	if path == "" {
		path = DefaultRoot
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	root = path
	enabled = true
	eyeOCR = newRollingCSV(filepath.Join(path, "eye_ocr.csv"), []string{
		"timestamp", "eye_x", "eye_y", "ocr_text", "ocr_cx", "ocr_cy", "distance_px", "region_w", "region_h",
	})
	cmds = newRollingCSV(filepath.Join(path, "commands.csv"), []string{
		"timestamp", "kind", "source", "category", "name", "detail",
	})
	return nil
}

// AppendEyeOCR appends rows for the eye/OCR rolling log.
func AppendEyeOCR(row []string) {
	mu.Lock()
	w := eyeOCR
	mu.Unlock()
	if w == nil {
		return
	}
	w.append(row)
}

// AppendCommand appends a row to the commands/hotkeys rolling log.
func AppendCommand(row []string) {
	mu.Lock()
	w := cmds
	mu.Unlock()
	if w == nil {
		return
	}
	w.append(row)
}

type csvRow struct {
	t   time.Time
	rec []string
}

type rollingCSV struct {
	mu       sync.Mutex
	path     string
	header   []string
	rows     []csvRow
	maxAge   time.Duration
}

func newRollingCSV(path string, header []string) *rollingCSV {
	return &rollingCSV{
		path:   path,
		header: header,
		maxAge: Window,
	}
}

func (r *rollingCSV) append(fields []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.rows = append(r.rows, csvRow{t: now, rec: fields})
	cutoff := now.Add(-r.maxAge)
	i := 0
	for i < len(r.rows) && r.rows[i].t.Before(cutoff) {
		i++
	}
	if i > 0 {
		r.rows = r.rows[i:]
	}
	r.flushLocked()
}

func (r *rollingCSV) flushLocked() {
	tmp := r.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	w := csv.NewWriter(f)
	_ = w.Write(r.header)
	for _, row := range r.rows {
		line := make([]string, 0, len(row.rec)+1)
		line = append(line, row.t.Format(time.RFC3339Nano))
		line = append(line, row.rec...)
		_ = w.Write(line)
	}
	w.Flush()
	_ = f.Close()
	if err := w.Error(); err != nil {
		_ = os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, r.path)
}

// PrunePOIFrameDirs keeps at most MaxPOIFrameDirs batch directories under poi_frames,
// deleting the oldest when a new one is added.
func PrunePOIFrameDirs() {
	mu.Lock()
	dir := root
	mu.Unlock()
	if dir == "" {
		return
	}
	framesDir := filepath.Join(dir, "poi_frames")
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		return
	}
	type batch struct {
		name string
		mod  time.Time
	}
	var batches []batch
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(framesDir, e.Name())
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		batches = append(batches, batch{name: e.Name(), mod: st.ModTime()})
	}
	if len(batches) <= MaxPOIFrameDirs {
		return
	}
	sort.Slice(batches, func(i, j int) bool {
		return batches[i].mod.Before(batches[j].mod)
	})
	for i := 0; i < len(batches)-MaxPOIFrameDirs; i++ {
		_ = os.RemoveAll(filepath.Join(framesDir, batches[i].name))
	}
}
