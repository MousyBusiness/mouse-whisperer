package impl

import (
	"context"
	"fmt"
	"image"
	"math"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/features"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/ocr"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

// pickTarget decides how to act on ranked scan targets: a single match is
// unambiguous; with gaze available the nearest target within radius wins;
// anything else needs the numbered disambiguation overlay.
func pickTarget(targets []api.Target, gazeOK bool, gx, gy, radius int) (idx int, disambiguate bool) {
	switch {
	case len(targets) == 0:
		return -1, false
	case len(targets) == 1:
		return 0, false
	case !gazeOK:
		return -1, true
	}
	best, bestDist := -1, math.MaxFloat64
	for i, t := range targets {
		cx := t.Rect.X + t.Rect.W/2
		cy := t.Rect.Y + t.Rect.H/2
		if d := math.Hypot(float64(cx-gx), float64(cy-gy)); d < bestDist {
			best, bestDist = i, d
		}
	}
	if bestDist > float64(radius) {
		return -1, true
	}
	return best, false
}

// performTargetAction acts on a target rect (logical coords) at its center.
func performTargetAction(rect image.Rectangle, action string) {
	cx := rect.Min.X + rect.Dx()/2
	cy := rect.Min.Y + rect.Dy()/2
	switch action {
	case "move":
		robotgo.Move(cx, cy)
	case "select":
		robotgo.MoveClick(cx, cy, "left", true) // double-click selects the word
	default:
		robotgo.MoveClick(cx, cy)
	}
}

func actionPastTense(action string) string {
	switch action {
	case "move":
		return "Moved to"
	case "select":
		return "Selected"
	default:
		return "Clicked"
	}
}

// showTargetsOverlay renders the numbered boxes for resp's targets and
// records pendingAction so a bare spoken number completes it. Returns false
// on any failure so callers can fall through to the local OCR path.
func showTargetsOverlay(s *state.State, resp *api.TargetsResponse, pendingAction string) bool {
	if resp.Screen.Width <= 0 || resp.Screen.Height <= 0 {
		log.Warnf("scan server returned invalid screen size %dx%d, falling back to local OCR", resp.Screen.Width, resp.Screen.Height)
		return false
	}
	items := make([]ocr.TextItem, len(resp.Targets))
	for i, t := range resp.Targets {
		items[i] = ocr.TextItem{Text: t.Label, Rect: t.Rect.ToImageRect()}
	}
	// DrawOverlay paints boxes + number labels on a transparent layer and
	// only reads the canvas for its size, so no screenshot transfer is needed
	canvas := image.NewRGBA(image.Rect(0, 0, resp.Screen.Width, resp.Screen.Height))
	rects, overlayImg, err := ocr.DrawOverlay(canvas, items)
	if err != nil {
		log.Errorf("scan server overlay draw failed, falling back to local OCR: %v", err)
		return false
	}

	// logical coords; spoken numbers and "click/move/select <n>" index these
	s.CV.OCRLastRects = rects
	s.CV.OCRPendingAction = pendingAction
	if err := overlay.SendUIScreenshot(overlayImg); err != nil {
		log.Errorf("failed to send overlay, falling back to local OCR: %v", err)
		return false
	}
	return true
}

// findTargets queries screend providers in the configured preference order
// (accessibility elements before OCR when ax_enabled). A provider yields to
// the next one when it has nothing or only fuzzy-tier matches — an AX tree
// that doesn't know the word must not shadow OCR, which sees everything
// rendered — but a weak answer still beats a later provider's silence.
// Errors only when no provider answered at all.
func findTargets(s *state.State, searchText string) (*api.TargetsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	providers := s.ScanProviders
	if len(providers) == 0 {
		providers = []string{api.ProviderOCR}
	}

	var weak, empty *api.TargetsResponse
	var lastErr error
	for _, provider := range providers {
		// bounded staleness: OCR accepts the prewarm capture from utterance
		// start; ax snapshots are cheap tree walks, so they get a tighter
		// bound instead of reusing a pre-utterance UI state
		maxAge := 4000
		if provider == api.ProviderAX {
			maxAge = 1000
		}
		resp, err := s.ScanClient.FindTargets(ctx, api.TargetsRequest{
			Provider: provider,
			Query:    searchText,
			MaxAgeMs: &maxAge,
		})
		if err != nil {
			lastErr = err
			log.Warnf("scan provider %s failed: %v", provider, err)
			continue
		}
		switch {
		case len(resp.Targets) == 0:
			empty = resp
		case resp.Targets[0].Match == "fuzzy":
			// targets are tier-sorted, so a fuzzy head means fuzzy-only
			if weak == nil {
				weak = resp
			}
		default:
			return resp, nil
		}
	}
	if weak != nil {
		return weak, nil
	}
	if empty != nil {
		return empty, nil
	}
	return nil, lastErr
}

// actViaServer resolves "click/move/select <text>" through screend: a single
// match (or the nearest to gaze) acts immediately; anything ambiguous shows
// the numbered overlay and stores the action so saying the number finishes
// it. Returns false only when the caller should fall back to local OCR.
func actViaServer(s *state.State, searchText, action string) bool {
	resp, err := findTargets(s, searchText)
	if err != nil {
		log.Warnf("scan server unavailable, falling back to local OCR: %v", err)
		return false
	}
	if len(resp.Targets) == 0 {
		overlay.SendUIMessage(fmt.Sprintf("No matches found for '%s'", searchText))
		return true
	}

	gx, gy, gazeOK := s.GetPositionAtTime(s.GetRecordingStarted())
	idx, disambiguate := pickTarget(resp.Targets, gazeOK, gx, gy, s.GetOCRRegionSize())
	if disambiguate {
		if !showTargetsOverlay(s, resp, action) {
			return false
		}
		overlay.SendUIMessage(fmt.Sprintf("%d matches for '%s' - say a number to %s", len(resp.Targets), searchText, action))
		return true
	}

	t := resp.Targets[idx]
	log.Infof("scan server: %s '%s' (%s, %s match, gaze=%v cache=%v age=%dms)", action, t.Label, resp.Provider, t.Match, gazeOK, resp.FromCache, resp.AgeMs)
	performTargetAction(t.Rect.ToImageRect(), action)
	overlay.SendUIMessage(fmt.Sprintf("%s '%s'", actionPastTense(action), t.Label))
	return true
}

// scanViaServer resolves a scan through the standalone screend server: the
// prewarmed index answers in milliseconds and full-screen coverage is cheap,
// so both scan commands use the whole screen here. Returns true when fully
// handled (including a clean no-match); any failure returns false so the
// caller falls through to the local OCR path unchanged.
func scanViaServer(s *state.State, searchText string) bool {
	resp, err := findTargets(s, searchText)
	if err != nil {
		log.Warnf("scan server unavailable, falling back to local OCR: %v", err)
		return false
	}
	log.Infof("scan server: %d targets for %q (%s, cache=%v age=%dms)", len(resp.Targets), searchText, resp.Provider, resp.FromCache, resp.AgeMs)

	if len(resp.Targets) == 0 {
		overlay.SendUIMessage(fmt.Sprintf("No matches found for '%s'", searchText))
		return true
	}
	// a scan's most common follow-up is a click, so that's what a bare
	// spoken number does; "move/select <n>" still override
	if !showTargetsOverlay(s, resp, "click") {
		return false
	}
	overlay.SendUIMessage(fmt.Sprintf("Found %d matches - say a number to click, or 'move/select <n>'", len(resp.Targets)))
	return true
}

// noLocalOCR reports, and tells the user, when the scan server couldn't
// answer and this binary has no local OCR to fall back to (noocr build)
func noLocalOCR() bool {
	if features.OCR {
		return false
	}
	overlay.SendUIMessage("Scan server unavailable and local OCR is not built in")
	return true
}

// RegisterElementHandlers registers the commands built on the UI elements
// the local OpenCV pipeline detects (ocr feature only).
func RegisterElementHandlers(registry Registry, state *state.State) {
	// Interactive mouse mode (placeholder): numbers the detected UI elements
	registry.Register("mouse_interactive", func(ctx context.Context, args map[string]any) (context.Context, error) {
		rects := state.Screen.Rects()
		if len(rects) == 0 {
			overlay.SendUIMessage("No interesting points")
			return ctx, nil
		}

		rects = state.Screen.SendToOverlay()

		// Scale rects from physical to logical coordinates (robotgo expects logical/point coordinates)
		scaleFactor := robotgo.ScaleF(robotgo.DisplayID)
		logicalRects := make([]image.Rectangle, len(rects))
		for i, rect := range rects {
			logicalRects[i] = image.Rect(
				int(float64(rect.Min.X)/scaleFactor),
				int(float64(rect.Min.Y)/scaleFactor),
				int(float64(rect.Max.X)/scaleFactor),
				int(float64(rect.Max.Y)/scaleFactor),
			)
		}
		state.CV.OCRLastRects = logicalRects

		return ctx, nil
	})
}

func RegisterOCRHandlers(registry Registry, state *state.State) {

	// OCR handlers
	registry.Register("ocr_scan_full", func(ctx context.Context, args map[string]any) (context.Context, error) {

		overlay.SendUIMessage("Scanning...")

		_, searchText, err := captureInput(ctx, state, &commands.InputAction{
			Prompt:  "Text to search for (e.g. 'readme.md'): ",
			Break:   750 * time.Millisecond,
			Timeout: 3 * time.Second,
		})

		if strings.TrimSpace(searchText) == "" {
			overlay.SendUIMessage("No search text provided")
			return ctx, nil
		}

		log.Infof("OCR scan for: %s", searchText)

		if state.ScanClient != nil && scanViaServer(state, searchText) {
			return ctx, nil
		}
		if noLocalOCR() {
			return ctx, nil
		}

		// Capture the screen
		img, err := state.CaptureScreen()
		if err != nil {
			log.Errorf("Failed to capture screen: %v", err)
			overlay.SendUIMessage("Failed to capture screen")
			return ctx, nil
		}

		scaleFactor := robotgo.ScaleF(robotgo.DisplayID)

		// Use OCR package helper for scanning
		result, err := ocr.ScanFullScreen(img, scaleFactor, searchText)
		if err != nil {
			log.Errorf("OCR scan failed: %v", err)
			overlay.SendUIMessage("OCR failed")
			return ctx, nil
		}

		if len(result.Matches) == 0 {
			overlay.SendUIMessage(fmt.Sprintf("No matches found for '%s'", searchText))
			return ctx, nil
		}

		log.Infof("Found %d matches for '%s'", len(result.Matches), searchText)

		// Scale image to logical coords for overlay
		logicalImg, err := ocr.ScaleImageToLogical(img, scaleFactor)
		if err != nil {
			log.Errorf("Failed to scale image: %v", err)
			overlay.SendUIMessage("Failed to create overlay")
			return ctx, nil
		}

		// Draw overlay
		_, overlayImg, err := ocr.DrawOverlay(logicalImg, result.Matches)
		if err != nil {
			log.Errorf("Failed to draw overlay: %v", err)
			overlay.SendUIMessage("Failed to create overlay")
			return ctx, nil
		}

		// Store logical rects for mouse commands (robotgo expects logical/point coordinates)
		state.CV.OCRLastRects = result.LogicalRects

		if err := overlay.SendUIScreenshot(overlayImg); err != nil {
			log.Errorf("Failed to send overlay: %v", err)
			overlay.SendUIMessage("Failed to display overlay")
			return ctx, nil
		}

		state.CV.OCRPendingAction = "click"
		overlay.SendUIMessage(fmt.Sprintf("Found %d matches - say a number to click, or 'move/select <n>'", len(result.Matches)))
		return ctx, nil
	})

	registry.Register("ocr_scan_nearby", func(ctx context.Context, args map[string]any) (context.Context, error) {
		_, searchText, err := captureInput(ctx, state, &commands.InputAction{
			Prompt:  "Text to search for (e.g. 'readme.md'): ",
			Break:   750 * time.Millisecond,
			Timeout: 3 * time.Second,
		})

		// Get the eye position from the start time of the recording (from cache)
		x, y, ok := state.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			overlay.SendUIMessage("No eye position found at recording start")
			x, y = robotgo.Location()
			log.Infof("No eye position found at recording start, using mouse position: (%d,%d)", x, y)
		}

		overlay.SendUIMessage(fmt.Sprintf("Scanning nearby (%d,%d)...", x, y))

		if strings.TrimSpace(searchText) == "" {
			overlay.SendUIMessage("No search text provided")
			return ctx, nil
		}

		log.Infof("OCR scan nearby for: %s at position (%d, %d)", searchText, x, y)

		// server path scans the whole screen: full coverage is cheap there,
		// and "scan X" should find every visible instance
		if state.ScanClient != nil && scanViaServer(state, searchText) {
			return ctx, nil
		}
		if noLocalOCR() {
			return ctx, nil
		}

		// Capture a region around the eye position
		regionSize := state.GetOCRRegionSize()
		regionImg, regionRect, err := state.CaptureRegion(x, y, regionSize, regionSize)
		if err != nil {
			log.Errorf("Failed to capture region: %v", err)
			overlay.SendUIMessage("Failed to capture region")
			return ctx, nil
		}

		scaleFactor := robotgo.ScaleF(robotgo.DisplayID)

		// Use OCR package helper for scanning region
		result, err := ocr.ScanRegion(regionImg, regionRect, scaleFactor, searchText)
		if err != nil {
			log.Errorf("OCR scan failed: %v", err)
			overlay.SendUIMessage("OCR failed")
			return ctx, nil
		}

		if len(result.Matches) == 0 {
			overlay.SendUIMessage(fmt.Sprintf("No matches found for '%s' nearby", searchText))
			return ctx, nil
		}

		log.Infof("Found %d matches for '%s' in nearby region", len(result.Matches), searchText)

		// For the overlay, we need to capture the full screen to draw on
		fullScreenImg, err := state.CaptureScreen()
		if err != nil {
			log.Errorf("Failed to capture full screen for overlay: %v", err)
			overlay.SendUIMessage("Failed to create overlay")
			return ctx, nil
		}

		// Scale image to logical coords for overlay
		logicalImg, err := ocr.ScaleImageToLogical(fullScreenImg, scaleFactor)
		if err != nil {
			log.Errorf("Failed to scale image: %v", err)
			overlay.SendUIMessage("Failed to create overlay")
			return ctx, nil
		}

		// Draw overlay with region indicator
		_, overlayImg, err := ocr.DrawOverlayWithRegion(logicalImg, result.Matches, &regionRect)
		if err != nil {
			log.Errorf("Failed to draw overlay: %v", err)
			overlay.SendUIMessage("Failed to create overlay")
			return ctx, nil
		}

		// Store logical rects for mouse commands (robotgo expects logical/point coordinates)
		state.CV.OCRLastRects = result.LogicalRects

		if err := overlay.SendUIScreenshot(overlayImg); err != nil {
			log.Errorf("Failed to send overlay: %v", err)
			overlay.SendUIMessage("Failed to display overlay")
			return ctx, nil
		}

		state.CV.OCRPendingAction = "click"
		overlay.SendUIMessage(fmt.Sprintf("Found %d matches nearby - say a number to click, or 'move/select <n>'", len(result.Matches)))
		return ctx, nil
	})

	registerNearbyAction(registry, state, "ocr_click_nearby", "Text to click: ", "click")
	registerNearbyAction(registry, state, "ocr_move_nearby", "Text to move to: ", "move")
	registerNearbyAction(registry, state, "ocr_select_nearby", "Text to select: ", "select")

	// ocr_pick_target completes the action stored when the disambiguation
	// overlay was shown; the matched trigger IS the spoken target number.
	registry.Register("ocr_pick_target", func(ctx context.Context, args map[string]any) (context.Context, error) {
		words := xctx.Get(ctx).TriggerWords
		if len(words) == 0 {
			return ctx, fmt.Errorf("no target number spoken")
		}
		if !pickTargetByNumber(state, strings.Join(words, " ")) {
			return ctx, fmt.Errorf("invalid target number %q", strings.Join(words, " "))
		}
		return ctx, nil
	})
}

// pickTargetByNumber acts on overlay target n (spoken as digits or words)
// with the pending action; false when text isn't a number on screen.
func pickTargetByNumber(s *state.State, text string) bool {
	index, ok := util.ParseSpokenNumber(text)
	if !ok {
		return false
	}
	rects := s.GetOCRRects()
	if index < 0 || index >= len(rects) {
		overlay.SendUIMessage(fmt.Sprintf("No target %d on screen", index))
		return false
	}
	action := s.CV.OCRPendingAction
	if action == "" {
		action = "click"
	}
	rect := rects[index]
	s.ClearOverlay()
	performTargetAction(rect, action)
	overlay.SendUIMessage(fmt.Sprintf("%s target %d", actionPastTense(action), index))
	return true
}

// TryPickTarget completes the pending overlay action when an otherwise
// unmatched utterance is just a target number — bare digit triggers only
// cover 0-9, so "fifty" with 50+ targets on screen lands here via the
// fallback chain. Reports whether it acted.
func TryPickTarget(s *state.State, text string) bool {
	if !s.HasState(state.StateOverlayHex) || len(s.GetOCRRects()) == 0 {
		return false
	}
	if _, ok := util.ParseSpokenNumber(text); !ok {
		return false
	}
	return pickTargetByNumber(s, text)
}

// registerNearbyAction registers one of the "act on text" handlers
// (click/move/select). The scan server resolves it full-screen with gaze
// disambiguation; without the server it falls back to the legacy local OCR
// of the region around the gaze, acting on the nearest match.
func registerNearbyAction(registry Registry, state *state.State, method commands.CommandName, prompt, action string) {
	registry.Register(method, func(ctx context.Context, args map[string]any) (context.Context, error) {
		_, searchText, _ := captureInput(ctx, state, &commands.InputAction{
			Prompt:  prompt,
			Break:   750 * time.Millisecond,
			Timeout: 3 * time.Second,
		})
		if strings.TrimSpace(searchText) == "" {
			overlay.SendUIMessage("No search text provided")
			return ctx, nil
		}

		if state.ScanClient != nil && actViaServer(state, searchText, action) {
			return ctx, nil
		}
		if noLocalOCR() {
			return ctx, nil
		}

		// local fallback: OCR the region around the gaze, act on the nearest match
		x, y, ok := state.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			overlay.SendUIMessage("No eye position found at recording start")
			x, y = robotgo.Location()
			log.Infof("No eye position found at recording start, using mouse position: (%d,%d)", x, y)
		}
		log.Infof("OCR %s nearby for: %s at position (%d, %d)", action, searchText, x, y)

		regionSize := state.GetOCRRegionSize()
		regionImg, regionRect, err := state.CaptureRegion(x, y, regionSize, regionSize)
		if err != nil {
			log.Errorf("Failed to capture region: %v", err)
			overlay.SendUIMessage("Failed to capture region")
			return ctx, nil
		}

		result, err := ocr.ScanRegion(regionImg, regionRect, robotgo.ScaleF(robotgo.DisplayID), searchText)
		if err != nil {
			log.Errorf("OCR scan failed: %v", err)
			overlay.SendUIMessage("OCR failed")
			return ctx, nil
		}
		if len(result.Matches) == 0 {
			overlay.SendUIMessage(fmt.Sprintf("No matches found for '%s' nearby", searchText))
			return ctx, nil
		}

		// matches are already in full-screen logical coords
		nearestMatch := ocr.FindNearestMatch(result.Matches, x, y)
		if nearestMatch == nil {
			overlay.SendUIMessage(fmt.Sprintf("No match to %s", action))
			return ctx, nil
		}

		log.Infof("%s nearest match '%s' at logical %v", action, nearestMatch.Text, nearestMatch.Rect)
		performTargetAction(nearestMatch.Rect, action)
		overlay.SendUIMessage(fmt.Sprintf("%s '%s'", actionPastTense(action), nearestMatch.Text))
		return ctx, nil
	})
}

// TryOCRMoveFallback attempts to find text on screen and move to it.
// Returns true if text was found and mouse was moved, false otherwise.
// Only triggers if the search text has maxWords or fewer words.
// Uses cached OCR data from background screen capture for fast lookup.
func TryOCRMoveFallback(s *state.State, searchText string, regionSize int, maxWords int) bool {
	searchText = strings.TrimSpace(searchText)
	if searchText == "" {
		return false
	}

	// Check word count
	words := strings.Fields(searchText)
	if len(words) > maxWords {
		log.Debugf("OCR fallback skipped: %d words exceeds max %d", len(words), maxWords)
		return false
	}

	// Get eye position or mouse position
	x, y, ok := s.GetPositionAtTime(s.GetRecordingStarted())
	if !ok {
		x, y = robotgo.Location()
		log.Debugf("OCR fallback using mouse position: (%d,%d)", x, y)
	}

	log.Infof("OCR fallback for: '%s' at position (%d, %d) with region %dx%d", searchText, x, y, regionSize, regionSize)

	// Try cached OCR data first (fast path)
	if s.Screen != nil {
		if moveX, moveY, found := s.Screen.FindTextNearby(searchText, x, y, regionSize); found {
			log.Infof("OCR fallback (cached): moving to (%d, %d)", moveX, moveY)
			robotgo.Move(moveX, moveY)
			overlay.SendUIMessage(fmt.Sprintf("Moved to '%s'", searchText))
			return true
		}
	}

	log.Debugf("OCR fallback: no cached match for '%s'", searchText)
	return false
}
