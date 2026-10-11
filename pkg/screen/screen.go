package screen

import (
	"fmt"
	"image"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/debuglog"
	"github.com/mousybusiness/mouse_whisperer/internal/features"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/pkg/cv"
	"github.com/mousybusiness/mouse_whisperer/pkg/ocr"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

type Screen struct {
	screenCapturing  bool
	interestingRects []image.Rectangle
	lastRects        []image.Rectangle
	image            image.Image
	mu               sync.Mutex

	// OCR caching - text items extracted from region near eye/mouse in logical coordinates
	ocrTextItems          []ocr.TextItem
	ocrRegionSize         int                     // Size of region to scan for OCR (from settings)
	ocrRegionSizeFallback int                     // Size of region for OCR command fallback (default 300)
	getEyePosition        func() (int, int, bool) // Callback to get current eye position
	ocrMu                 sync.RWMutex

	debugMode bool
	debugMu   sync.RWMutex
}

func New() *Screen {
	s := Screen{}

	// the capture loop exists to feed OpenCV element detection and the
	// local OCR cache; without them it would only burn a screenshot a second
	if features.OCR {
		go s.captureScreenContinuous()
	}
	return &s
}

// SetOCRRegionSize sets the region size for background OCR scanning
func (m *Screen) SetOCRRegionSize(size int) {
	m.ocrMu.Lock()
	defer m.ocrMu.Unlock()
	m.ocrRegionSize = size
}

// SetOCRRegionSize sets the region size for background OCR scanning
func (m *Screen) SetOCRRegionSizeFallback(size int) {
	m.ocrMu.Lock()
	defer m.ocrMu.Unlock()
	m.ocrRegionSizeFallback = size
}

// SetEyePositionGetter sets the callback to get current eye position
func (m *Screen) SetEyePositionGetter(getter func() (int, int, bool)) {
	m.ocrMu.Lock()
	defer m.ocrMu.Unlock()
	m.getEyePosition = getter
}

// SetDebug enables CSV logging for OCR and POI frame dumps when combined with debuglog.Init (-debug).
func (m *Screen) SetDebug(on bool) {
	m.debugMu.Lock()
	defer m.debugMu.Unlock()
	m.debugMode = on
}

func (m *Screen) debugEnabled() bool {
	m.debugMu.RLock()
	defer m.debugMu.RUnlock()
	return m.debugMode && debuglog.Enabled()
}

// FindNearest returns the center of the detected UI element nearest to vector
// (physical pixels), converted to logical coordinates. ok is false when no
// elements have been detected, e.g. in a noocr build.
func (m *Screen) FindNearest(vector util.Vector) (util.Vector, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mag := math.MaxFloat64

	var best util.Vector
	var tmp util.Vector
	var center image.Point
	var dist float64
	// log.Debugf("FindNearest called with vector: %+v, have %d rects", vector, len(m.interestingRects))
	for _, rect := range m.interestingRects {
		center = rect.Max.Sub(rect.Min).Div(2).Add(rect.Min)
		tmp.X = float64(center.X)
		tmp.Y = float64(center.Y)

		dist = vector.Distance(tmp)

		if dist < mag {
			mag = dist
			best = tmp
		}
	}

	if mag == math.MaxFloat64 {
		return util.Vector{}, false
	}

	result := best.Mul(1.0 / robotgo.ScaleF(robotgo.DisplayID))
	return result, true
}

func (m *Screen) Rects() []image.Rectangle {
	return m.interestingRects
}

func (m *Screen) GetImage() image.Image {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.image
}

func (m *Screen) SendToOverlay() []image.Rectangle {
	if m.image == nil {
		log.Warn("No image")
		return nil
	}

	if err := overlay.SendUIScreenshot(m.image); err != nil {
		log.Error(err)
		return nil
	}
	m.lastRects = slices.Clone(m.interestingRects)
	return m.lastRects
}

func (m *Screen) Cached() []image.Rectangle {
	return m.lastRects
}

// CaptureForOCR captures the current screen and returns the raw image for OCR processing
func (m *Screen) CaptureForOCR() (image.Image, error) {
	// Use the display that robotgo.DisplayID is set to (set in eye_tracking.go)
	_, _, w, h := robotgo.GetDisplayBounds(robotgo.DisplayID)
	bitmap := robotgo.CaptureScreen(0, 0, w, h)
	if bitmap == nil {
		return nil, fmt.Errorf("failed to capture screen")
	}
	defer robotgo.FreeBitmap(bitmap)

	// Convert the bitmap to an image.Image
	img := robotgo.ToImage(bitmap)
	log.Debugf("Captured screen for OCR: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	return img, nil
}

// CaptureRegionForOCR captures a specific region of the screen centered at the given position
// and returns the raw image for OCR processing along with the actual capture bounds.
// The returned rectangle represents the actual captured region in screen coordinates.
func (m *Screen) CaptureRegionForOCR(centerX, centerY, width, height int) (image.Image, image.Rectangle, error) {
	// Use the display that robotgo.DisplayID is set to (set in eye_tracking.go)
	_, _, displayW, displayH := robotgo.GetDisplayBounds(robotgo.DisplayID)

	// Calculate the capture region, centered on the given point
	x := centerX - width/2
	y := centerY - height/2

	// Clamp to screen bounds
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+width > displayW {
		x = displayW - width
		if x < 0 {
			x = 0
			width = displayW
		}
	}
	if y+height > displayH {
		y = displayH - height
		if y < 0 {
			y = 0
			height = displayH
		}
	}

	// Capture the region
	bitmap := robotgo.CaptureScreen(x, y, width, height)
	if bitmap == nil {
		return nil, image.Rectangle{}, fmt.Errorf("failed to capture screen region")
	}
	defer robotgo.FreeBitmap(bitmap)

	// Convert the bitmap to an image.Image
	img := robotgo.ToImage(bitmap)
	captureRect := image.Rect(x, y, x+width, y+height)

	// log.Debugf("Captured region for OCR: center=(%d,%d), size=%dx%d, actual bounds=%v",
	// 	centerX, centerY, img.Bounds().Dx(), img.Bounds().Dy(), captureRect)

	return img, captureRect, nil
}

func (m *Screen) captureScreen() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.screenCapturing {
		return
	}
	// Use the display that robotgo.DisplayID is set to (set in eye_tracking.go)
	_, _, w, h := robotgo.GetDisplayBounds(robotgo.DisplayID)
	bitmap := robotgo.CaptureScreen(0, 0, w, h)
	if bitmap == nil {
		fmt.Println("Failed to capture screen")
		return
	}
	defer robotgo.FreeBitmap(bitmap)

	// Convert the bitmap to an image.Image and encode it as a PNG
	img := robotgo.ToImage(bitmap)
	// log.Debugf("Captured screen: requested=%dx%d, actual image size=%dx%d", w, h, img.Bounds().Dx(), img.Bounds().Dy())

	var poiDir string
	if m.debugEnabled() {
		poiDir = filepath.Join(debuglog.Root(), "poi_frames", fmt.Sprintf("%d", time.Now().UnixNano()))
		defer debuglog.PrunePOIFrameDirs()
	}

	var err error
	m.interestingRects, m.image, err = cv.ExtractUIElementsRects(img, poiDir)
	if err != nil {
		return
	}
	// log.Debugf("Extracted %d interesting rects from screen capture", len(m.interestingRects))

	// Run OCR in background on a region around the mouse position
	go m.extractOCRTextNearMouse()
}

func (m *Screen) captureScreenContinuous() {
	m.screenCapturing = true
	for {
		if !m.screenCapturing {
			return
		}
		//start := time.Now()
		m.captureScreen()
		//fmt.Printf("Screenshot took: %v\n", time.Since(start))
		time.Sleep(time.Millisecond * 1000)
	}
}

// extractOCRTextNearMouse captures a region around the eye/mouse position and runs OCR on it
func (m *Screen) extractOCRTextNearMouse() {
	// Get OCR region size and eye position getter
	m.ocrMu.RLock()
	regionSize := m.ocrRegionSizeFallback
	getEyePos := m.getEyePosition
	m.ocrMu.RUnlock()

	if regionSize <= 0 {
		return // OCR not configured
	}

	// Try eye position first, fall back to mouse position
	var x, y int
	if getEyePos != nil {
		if ex, ey, ok := getEyePos(); ok {
			x, y = ex, ey
		} else {
			x, y = robotgo.Location()
		}
	} else {
		x, y = robotgo.Location()
	}

	// Capture a region around the position
	regionImg, regionRect, err := m.CaptureRegionForOCR(x, y, regionSize, regionSize)
	if err != nil {
		log.Debugf("OCR region capture failed: %v", err)
		return
	}

	scaleFactor := robotgo.ScaleF(robotgo.DisplayID)

	// Validate image before OCR to avoid leptonica errors
	if regionImg == nil || regionImg.Bounds().Dx() == 0 || regionImg.Bounds().Dy() == 0 {
		return
	}

	// Extract text items from the region (in physical coordinates relative to region)
	items, err := ocr.ExtractText(regionImg)
	if err != nil {
		return
	}

	// Convert to logical screen coordinates
	// OCR returns coordinates relative to the region image (in physical pixels)
	// We need to: 1) scale to logical, 2) offset by region position
	logicalItems := make([]ocr.TextItem, len(items))
	for i, item := range items {
		logicalItems[i] = ocr.TextItem{
			Text: item.Text,
			Rect: image.Rect(
				regionRect.Min.X+int(float64(item.Rect.Min.X)/scaleFactor),
				regionRect.Min.Y+int(float64(item.Rect.Min.Y)/scaleFactor),
				regionRect.Min.X+int(float64(item.Rect.Max.X)/scaleFactor),
				regionRect.Min.Y+int(float64(item.Rect.Max.Y)/scaleFactor),
			),
		}
	}

	m.ocrMu.Lock()
	m.ocrTextItems = logicalItems
	m.ocrMu.Unlock()

	if m.debugEnabled() {
		rw := regionRect.Dx()
		rh := regionRect.Dy()
		for _, item := range logicalItems {
			cx := item.Rect.Min.X + item.Rect.Dx()/2
			cy := item.Rect.Min.Y + item.Rect.Dy()/2
			dist := math.Hypot(float64(cx-x), float64(cy-y))
			debuglog.AppendEyeOCR([]string{
				strconv.Itoa(x),
				strconv.Itoa(y),
				item.Text,
				strconv.Itoa(cx),
				strconv.Itoa(cy),
				strconv.FormatFloat(dist, 'f', 2, 64),
				strconv.Itoa(rw),
				strconv.Itoa(rh),
			})
		}
	}
}

// FindTextNearby searches the cached OCR text items for a match near the given position.
// Returns the center coordinates of the nearest match, or (0, 0, false) if not found.
func (m *Screen) FindTextNearby(searchText string, x, y, regionSize int) (int, int, bool) {
	m.ocrMu.RLock()
	items := m.ocrTextItems
	m.ocrMu.RUnlock()

	if len(items) == 0 {
		log.Debug("OCR cache empty")
		return 0, 0, false
	}

	searchText = strings.ToLower(strings.TrimSpace(searchText))
	if searchText == "" {
		return 0, 0, false
	}

	// Define the search region
	halfSize := regionSize / 2
	regionRect := image.Rect(x-halfSize, y-halfSize, x+halfSize, y+halfSize)

	// Find matches within the region
	var matches []ocr.TextItem
	for _, item := range items {
		// Check if item is within the region
		if !item.Rect.Overlaps(regionRect) {
			continue
		}

		// Check for text match (case-insensitive substring or exact)
		itemText := strings.ToLower(item.Text)
		if strings.Contains(itemText, searchText) || strings.Contains(searchText, itemText) {
			matches = append(matches, item)
		}
	}

	if len(matches) == 0 {
		log.Debugf("No OCR matches for '%s' in region around (%d,%d)", searchText, x, y)
		return 0, 0, false
	}

	// Find the nearest match to the given position
	nearest := ocr.FindNearestMatch(matches, x, y)
	if nearest == nil {
		return 0, 0, false
	}

	// Return center of the match
	centerX := nearest.Rect.Min.X + (nearest.Rect.Dx() / 2)
	centerY := nearest.Rect.Min.Y + (nearest.Rect.Dy() / 2)

	log.Infof("OCR cache hit: '%s' -> '%s' at (%d,%d)", searchText, nearest.Text, centerX, centerY)
	return centerX, centerY, true
}
