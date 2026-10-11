package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
)

// Config is the root configuration structure
type Config struct {
	Settings Settings `toml:"settings"`
	//Commands    *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, commands.Command]] `toml:"-"`        // Built from TOML with preserved order
	RawCommands map[string]map[string]commands.Command `toml:"commands"` // For TOML unmarshaling
	meta        toml.MetaData
}

// Settings contains global configuration options
type Settings struct {
	NaturalScrolling      bool     `toml:"natural_scrolling"`                // Invert scroll direction for natural scrolling (macOS default)
	CategoryOrder         []string `toml:"category_order"`                   // Order of categories as displayed in the UI
	OCRRegionSize         int      `toml:"ocr_region_size"`                  // Size of the region to capture for OCR background scanning
	OCRRegionSizeFallback int      `toml:"ocr_region_size_command_fallback"` // Size of the region for OCR command fallback (default 300)
	// Screenshot dimensions (width x height in logical pixels)
	ScreenshotWidth      int `toml:"screenshot_width"`
	ScreenshotHeight     int `toml:"screenshot_height"`
	ScreenshotTallWidth  int `toml:"screenshot_tall_width"`
	ScreenshotTallHeight int `toml:"screenshot_tall_height"`
	ScreenshotWideWidth  int `toml:"screenshot_wide_width"`
	ScreenshotWideHeight int `toml:"screenshot_wide_height"`
	// Companion app sensitivity multipliers (1.0 = default)
	CompanionScrollSensitivity float64 `toml:"companion_scroll_sensitivity"`
	CompanionMouseSensitivity  float64 `toml:"companion_mouse_sensitivity"`
	CompanionFlingDistance     int     `toml:"companion_fling_distance"`
	// Streaming transcription: evaluate commands while speech is in progress
	StreamingASR            bool  `toml:"streaming_asr"`             // enable mid-utterance transcription and evaluation
	StreamingTickMs         int   `toml:"streaming_tick_ms"`         // snapshot interval while speaking (default 350)
	StreamingMaxSeconds     int   `toml:"streaming_max_seconds"`     // stop ticking beyond this utterance length (default 12)
	StreamStabilityTicks    int   `toml:"stream_stability_ticks"`    // ticks a token must be unchanged before commands act on it (default 2)
	ProseSalvageMaxTokens   int   `toml:"prose_salvage_max_tokens"`  // prose utterances at most this long still get fuzzy/OCR salvage (default 2)
	DictationEnabled        *bool `toml:"dictation_enabled"`         // accumulate unmatched speech in the dictation buffer (default true)
	DictationAccurateFinals *bool `toml:"dictation_accurate_finals"` // re-transcribe dictation finals with the accurate model (default true)
	CommandAccurateFinals   *bool `toml:"command_accurate_finals"`   // re-transcribe finals that fire trailer commands (click/move/select/scan) with the accurate model (default true)
	DictationBreakMs        int   `toml:"dictation_break_ms"`        // silence that ends an utterance while dictating (default 1200; commands use 500)
	// Standalone screen-perception server (screend): scan commands query it
	// and fall back to the local OCR path when it is down
	ScanServerEnabled bool   `toml:"scan_server_enabled"` // route OCR scans through screend
	ScanServerURL     string `toml:"scan_server_url"`     // default http://localhost:51013
	AXEnabled         bool   `toml:"ax_enabled"`          // query the accessibility element tree before OCR for scan/click commands
}

// Load reads and parses all TOML configuration files from a directory
func Load(dirPath string) (*Config, error) {
	cfg := &Config{
		RawCommands: make(map[string]map[string]commands.Command),
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Skip non-TOML files and files starting with _ (like _backup.toml)
		if !strings.HasSuffix(name, ".toml") || strings.HasPrefix(name, "_") {
			continue
		}

		filePath := filepath.Join(dirPath, name)
		var partial Config
		_, err := toml.DecodeFile(filePath, &partial)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", name, err)
		}

		// Merge settings (later files override earlier ones)
		if partial.Settings.CategoryOrder != nil {
			cfg.Settings.CategoryOrder = partial.Settings.CategoryOrder
		}
		if partial.Settings.NaturalScrolling {
			cfg.Settings.NaturalScrolling = partial.Settings.NaturalScrolling
		}
		if partial.Settings.OCRRegionSize > 0 {
			cfg.Settings.OCRRegionSize = partial.Settings.OCRRegionSize
		}
		if partial.Settings.OCRRegionSizeFallback > 0 {
			cfg.Settings.OCRRegionSizeFallback = partial.Settings.OCRRegionSizeFallback
		}
		if partial.Settings.ScreenshotWidth > 0 {
			cfg.Settings.ScreenshotWidth = partial.Settings.ScreenshotWidth
		}
		if partial.Settings.ScreenshotHeight > 0 {
			cfg.Settings.ScreenshotHeight = partial.Settings.ScreenshotHeight
		}
		if partial.Settings.ScreenshotTallWidth > 0 {
			cfg.Settings.ScreenshotTallWidth = partial.Settings.ScreenshotTallWidth
		}
		if partial.Settings.ScreenshotTallHeight > 0 {
			cfg.Settings.ScreenshotTallHeight = partial.Settings.ScreenshotTallHeight
		}
		if partial.Settings.ScreenshotWideWidth > 0 {
			cfg.Settings.ScreenshotWideWidth = partial.Settings.ScreenshotWideWidth
		}
		if partial.Settings.ScreenshotWideHeight > 0 {
			cfg.Settings.ScreenshotWideHeight = partial.Settings.ScreenshotWideHeight
		}
		if partial.Settings.CompanionScrollSensitivity != 0 {
			cfg.Settings.CompanionScrollSensitivity = partial.Settings.CompanionScrollSensitivity
		}
		if partial.Settings.CompanionMouseSensitivity != 0 {
			cfg.Settings.CompanionMouseSensitivity = partial.Settings.CompanionMouseSensitivity
		}
		if partial.Settings.CompanionFlingDistance > 0 {
			cfg.Settings.CompanionFlingDistance = partial.Settings.CompanionFlingDistance
		}
		if partial.Settings.StreamingASR {
			cfg.Settings.StreamingASR = partial.Settings.StreamingASR
		}
		if partial.Settings.StreamingTickMs > 0 {
			cfg.Settings.StreamingTickMs = partial.Settings.StreamingTickMs
		}
		if partial.Settings.StreamingMaxSeconds > 0 {
			cfg.Settings.StreamingMaxSeconds = partial.Settings.StreamingMaxSeconds
		}
		if partial.Settings.StreamStabilityTicks > 0 {
			cfg.Settings.StreamStabilityTicks = partial.Settings.StreamStabilityTicks
		}
		if partial.Settings.ProseSalvageMaxTokens > 0 {
			cfg.Settings.ProseSalvageMaxTokens = partial.Settings.ProseSalvageMaxTokens
		}
		if partial.Settings.DictationEnabled != nil {
			cfg.Settings.DictationEnabled = partial.Settings.DictationEnabled
		}
		if partial.Settings.DictationAccurateFinals != nil {
			cfg.Settings.DictationAccurateFinals = partial.Settings.DictationAccurateFinals
		}
		if partial.Settings.CommandAccurateFinals != nil {
			cfg.Settings.CommandAccurateFinals = partial.Settings.CommandAccurateFinals
		}
		if partial.Settings.DictationBreakMs > 0 {
			cfg.Settings.DictationBreakMs = partial.Settings.DictationBreakMs
		}
		if partial.Settings.ScanServerEnabled {
			cfg.Settings.ScanServerEnabled = partial.Settings.ScanServerEnabled
		}
		if partial.Settings.ScanServerURL != "" {
			cfg.Settings.ScanServerURL = partial.Settings.ScanServerURL
		}
		if partial.Settings.AXEnabled {
			cfg.Settings.AXEnabled = partial.Settings.AXEnabled
		}

		// Merge commands
		for category, cmds := range partial.RawCommands {
			if cfg.RawCommands[category] == nil {
				cfg.RawCommands[category] = make(map[string]commands.Command)
			}
			for cmdName, cmd := range cmds {
				cfg.RawCommands[category][cmdName] = cmd
			}
		}
	}

	// Set defaults for sensitivity (1.0 = no scaling)
	if cfg.Settings.CompanionScrollSensitivity == 0 {
		cfg.Settings.CompanionScrollSensitivity = 1.0
	}
	if cfg.Settings.CompanionMouseSensitivity == 0 {
		cfg.Settings.CompanionMouseSensitivity = 1.0
	}
	if cfg.Settings.CompanionFlingDistance == 0 {
		cfg.Settings.CompanionFlingDistance = 100
	}
	if cfg.Settings.StreamingTickMs == 0 {
		cfg.Settings.StreamingTickMs = 350
	}
	if cfg.Settings.StreamingMaxSeconds == 0 {
		cfg.Settings.StreamingMaxSeconds = 12
	}
	if cfg.Settings.DictationBreakMs == 0 {
		cfg.Settings.DictationBreakMs = 1200
	}
	if cfg.Settings.StreamStabilityTicks == 0 {
		cfg.Settings.StreamStabilityTicks = 2
	}
	if cfg.Settings.ProseSalvageMaxTokens == 0 {
		cfg.Settings.ProseSalvageMaxTokens = 2
	}
	if cfg.Settings.DictationEnabled == nil {
		// *bool because the merge pattern above cannot express a default
		// of true for a plain bool
		enabled := true
		cfg.Settings.DictationEnabled = &enabled
	}
	if cfg.Settings.DictationAccurateFinals == nil {
		enabled := true
		cfg.Settings.DictationAccurateFinals = &enabled
	}
	if cfg.Settings.CommandAccurateFinals == nil {
		enabled := true
		cfg.Settings.CommandAccurateFinals = &enabled
	}
	if cfg.Settings.ScanServerURL == "" {
		cfg.Settings.ScanServerURL = "http://localhost:51013"
	}

	return cfg, nil
}

//
//// buildOrderedCommands creates ordered maps from raw command data using TOML key order
//func buildOrderedCommands(raw map[string]map[string]commands.Command, meta toml.MetaData) *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, Command]] {
//	result := orderedmap.NewOrderedMap[string, *orderedmap.OrderedMap[string, Command]]()
//
//	// Track category and command order from TOML metadata
//	categoryOrder := []string{}
//	commandOrder := make(map[string][]string)
//	seenCategories := make(map[string]bool)
//	seenCommands := make(map[string]map[string]bool)
//
//	// Keys() returns all keys in document order
//	for _, key := range meta.Keys() {
//		// Looking for keys like ["commands", "global"] for category
//		if len(key) == 2 && key[0] == "commands" {
//			category := strings.ToLower(key[1])
//			if !seenCategories[category] {
//				categoryOrder = append(categoryOrder, category)
//				seenCategories[category] = true
//			}
//		}
//		// Looking for keys like ["commands", "global", "enable_voice"] for command
//		if len(key) == 3 && key[0] == "commands" {
//			category := strings.ToLower(key[1])
//			commandName := strings.ToLower(key[2])
//
//			// Ensure category is in order list
//			if !seenCategories[category] {
//				categoryOrder = append(categoryOrder, category)
//				seenCategories[category] = true
//			}
//
//			if seenCommands[category] == nil {
//				seenCommands[category] = make(map[string]bool)
//			}
//			if !seenCommands[category][commandName] {
//				commandOrder[category] = append(commandOrder[category], commandName)
//				seenCommands[category][commandName] = true
//			}
//		}
//	}
//
//	// Build ordered maps using the extracted order
//	for _, categoryName := range categoryOrder {
//		rawCategory, exists := raw[categoryName]
//		if !exists {
//			continue
//		}
//
//		categoryMap := orderedmap.NewOrderedMap[string, Command]()
//
//		// Add commands in order
//		for _, cmdName := range commandOrder[categoryName] {
//			if cmd, exists := rawCategory[cmdName]; exists {
//				categoryMap.Set(cmdName, cmd)
//			}
//		}
//
//		result.Set(categoryName, categoryMap)
//	}
//
//	return result
//}
