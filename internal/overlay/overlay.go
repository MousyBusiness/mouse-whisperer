package overlay

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"image"
	"image/png"
	"net/http"

	gen "github.com/mousybusiness/mouse_whisperer/gen/mouse_whisperer/overlay/v1"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

const port = "51012"

var (
	serverOnline     = true
	serverOnlineMux  sync.RWMutex
	lastCheckTime    time.Time
	checkInterval    = time.Second      // Initial check interval
	maxCheckInterval = 30 * time.Second // Max backoff interval
	checkIntervalMux sync.Mutex
	overlayCh        chan bool
)

func WatchOverlay() chan bool {
	overlayCh = make(chan bool, 1)
	return overlayCh
}

func ShowToast(title string, msg string, level string) error {
	toast := &gen.Toast{
		Title:   title,
		Message: msg,
		Level:   level,
	}

	b, err := protojson.Marshal(toast)
	if err != nil {
		return err
	}

	log.Debugf("show toast")
	return doPost("/toast", b)
}

// SendUIMessage sends brief command text shown at bottom of screen (auto-dismiss ~1s)
func SendUIMessage(msg string) error {
	displayMsg := &gen.DisplayMessage{
		Content: &gen.DisplayMessage_CommandText{
			CommandText: msg,
		},
	}

	b, err := protojson.Marshal(displayMsg)
	if err != nil {
		return err
	}

	return doPost("/display", b)
}

// SendCommandsList sends a commands list panel shown on the right (tap to close)
func SendCommandsList(title string, items []string, dismissSeconds int32) error {
	displayMsg := &gen.DisplayMessage{
		Content: &gen.DisplayMessage_CommandsList{
			CommandsList: &gen.CommandsList{
				Title:          title,
				Items:          items,
				DismissSeconds: dismissSeconds,
			},
		},
	}

	b, err := protojson.Marshal(displayMsg)
	if err != nil {
		return err
	}

	return doPost("/display", b)
}

// SendCommandConfig sends a command config/spec panel shown on the right (tap to close)
func SendCommandConfig(cfg *gen.CommandConfig) error {
	displayMsg := &gen.DisplayMessage{
		Content: &gen.DisplayMessage_CommandConfig{
			CommandConfig: cfg,
		},
	}

	b, err := protojson.Marshal(displayMsg)
	if err != nil {
		return err
	}

	return doPost("/display", b)
}

func ClearScreenshot() error {
	log.Infof("ClearScreenshot called")
	return SendUIScreenshot(nil)
}

// Reads the file from the given path, encodes its content to base64,
// and sends it in a POST request to the specified URL.
func SendUIScreenshot(img image.Image) error {
	if img == nil {
		overlayCh <- false
	} else {
		overlayCh <- true
	}
	//if img != nil {
	//	// Cancel any existing timer
	//	if s.OCR.OverlayTimer != nil {
	//		s.OCR.OverlayTimer.Stop()
	//	}
	//
	//	// Start auto-dismiss timer
	//	s.OCR.OverlayTimer = time.AfterFunc(s.OCR.OverlayTimeout, func() {
	//		log.Info("Overlay timeout - auto-dismissing")
	//		s.ClearOverlay()
	//	})
	//} else {
	//	// Clearing screenshot - stop timer but don't reset state
	//	// (state will be managed by the calling handler if needed)
	//	if s.OCR.OverlayTimer != nil {
	//		s.OCR.OverlayTimer.Stop()
	//		s.OCR.OverlayTimer = nil
	//	}
	//}
	//
	var gimg *gen.Image
	if img != nil {
		buf := new(bytes.Buffer)
		// BestSpeed: this encodes a full-screen (mostly transparent) overlay
		// on the interactive path; default compression costs hundreds of ms
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		err := enc.Encode(buf, img)
		if err != nil {
			fmt.Printf("Failed to encode image: %s\n", err)
			return err
		}
		value := buf.Bytes()
		buf = nil
		// Encode the file content to base64
		encodedContent := base64.StdEncoding.EncodeToString(value)
		gimg = &gen.Image{
			ImageBase64: encodedContent,
		}
	} else {
		gimg = &gen.Image{
			Delete: true,
		}
	}

	b, err := protojson.Marshal(gimg)
	if err != nil {
		return err
	}

	return doPost("/image", b)
}

func SendEyesPosition(vector *util.Vector) error {
	var eyesPos *gen.EyesPosition
	if vector != nil {
		eyesPos = &gen.EyesPosition{
			X: vector.X,
			Y: vector.Y,
		}
	} else {
		eyesPos = &gen.EyesPosition{
			Clear: true,
		}
	}

	b, err := protojson.Marshal(eyesPos)
	if err != nil {
		return err
	}

	return doPost("/eyes", b)
}

func ConfirmAIResponsePosition(prompt, aiResponse string) error {
	resp := &gen.AIResponse{
		Prompt:     prompt,
		AiResponse: aiResponse,
	}

	b, err := protojson.Marshal(resp)
	if err != nil {
		return err
	}

	return doPost("/ai", b)
}

func DismissAIResponse() error {
	resp := &gen.AIResponse{
		Delete: true,
	}

	b, err := protojson.Marshal(resp)
	if err != nil {
		return err
	}

	return doPost("/ai", b)
}

// DismissCommands dismisses any active commands UI panels (commandsList or commandConfig)
func DismissCommands() error {
	// Send an empty DisplayMessage (notSet) to clear all command UI panels
	displayMsg := &gen.DisplayMessage{}

	b, err := protojson.Marshal(displayMsg)
	if err != nil {
		return err
	}

	return doPost("/display", b)
}

// SendUIAction sends a UI action (scroll, dismiss, etc.) to the frontend
func SendUIAction(action string, amount int32) error {
	uiAction := &gen.UIAction{
		Action: action,
		Amount: amount,
	}

	b, err := protojson.Marshal(uiAction)
	if err != nil {
		return err
	}

	return doPost("/action", b)
}

// SendTunnelVision toggles tunnel vision mode in the overlay
func SendTunnelVision(enabled bool, radius int32, opacity float64, blendDistance float64) error {
	tv := &gen.TunnelVision{
		Enabled:       enabled,
		Radius:        radius,
		Opacity:       opacity,
		BlendDistance: blendDistance,
	}

	b, err := protojson.Marshal(tv)
	if err != nil {
		return err
	}

	return doPost("/tunnel", b)
}

// SendGhostVisibility toggles the ghost cursor indicator visibility
func SendGhostVisibility(visible bool) error {
	gv := &gen.GhostVisibility{
		Visible: visible,
	}

	b, err := protojson.Marshal(gv)
	if err != nil {
		return err
	}

	return doPost("/ghost", b)
}

// SendRecordingIndicator sends recording state to the UI overlay
func SendRecordingIndicator(recording bool, timeoutMs int32) error {
	ri := &gen.RecordingIndicator{
		Recording: recording,
		TimeoutMs: timeoutMs,
	}

	b, err := protojson.Marshal(ri)
	if err != nil {
		return err
	}

	return doPost("/recording", b)
}

// SendMouseDownIndicator sends mouse button down state to the UI overlay
func SendMouseDownIndicator(active bool) error {
	mdi := &gen.MouseDownIndicator{
		Active: active,
	}

	b, err := protojson.Marshal(mdi)
	if err != nil {
		return err
	}

	return doPost("/mousedown", b)
}

// SendNotListeningIndicator sends disabled voice state to the UI overlay
func SendNotListeningIndicator(enabled bool) error {
	nli := &gen.NotListeningIndicator{
		Enabled: enabled,
	}

	b, err := protojson.Marshal(nli)
	if err != nil {
		return err
	}

	return doPost("/notlistening", b)
}

// SendDictation shows/updates the live dictation panel. text is the
// committed buffer; pending is the in-flight utterance shown dimmed.
// hints lists the spoken control commands (the overlay caches the last
// non-empty set, so nil is fine for incremental updates).
func SendDictation(text, pending string, hints []string) error {
	d := &gen.Dictation{
		Text:    text,
		Pending: pending,
		Hints:   hints,
	}

	b, err := protojson.Marshal(d)
	if err != nil {
		return err
	}

	return doPost("/dictation", b)
}

// DismissDictation hides the dictation panel
func DismissDictation() error {
	d := &gen.Dictation{
		Delete: true,
	}

	b, err := protojson.Marshal(d)
	if err != nil {
		return err
	}

	return doPost("/dictation", b)
}

// isServerOnline checks if the server is currently considered online
func isServerOnline() bool {
	serverOnlineMux.RLock()
	defer serverOnlineMux.RUnlock()
	return serverOnline
}

// setServerOnline updates the server online status
func setServerOnline(online bool) {
	serverOnlineMux.Lock()
	defer serverOnlineMux.Unlock()

	wasOnline := serverOnline
	serverOnline = online

	if !wasOnline && online {
		// Server came back online, reset check interval
		log.Info("UI server is back online")
		resetCheckInterval()
	} else if wasOnline && !online {
		log.Info("UI server is offline, will retry with exponential backoff")
	}
}

// resetCheckInterval resets the exponential backoff interval
func resetCheckInterval() {
	checkIntervalMux.Lock()
	defer checkIntervalMux.Unlock()
	checkInterval = time.Second
}

// increaseCheckInterval increases the check interval with exponential backoff
func increaseCheckInterval() time.Duration {
	checkIntervalMux.Lock()
	defer checkIntervalMux.Unlock()
	checkInterval *= 2
	if checkInterval > maxCheckInterval {
		checkInterval = maxCheckInterval
	}
	return checkInterval
}

// shouldCheckServer determines if we should attempt to check the server now
func shouldCheckServer() bool {
	checkIntervalMux.Lock()
	defer checkIntervalMux.Unlock()

	if time.Since(lastCheckTime) >= checkInterval {
		lastCheckTime = time.Now()
		return true
	}
	return false
}

// checkServerHealth performs a health check on the server
func checkServerHealth() bool {
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	resp, err := client.Get("http://localhost:" + port + "/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500
}

func doPost(path string, b []byte) error {
	// If server is offline, check if we should try again
	if !isServerOnline() {
		if shouldCheckServer() {
			if checkServerHealth() {
				setServerOnline(true)
			} else {
				// Increase backoff silently
				increaseCheckInterval()
				return nil
			}
		} else {
			// Still in backoff period, don't check yet
			return nil
		}
	}

	req, err := http.NewRequest("POST", "http://localhost:"+port+path, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		setServerOnline(false)
		return err
	}
	defer resp.Body.Close()

	// Mark server as online since we got a response
	if !isServerOnline() {
		setServerOnline(true)
	}

	return nil
}
