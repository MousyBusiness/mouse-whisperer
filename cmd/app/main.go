package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/asr"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/debuglog"
	"github.com/mousybusiness/mouse_whisperer/internal/evaluation"
	"github.com/mousybusiness/mouse_whisperer/internal/execution"
	"github.com/mousybusiness/mouse_whisperer/internal/features"
	"github.com/mousybusiness/mouse_whisperer/internal/registry"
	"github.com/mousybusiness/mouse_whisperer/internal/screenclient"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/internal/voice"
	"github.com/mousybusiness/mouse_whisperer/pkg/eye_tracking"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/screen"

	log "github.com/sirupsen/logrus"
)

func main() {
	debug := flag.Bool("debug", false, "write rolling CSV logs and POI frame dumps under /tmp/mousewhisperer")
	flag.Parse()
	if *debug {
		if err := debuglog.Init(debuglog.DefaultRoot); err != nil {
			panic(err)
		}
	}

	// Ignore SIGPIPE to prevent crashes when writing to closed connections
	// This commonly occurs when the eye tracking server or overlay disconnects
	signal.Ignore(syscall.SIGPIPE)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case s := <-sig:
			fmt.Printf("Received signal: %v\n", s)
			cancel()
			return
		case <-ctx.Done():
			log.Warn("Sig exit")
		}
	}()

	log.SetLevel(log.DebugLevel)

	// Load config
	configPath := "config"
	cfg, err := config.Load(configPath)
	if err != nil {
		panic(err)
	}

	// Find and use the largest display
	numDisplays := robotgo.DisplaysNum()
	var maxArea, largestIdx int
	var largestW, largestH int

	for i := 0; i < numDisplays; i++ {
		_, _, w, h := robotgo.GetDisplayBounds(i)
		area := w * h
		log.Debugf("Display %d: %dx%d (area: %d)", i, w, h, area)
		if area > maxArea {
			maxArea = area
			largestIdx = i
			largestW = w
			largestH = h
		}
	}

	robotgo.DisplayID = largestIdx
	log.Infof("Using display %d (%dx%d, scale: %.1fx) as largest display", largestIdx, largestW, largestH, robotgo.ScaleF(robotgo.DisplayID))

	if disabled := features.Disabled(); len(disabled) > 0 {
		log.Infof("Built without: %s", strings.Join(disabled, ", "))
	}

	screen := screen.New()
	screen.SetDebug(*debug)

	recordingRequestCh := make(chan *mic.RecordingRequest)
	recordingSessionsCh := make(chan mic.RecordingSession, 10)
	input := voice.NewInput(recordingRequestCh)

	// Registry before lookup: commands naming a handler from a feature
	// compiled out of this binary are pruned before the lookup is built
	reg := registry.NewHandlerRegistry()
	lstate := state.NewHandlerState(cfg.Settings, nil, input, screen)

	// Scan server (screend): scan commands query it for OCR targets and the
	// stream evaluator prewarms it at utterance start; nil client keeps every
	// existing path untouched. Set up before the handlers register: without
	// local OCR, the scan commands only exist when there is a client
	var scanClient *screenclient.Client
	switch {
	case cfg.Settings.ScanServerEnabled && !features.Screend:
		// left out on purpose, so the shipped setting isn't a misconfiguration
		log.Debug("scan_server_enabled is set but this binary was built without screend")
	case cfg.Settings.ScanServerEnabled:
		scanClient = screenclient.New(cfg.Settings.ScanServerURL)
		lstate.ScanClient = scanClient
		if cfg.Settings.AXEnabled {
			// accessibility elements answer first; OCR still sees whatever
			// the AX tree doesn't expose
			lstate.ScanProviders = []string{api.ProviderAX, api.ProviderOCR}
		}
		log.Infof("Scan server enabled at %s (ax: %v)", cfg.Settings.ScanServerURL, cfg.Settings.AXEnabled)
	}

	reg.RegisterAll(lstate)

	if changes := cfg.PruneUnhandled(reg.Has); len(changes) > 0 {
		log.Infof("Adapted %d commands to this build: %s", len(changes), strings.Join(changes, ", "))
	}
	lookup := commands.NewCommandLookup(cfg.RawCommands, cfg.Settings.CategoryOrder)
	lstate.SetLookup(lookup)

	executor := execution.NewExecutor(lookup, reg, lstate)

	eval, err := evaluation.NewEvaluator(cfg, lookup, executor, input, lstate)
	if err != nil {
		panic(err)
	}

	startCompanion(ctx, cfg, eval, lstate)

	if features.Voice {
		// Streaming: transcribe mid-utterance snapshots and evaluate commands on
		// the fly; a nil stream channel keeps the legacy batch-per-utterance path
		var streamEventCh chan mic.StreamEvent
		if cfg.Settings.StreamingASR {
			streamEventCh = make(chan mic.StreamEvent, 8)
			partialCh := asr.Run(ctx, streamEventCh)
			streamEval := evaluation.NewStreamEvaluator(eval, evaluation.StreamConfig{
				StabilityTicks:          cfg.Settings.StreamStabilityTicks,
				ProseSalvageMaxTokens:   cfg.Settings.ProseSalvageMaxTokens,
				DictationEnabled:        *cfg.Settings.DictationEnabled,
				AccurateDictationFinals: *cfg.Settings.DictationAccurateFinals,
				AccurateCommandFinals:   *cfg.Settings.CommandAccurateFinals,
			})
			if scanClient != nil {
				streamEval.OnUtteranceStart = func() { go scanClient.Prewarm() }
			}
			go streamEval.Run(ctx, partialCh)
			log.Infof("Streaming ASR enabled (tick: %dms, stability: %d ticks)",
				cfg.Settings.StreamingTickMs, cfg.Settings.StreamStabilityTicks)
		}
		voice.Run(ctx, eval, recordingRequestCh, recordingSessionsCh, streamEventCh, mic.StreamOptions{
			TickInterval: time.Duration(cfg.Settings.StreamingTickMs) * time.Millisecond,
			MaxDuration:  time.Duration(cfg.Settings.StreamingMaxSeconds) * time.Second,
			// dictation is in progress exactly while the buffer has text
			DictationBreak: time.Duration(cfg.Settings.DictationBreakMs) * time.Millisecond,
			Dictating:      lstate.Dictation.HasText,
		})
	}

	// Blocks until shutdown: with eye tracking it owns the main thread for
	// the hotkey event loop; without, it just waits on ctx
	eye_tracking.Run(ctx, lstate, screen, largestW, largestH)

	<-ctx.Done()
}
