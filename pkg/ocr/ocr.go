package ocr

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	log "github.com/sirupsen/logrus"
)

// TextItem represents a piece of text found via OCR with its location
type TextItem struct {
	Text string
	Rect image.Rectangle
}

// FindMatches performs fuzzy matching on text items and returns matching items
// with bounding boxes narrowed to just the matched substring
func FindMatches(items []TextItem, query string) []TextItem {
	if query == "" {
		return items
	}

	query = strings.ToLower(strings.TrimSpace(query))
	var matches []TextItem

	// First pass: exact substring matches (case-insensitive)
	for _, item := range items {
		itemText := strings.ToLower(item.Text)
		if idx := strings.Index(itemText, query); idx >= 0 {
			// Narrow the bounding box to just the matched substring
			narrowedItem := narrowBoundingBox(item, idx, len(query))
			matches = append(matches, narrowedItem)
		}
	}

	// If we have exact matches, return them
	if len(matches) > 0 {
		log.Infof("Found %d exact matches for '%s'", len(matches), query)
		return matches
	}

	// Second pass: fuzzy matching for each word in the query
	queryWords := strings.Fields(query)
	for _, item := range items {
		itemText := strings.ToLower(item.Text)

		// Check if any query word fuzzy-matches the item text
		for _, word := range queryWords {
			if fuzzy.Match(word, itemText) {
				// For fuzzy matches, try to find the best matching position
				// by looking for the longest common substring
				idx, length := findBestFuzzyPosition(itemText, word)
				narrowedItem := narrowBoundingBox(item, idx, length)
				matches = append(matches, narrowedItem)
				break
			}
		}
	}

	log.Infof("Found %d fuzzy matches for '%s'", len(matches), query)
	return matches
}

// narrowBoundingBox calculates a proportionally narrowed bounding box
// based on where the match appears within the text
func narrowBoundingBox(item TextItem, startIdx, matchLen int) TextItem {
	totalLen := len(item.Text)
	if totalLen == 0 || matchLen >= totalLen {
		return item
	}

	rect := item.Rect
	width := rect.Dx()

	// Calculate proportional X coordinates
	startRatio := float64(startIdx) / float64(totalLen)
	endRatio := float64(startIdx+matchLen) / float64(totalLen)

	newMinX := rect.Min.X + int(float64(width)*startRatio)
	newMaxX := rect.Min.X + int(float64(width)*endRatio)

	// Ensure minimum width
	if newMaxX-newMinX < 2 {
		newMaxX = newMinX + 2
	}

	return TextItem{
		Text: item.Text[startIdx : startIdx+matchLen],
		Rect: image.Rect(newMinX, rect.Min.Y, newMaxX, rect.Max.Y),
	}
}

// findBestFuzzyPosition finds the best position and length for a fuzzy match
// Returns the start index and length of the matching portion
func findBestFuzzyPosition(text, query string) (int, int) {
	// Try to find the query as a substring first
	if idx := strings.Index(text, query); idx >= 0 {
		return idx, len(query)
	}

	// For fuzzy matches, assume the match starts at the beginning
	// and covers approximately the query length
	// This is a heuristic - fuzzy matches are less precise
	matchLen := len(query)
	if matchLen > len(text) {
		matchLen = len(text)
	}
	return 0, matchLen
}

// DrawOverlay creates an overlay image with rectangles and hex IDs for the matched text items
func DrawOverlay(img image.Image, matches []TextItem) ([]image.Rectangle, image.Image, error) {
	return DrawOverlayWithRegion(img, matches, nil)
}

// ScanRegionResult contains the results of a region scan
type ScanRegionResult struct {
	Matches       []TextItem        // Matches in full-screen logical coords
	LogicalRects  []image.Rectangle // For overlay display (logical coords)
	PhysicalRects []image.Rectangle // For mouse commands (physical coords)
}

// ScanRegion performs OCR on a region image and returns matches in full-screen logical coordinates
// regionImg: the captured region image (in physical pixels)
// regionRect: the region bounds in logical screen coordinates
// scaleFactor: the display scale factor (e.g., 2.0 for Retina)
// searchText: the text to search for
func ScanRegion(regionImg image.Image, regionRect image.Rectangle, scaleFactor float64, searchText string) (*ScanRegionResult, error) {
	// Perform OCR extraction on the region
	textItems, err := ExtractText(regionImg)
	if err != nil {
		return nil, fmt.Errorf("OCR extraction failed: %w", err)
	}

	// First, scale OCR rects from physical to logical coords (still relative to region)
	// Then add the logical region offset to get full-screen logical coords
	offsetX := regionRect.Min.X // logical
	offsetY := regionRect.Min.Y // logical
	for i := range textItems {
		// Scale from physical to logical first
		scaledRect := image.Rect(
			int(float64(textItems[i].Rect.Min.X)/scaleFactor),
			int(float64(textItems[i].Rect.Min.Y)/scaleFactor),
			int(float64(textItems[i].Rect.Max.X)/scaleFactor),
			int(float64(textItems[i].Rect.Max.Y)/scaleFactor),
		)
		// Then add logical offset
		textItems[i].Rect = scaledRect.Add(image.Pt(offsetX, offsetY))
	}

	// Clean up search text
	searchText = strings.TrimSpace(strings.ToLower(searchText))

	// Find matches (now in full-screen logical coords)
	matches := FindMatches(textItems, searchText)

	// Build result with both logical and physical rects
	result := &ScanRegionResult{
		Matches: matches,
	}

	for _, match := range matches {
		result.LogicalRects = append(result.LogicalRects, match.Rect)
		// Scale back to physical coords for mouse commands
		result.PhysicalRects = append(result.PhysicalRects, image.Rect(
			int(float64(match.Rect.Min.X)*scaleFactor),
			int(float64(match.Rect.Min.Y)*scaleFactor),
			int(float64(match.Rect.Max.X)*scaleFactor),
			int(float64(match.Rect.Max.Y)*scaleFactor),
		))
	}

	return result, nil
}

// ScanFullScreen performs OCR on a full screen image and returns matches
// img: the captured screen image (in physical pixels)
// scaleFactor: the display scale factor (e.g., 2.0 for Retina)
// searchText: the text to search for
func ScanFullScreen(img image.Image, scaleFactor float64, searchText string) (*ScanRegionResult, error) {
	// Perform OCR extraction
	textItems, err := ExtractText(img)
	if err != nil {
		return nil, fmt.Errorf("OCR extraction failed: %w", err)
	}

	// Clean up search text
	searchText = strings.TrimSpace(strings.ToLower(searchText))

	// Find matches (in physical coords)
	matches := FindMatches(textItems, searchText)

	// Build result with both logical and physical rects
	result := &ScanRegionResult{
		Matches: matches,
	}

	for _, match := range matches {
		// Physical coords (original)
		result.PhysicalRects = append(result.PhysicalRects, match.Rect)
		// Scale to logical coords for overlay
		result.LogicalRects = append(result.LogicalRects, image.Rect(
			int(float64(match.Rect.Min.X)/scaleFactor),
			int(float64(match.Rect.Min.Y)/scaleFactor),
			int(float64(match.Rect.Max.X)/scaleFactor),
			int(float64(match.Rect.Max.Y)/scaleFactor),
		))
	}

	// Update matches to use logical coords (for overlay drawing)
	for i := range result.Matches {
		result.Matches[i].Rect = result.LogicalRects[i]
	}

	return result, nil
}

// FindNearestMatch finds the match closest to the given point (in logical coords)
// Returns nil if no matches provided
func FindNearestMatch(matches []TextItem, x, y int) *TextItem {
	if len(matches) == 0 {
		return nil
	}

	var nearest *TextItem
	minDistance := math.MaxFloat64

	for i := range matches {
		// Calculate center of the match rectangle
		centerX := matches[i].Rect.Min.X + (matches[i].Rect.Dx() / 2)
		centerY := matches[i].Rect.Min.Y + (matches[i].Rect.Dy() / 2)

		// Calculate squared distance from point
		dx := float64(centerX - x)
		dy := float64(centerY - y)
		distance := dx*dx + dy*dy

		if distance < minDistance {
			minDistance = distance
			nearest = &matches[i]
		}
	}

	return nearest
}
