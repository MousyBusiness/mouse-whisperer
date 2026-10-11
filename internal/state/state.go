package state

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/screenclient"
	"github.com/mousybusiness/mouse_whisperer/pkg/eye_tracking"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/screen"
	"github.com/mousybusiness/mouse_whisperer/types"
	log "github.com/sirupsen/logrus"
)

// Application states for command filtering
const (
	StateNotListening = types.ExecutionState("not_listening")
	StateOverlayHex   = types.ExecutionState("overlay_hex")
	StateGhost        = types.ExecutionState("ghost")
	StateMouseDown    = types.ExecutionState("mouse_down")
	StateDictating    = types.ExecutionState("dictating")
)

// CodingState holds state for coding-related handlers
type CodingState struct {
	CodeLanguageContext string
	PendingSuggestion   *string
}

// DictationState accumulates spoken prose, one segment per utterance,
// until it is committed ("insert"/"take") or discarded ("reject").
// Every mutation snapshots the previous state so mistakes — an LLM edit
// gone wrong, an over-eager deletion — can be undone within the session.
type DictationState struct {
	mu        sync.Mutex
	segments  []segment
	undoStack [][]segment
	redoStack [][]segment
}

// segment is one buffer entry plus the recordings it was transcribed
// from, so "replay that" can re-read the audio as a whole. clips is never
// mutated in place (always replaced), so shallow copies are safe history.
type segment struct {
	text  string
	clips []string
}

// ReplayRun is a stretch of consecutive buffer segments: either recorded
// prose (Clips set) to re-transcribe as one, or text with no usable audio
// (commands fired mid-utterance, typed-in edits) to keep verbatim.
type ReplayRun struct {
	Text  string
	Clips []string
}

const dictationHistoryCap = 50

// snapshot must be called with the lock held, before a mutation that will
// actually change the buffer. A new mutation invalidates the redo stack.
func (d *DictationState) snapshot() {
	d.undoStack = append(d.undoStack, append([]segment(nil), d.segments...))
	if len(d.undoStack) > dictationHistoryCap {
		d.undoStack = d.undoStack[1:]
	}
	d.redoStack = nil
}

// Undo reverts the last buffer mutation ("undo" while dictating)
func (d *DictationState) Undo() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.undoStack) == 0 {
		return false
	}
	d.redoStack = append(d.redoStack, append([]segment(nil), d.segments...))
	last := len(d.undoStack) - 1
	d.segments = append([]segment(nil), d.undoStack[last]...)
	d.undoStack = d.undoStack[:last]
	return true
}

// Redo re-applies an undone mutation ("redo" while dictating)
func (d *DictationState) Redo() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.redoStack) == 0 {
		return false
	}
	d.undoStack = append(d.undoStack, append([]segment(nil), d.segments...))
	last := len(d.redoStack) - 1
	d.segments = append([]segment(nil), d.redoStack[last]...)
	d.redoStack = d.redoStack[:last]
	return true
}

// Append adds one utterance to the buffer. Raw whisper text is kept as-is
// (punctuation and case matter for prose) apart from trimming.
func (d *DictationState) Append(s string) {
	d.AppendClip(s, "")
}

// AppendClip is Append for text transcribed from exactly the given
// recording, which "replay that" can later re-read. Empty clip means the
// text has no replayable audio.
func (d *DictationState) AppendClip(s, clip string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot()
	seg := segment{text: s}
	if clip != "" {
		seg.clips = []string{clip}
	}
	d.segments = append(d.segments, seg)
}

// PopLast removes and returns the most recent utterance ("scratch that")
func (d *DictationState) PopLast() (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.segments) == 0 {
		return "", false
	}
	d.snapshot()
	last := d.segments[len(d.segments)-1]
	d.segments = d.segments[:len(d.segments)-1]
	return last.text, true
}

// PopWords removes the last n words from the buffer ("delete word"),
// crossing utterance boundaries backwards as segments empty out
func (d *DictationState) PopWords(n int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.segments) == 0 || n <= 0 {
		return false
	}
	d.snapshot()
	for n > 0 && len(d.segments) > 0 {
		last := len(d.segments) - 1
		words := strings.Fields(d.segments[last].text)
		if len(words) <= n {
			n -= len(words)
			d.segments = d.segments[:last]
			continue
		}
		d.segments[last].text = strings.Join(words[:len(words)-n], " ")
		n = 0
	}
	return true
}

// PopChars removes the last n characters of the buffer as displayed
// ("delete that"): rune-wise, counting the join space between utterance
// segments as one character
func (d *DictationState) PopChars(n int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.segments) == 0 || n <= 0 {
		return false
	}
	d.snapshot()
	for n > 0 && len(d.segments) > 0 {
		last := len(d.segments) - 1
		runes := []rune(d.segments[last].text)
		if len(runes) <= n {
			n -= len(runes)
			d.segments = d.segments[:last]
			// the space Text() renders between segments is a character too
			if n > 0 && len(d.segments) > 0 {
				n--
			}
			continue
		}
		d.segments[last].text = string(runes[:len(runes)-n])
		n = 0
	}
	return true
}

// SetLeadingCase upper/lowercases the first letter of the buffer, fixing
// whisper's inconsistent leading capitalization before insert
func (d *DictationState) SetLeadingCase(upper bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.segments) == 0 {
		return false
	}
	s := d.segments[0].text
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return false
	}
	if upper {
		r = unicode.ToUpper(r)
	} else {
		r = unicode.ToLower(r)
	}
	d.snapshot()
	d.segments[0].text = string(r) + s[size:]
	return true
}

// SetTrailingPunct replaces the buffer's trailing sentence punctuation
// with the given mark, so a dictated statement can be re-terminated
// ("question that" → "?") without retyping.
func (d *DictationState) SetTrailingPunct(mark string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.segments) == 0 {
		return false
	}
	last := len(d.segments) - 1
	s := strings.TrimRightFunc(d.segments[last].text, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(".,;:!?…", r)
	})
	if s == "" {
		return false
	}
	d.snapshot()
	d.segments[last].text = s + mark
	return true
}

// ReplaceAll swaps the entire buffer for the given text as one segment
// ("tidy that" / "fix that"). Empty text clears the buffer. The merged
// segment keeps every recording, so "replay that" can still undo an
// LLM rewrite back to what was actually said.
func (d *DictationState) ReplaceAll(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	s = strings.TrimSpace(s)
	d.snapshot()
	if s == "" {
		d.segments = nil
		return
	}
	var clips []string
	for _, seg := range d.segments {
		clips = append(clips, seg.clips...)
	}
	d.segments = []segment{{text: s, clips: clips}}
}

// ReplayRuns groups the buffer into consecutive runs for "replay that":
// segments with recordings merge into one run (re-transcribed together,
// so whisper hears whole sentences rather than pause-split fragments);
// segments without keep their text.
func (d *DictationState) ReplayRuns() []ReplayRun {
	d.mu.Lock()
	defer d.mu.Unlock()
	var runs []ReplayRun
	for _, seg := range d.segments {
		recorded := len(seg.clips) > 0
		if n := len(runs); n > 0 && (len(runs[n-1].Clips) > 0) == recorded {
			runs[n-1].Text += " " + seg.text
			runs[n-1].Clips = append(runs[n-1].Clips, seg.clips...)
			continue
		}
		runs = append(runs, ReplayRun{Text: seg.text, Clips: append([]string(nil), seg.clips...)})
	}
	return runs
}

// ReplaceRuns swaps the buffer for the replayed runs, one segment each
// (as an undoable mutation). Runs whose text came back empty are dropped.
func (d *DictationState) ReplaceRuns(runs []ReplayRun) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot()
	d.segments = nil
	for _, r := range runs {
		if t := strings.TrimSpace(r.Text); t != "" {
			d.segments = append(d.segments, segment{text: t, clips: append([]string(nil), r.Clips...)})
		}
	}
}

func (d *DictationState) Text() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.text()
}

func (d *DictationState) text() string {
	texts := make([]string, len(d.segments))
	for i, seg := range d.segments {
		texts[i] = seg.text
	}
	return strings.Join(texts, " ")
}

// TakeAll returns the joined buffer and clears it atomically. Committing
// ends the session: history is not kept across sessions.
func (d *DictationState) TakeAll() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.text()
	d.segments = nil
	d.undoStack = nil
	d.redoStack = nil
	return t
}

func (d *DictationState) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.segments = nil
	d.undoStack = nil
	d.redoStack = nil
}

func (d *DictationState) HasText() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.segments) > 0
}

// MouseState holds state for mouse-related handlers
type MouseState struct {
	Nudge                     int
	Scrolling                 bool
	Nudging                   bool
	Snapping                  bool
	Jumping                   bool
	Ghosting                  bool
	TunnelVision              bool
	MouseDown                 bool               // Track if left mouse button is held down
	NaturalScrolling          bool               // Invert scroll direction for natural scrolling
	CompanionMouseSensitivity float64            // Sensitivity multiplier for companion app mouse input
	CommandCh                 chan<- interface{} // Will be set during registration
}

func (m *MouseState) GetNudge() int {
	return m.Nudge
}
func (m *MouseState) SetNudge(n int) {
	m.Nudge = n
}

func (m *MouseState) GetNaturalScrolling() bool {
	return m.NaturalScrolling
}
func (m *MouseState) SetNaturalScrolling(b bool) {
	m.NaturalScrolling = b
}

func (m *MouseState) GetScrolling() bool {
	return m.Scrolling
}
func (m *MouseState) SetScrolling(b bool) {
	m.Scrolling = b
}

func (m *MouseState) GetGhosting() bool {
	return m.Ghosting
}
func (m *MouseState) SetGhosting(b bool) {
	m.Ghosting = b
}
func (m *MouseState) GetNudging() bool {
	return m.Nudging
}
func (m *MouseState) SetNudging(b bool) {
	m.Nudging = b
}
func (m *MouseState) GetSnapping() bool {
	return m.Snapping
}
func (m *MouseState) SetSnapping(b bool) {
	m.Snapping = b
}
func (m *MouseState) GetJumping() bool {
	return m.Jumping
}
func (m *MouseState) SetJumping(b bool) {
	m.Jumping = b
}

func (m *MouseState) GetTunnelVision() bool {
	return m.TunnelVision
}
func (m *MouseState) SetTunnelVision(b bool) {
	m.TunnelVision = b
}

func (m *MouseState) GetMouseDown() bool {
	return m.MouseDown
}
func (m *MouseState) SetMouseDown(b bool) {
	m.MouseDown = b
}

func (m *MouseState) GetCompanionMouseSensitivity() float64 {
	return m.CompanionMouseSensitivity
}

func (m *MouseState) SetCompanionMouseSensitivity(v float64) {
	m.CompanionMouseSensitivity = v
}

/*

type EyeTrackingState interface {
	GetGhosting() bool
	GetNudging() bool
	GetSnapping() bool
	GetJumping() bool
}*/

// DesktopState holds state for desktop-related handlers
type DesktopState struct {
	mu sync.Mutex
	// workspaceAliases maps a normalized spoken name to a Mission Control
	// space ID (in-memory only; space IDs don't survive logout anyway)
	workspaceAliases map[string]uint64
}

func (d *DesktopState) SetWorkspaceAlias(name string, spaceID uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.workspaceAliases == nil {
		d.workspaceAliases = make(map[string]uint64)
	}
	d.workspaceAliases[name] = spaceID
}

func (d *DesktopState) WorkspaceAlias(name string) (uint64, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id, ok := d.workspaceAliases[name]
	return id, ok
}

func (d *DesktopState) WorkspaceAliasNames() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	names := make([]string, 0, len(d.workspaceAliases))
	for n := range d.workspaceAliases {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// CVState holds state for CV-related handlers
type CVState struct {
	OCRLastRects          []image.Rectangle // Rectangles from last OCR scan
	OCRPendingAction      string            // action a spoken target number completes (click|move|select)
	OverlayTimer          *time.Timer       // Timer for auto-dismissing overlay
	OverlayTimeout        time.Duration     // Timeout duration for overlay (default 30s)
	OCRRegionSize         int               // Size of the region to capture for OCR scanning
	OCRRegionSizeFallback int               // Size of the region for OCR command fallback
	// Screenshot dimensions
	ScreenshotWidth      int
	ScreenshotHeight     int
	ScreenshotTallWidth  int
	ScreenshotTallHeight int
	ScreenshotWideWidth  int
	ScreenshotWideHeight int
}

// AudioInputDevice represents an available audio input device
type AudioInputDevice struct {
	Index      int
	Name       string
	DeviceInfo *interface{} // PortAudio device info (stored as interface{} to avoid import)
}

// GlobalState holds state for global handlers
type GlobalState struct {
	CategoryList     []string                            // Ordered list of categories for navigation
	CategoryCommands map[string][]*commands.CommandEntry // Commands grouped by category
	CurrentCategory  string                              // Currently selected category for drill-down
	CurrentCommands  []*commands.CommandEntry            // Commands in current category
	activeStates     map[types.ExecutionState]bool       // Set of active states (multiple can be active at once)
	AudioInputList   []AudioInputDevice                  // Available audio input devices
	ViewingInputs    bool                                // True when user is viewing audio inputs (for pick command routing)
}

type Functions struct {
	CaptureScreen     func() (image.Image, error)
	CaptureRegion     func(centerX, centerY, width, height int) (image.Image, image.Rectangle, error)
	GetPositionAtTime func(t time.Time) (x int, y int, ok bool)
	JumpToGhost       func() // Moves mouse to ghost position if ghosting is active
	Inputter
}

// State contains all state needed by handlers
type State struct {
	Coding *CodingState
	*MouseState
	Desktop   *DesktopState
	CV        *CVState
	Global    *GlobalState
	Dictation *DictationState

	Screen *screen.Screen
	// ScanClient talks to the standalone screend server; nil unless
	// scan_server_enabled is set, and scan handlers fall back to the local
	// OCR path when it errors
	ScanClient *screenclient.Client
	// ScanProviders is the screend provider preference order for scan/click
	// commands ("ax" before "ocr" when ax_enabled); empty means just "ocr"
	ScanProviders []string
	// For mouse commands that need the session
	lastSession mic.RecordingSession
	lookup      *commands.CommandLookup
	Functions
	sync.Mutex
	// Guards Global.activeStates. Deliberately separate from the embedded
	// Mutex: ClearOverlay/ClearMouse call SetState while holding it.
	statesMu sync.Mutex
}

type Inputter interface {
	AwaitInputWithTimeout(prompt string, allowedBreak, timeout time.Duration, model mic.ModelType) (string, bool)
}

// NewBareState creates a State with initialized containers and none of the
// side-effectful watchers; intended for tests.
func NewBareState() *State {
	return &State{
		Coding:     &CodingState{},
		Dictation:  &DictationState{},
		MouseState: &MouseState{},
		Desktop:    &DesktopState{},
		CV:         &CVState{},
		Global: &GlobalState{
			CategoryCommands: make(map[string][]*commands.CommandEntry),
			activeStates:     make(map[types.ExecutionState]bool),
		},
	}
}

// NewHandlerState creates initialized handler state
func NewHandlerState(settings config.Settings, lookup *commands.CommandLookup, inputter Inputter, screen *screen.Screen) *State {
	s := &State{
		Functions: Functions{Inputter: inputter},
		lookup:    lookup,
		Coding:    &CodingState{},
		Dictation: &DictationState{},
		MouseState: &MouseState{
			Nudge:                     20,
			CompanionMouseSensitivity: settings.CompanionMouseSensitivity,
		},
		Desktop: &DesktopState{},
		CV: &CVState{
			OverlayTimeout:        30 * time.Second, // Default 30 second timeout
			OCRRegionSize:         settings.OCRRegionSize,
			OCRRegionSizeFallback: settings.OCRRegionSizeFallback,
			ScreenshotWidth:       settings.ScreenshotWidth,
			ScreenshotHeight:      settings.ScreenshotHeight,
			ScreenshotTallWidth:   settings.ScreenshotTallWidth,
			ScreenshotTallHeight:  settings.ScreenshotTallHeight,
			ScreenshotWideWidth:   settings.ScreenshotWideWidth,
			ScreenshotWideHeight:  settings.ScreenshotWideHeight,
		},
		Global: &GlobalState{
			CategoryList:     []string{},
			CategoryCommands: make(map[string][]*commands.CommandEntry),
			CurrentCategory:  "",
			CurrentCommands:  nil,
			activeStates:     make(map[types.ExecutionState]bool),
			AudioInputList:   []AudioInputDevice{},
			ViewingInputs:    false,
		},
	}

	// Set JumpToGhost callback - moves mouse to ghost position before scrolling
	s.JumpToGhost = func() {
		eyePos, ok := eye_tracking.GetPositionAtTime(s.lastSession.RecordingStarted)
		if !ok {
			return
		}
		robotgo.Move(int(eyePos.X), int(eyePos.Y))
		// Small delay to allow jump to complete before scrolling
		time.Sleep(100 * time.Millisecond)
	}

	// Create handler state with callbacks
	s.NaturalScrolling = settings.NaturalScrolling
	s.Screen = screen
	s.CaptureScreen = screen.CaptureForOCR
	s.CaptureRegion = screen.CaptureRegionForOCR

	// Set OCR region size for background scanning
	if settings.OCRRegionSize > 0 {
		screen.SetOCRRegionSize(settings.OCRRegionSize)
	}

	if settings.OCRRegionSizeFallback > 0 {
		screen.SetOCRRegionSizeFallback(settings.OCRRegionSizeFallback)
	}

	// Set eye position getter for background OCR scanning
	screen.SetEyePositionGetter(func() (int, int, bool) {
		pos, ok := eye_tracking.GetPositionAtTime(time.Now())
		if !ok {
			return 0, 0, false
		}
		return int(pos.X), int(pos.Y), true
	})

	s.GetPositionAtTime = func(t time.Time) (int, int, bool) {
		pos, ok := eye_tracking.GetPositionAtTime(t)
		if !ok {
			return 0, 0, false
		}
		return int(pos.X), int(pos.Y), true
	}

	ch := overlay.WatchOverlay()
	go func() {
		for b := range ch {
			if b {
				s.SetState(StateOverlayHex, true)
			} else {
				s.SetState(StateOverlayHex, false)
			}
		}
	}()

	return s
}

// HasState checks if a specific state is currently active
func (s *State) HasState(state types.ExecutionState) bool {
	s.statesMu.Lock()
	defer s.statesMu.Unlock()
	return s.Global.activeStates[state]
}

// SetState enables or disables a specific state
func (s *State) SetState(state types.ExecutionState, active bool) {
	s.statesMu.Lock()
	defer s.statesMu.Unlock()
	if active {
		s.Global.activeStates[state] = true
	} else {
		delete(s.Global.activeStates, state)
	}
}

// GetActiveStates returns a copy of all active states
func (s *State) GetActiveStates() map[types.ExecutionState]bool {
	s.statesMu.Lock()
	defer s.statesMu.Unlock()
	result := make(map[types.ExecutionState]bool)
	for k, v := range s.Global.activeStates {
		result[k] = v
	}
	return result
}

// ClearAllStates removes all active states
func (s *State) ClearAllStates() {
	s.statesMu.Lock()
	defer s.statesMu.Unlock()
	s.Global.activeStates = make(map[types.ExecutionState]bool)
}

func (s *State) Stop() {
	s.ClearOverlay()
	s.ClearMouse()
	s.ClearAllStates()
}

func (s *State) ClearMouse() {
	s.SetJumping(false)
	s.SetNudging(false)
	s.SetSnapping(false)
	s.SetScrolling(false)
	if s.GetMouseDown() {
		robotgo.Toggle("left", "up")
		s.SetMouseDown(false)
		s.SetState(StateMouseDown, false)
		overlay.SendMouseDownIndicator(false)
	}
}

// Timer is automatically stopped by the SendUIScreenshot wrapper
func (s *State) ClearOverlay() {
	s.Lock()
	defer s.Unlock()

	overlay.ClearScreenshot() // This triggers SendUIScreenshot(nil) which stops the timer
	s.SetState(StateOverlayHex, false)
	s.CV.OCRLastRects = nil
	s.CV.OCRPendingAction = ""
}

// JumpToGhostIfActive jumps to ghost position if ghosting is active
func (s *State) JumpToGhostIfActive() {
	s.Lock()
	defer s.Unlock()

	if s.Ghosting && s.JumpToGhost != nil {
		log.Infof("Jumping to ghost position")
		s.JumpToGhost()
	}
}

// GetOCRRects implements OCRRectGetter interface for mouse commands
func (s *State) GetOCRRects() []image.Rectangle {
	s.Lock()
	defer s.Unlock()

	if s != nil && s.CV != nil {
		return s.CV.OCRLastRects
	}
	return nil
}

// SetRecordingStart sets the current recording start time for handlers to use
func (s *State) SetRecordingStart(rs mic.RecordingSession) {
	s.Lock()
	defer s.Unlock()

	s.lastSession = rs
}

// GetRecordingStarted returns the current recording start time
func (s *State) GetRecordingStarted() time.Time {
	s.Lock()
	defer s.Unlock()

	return s.lastSession.RecordingStarted
}

// GetLookup returns the command lookup for hints navigation
func (s *State) GetLookup() *commands.CommandLookup {
	s.Lock()
	defer s.Unlock()

	return s.lookup
}

// SetLookup updates the command lookup (used for config reload)
func (s *State) SetLookup(lookup *commands.CommandLookup) {
	s.Lock()
	defer s.Unlock()

	s.lookup = lookup
}

// GetOCRRegionSize returns the OCR region capture size (with default fallback)
func (s *State) GetOCRRegionSize() int {
	s.Lock()
	defer s.Unlock()

	if s.CV != nil && s.CV.OCRRegionSize > 0 {
		return s.CV.OCRRegionSize
	}
	return 600 // default
}

// GetOCRRegionSizeFallback returns the OCR region size for command fallback
func (s *State) GetOCRRegionSizeFallback() int {
	s.Lock()
	defer s.Unlock()

	if s.CV != nil && s.CV.OCRRegionSizeFallback > 0 {
		return s.CV.OCRRegionSizeFallback
	}
	return 300 // default
}

// EnumerateAudioInputs returns a list of available audio input devices
func (s *State) EnumerateAudioInputs() ([]AudioInputDevice, error) {
	devices, err := mic.EnumerateInputDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate audio inputs: %w", err)
	}

	var audioDevices []AudioInputDevice
	for _, device := range devices {
		audioDevices = append(audioDevices, AudioInputDevice{
			Index: device.Index,
			Name:  device.Name,
		})
	}

	return audioDevices, nil
}

// SelectAudioInput requests a change to the specified audio input device
func (s *State) SelectAudioInput(index int) error {
	return mic.RequestDeviceChange(index)
}
