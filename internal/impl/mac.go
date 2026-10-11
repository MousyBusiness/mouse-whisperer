package impl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	"github.com/lithammer/fuzzysearch/fuzzy"
	log "github.com/sirupsen/logrus"
)

func RegisterMacHandlers(registry Registry, state *state.State) {
	// Desktop handlers
	registry.Register("desktop_open", func(ctx context.Context, args map[string]any) (context.Context, error) {

		_, appName, err := captureInput(ctx, state, &commands.InputAction{
			Prompt:  "A MacOS application name e.g. Goland, Terminal, Cursor. output: ",
			Break:   750 * time.Millisecond,
			Timeout: 3 * time.Second,
		})
		if err != nil {
			return ctx, fmt.Errorf("failed to capture input: %v", err)
		}

		if appName == "" {
			return ctx, fmt.Errorf("no application name provided")
		}
		// Use 'open -a' which is more reliable than robotgo.ActiveName:
		// - Works case-insensitively
		// - Opens the app if not running, activates if already running
		// - Can find apps by partial name
		if err := exec.Command("open", "-a", appName).Run(); err != nil {
			log.Debugf("Direct open failed for '%s', trying fuzzy search", appName)
			// Fallback: fuzzy search against installed applications
			if match := fuzzyFindApp(appName); match != "" {
				log.Infof("Fuzzy matched '%s' -> '%s'", appName, match)
				if err := exec.Command("open", "-a", match).Run(); err != nil {
					log.Warnf("Failed to open fuzzy-matched app '%s': %v", match, err)
				}
			} else {
				log.Warnf("No fuzzy match found for '%s'", appName)
			}
		}

		return ctx, nil
	})

	registry.Register("desktop_switch", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.KeyDown("lcmd")
		robotgo.MilliSleep(100)
		robotgo.KeyTap("tab")

		_, direction, err := captureInput(ctx, state, &commands.InputAction{
			Prompt:  "Left or right. output: ",
			Break:   750 * time.Millisecond,
			Timeout: 3 * time.Second,
		})
		if err != nil {
			return ctx, fmt.Errorf("failed to capture input: %v", err)
		}

		c, i := multiplierCommand(util.CommonMistakes(strings.ToLower(direction)))
		if err != nil {
			return ctx, fmt.Errorf("failed to parse multiplier: %v", err)
		}
		if c == "right" {
			for j := 0; j < i; j++ {
				robotgo.KeyTap("tab", "lcmd")
			}
		}
		if c == "left" {
			for j := 0; j < i; j++ {
				robotgo.KeyTap("tab", "lcmd", "lshift")
			}
		}
		robotgo.MilliSleep(100)
		robotgo.KeyTap("enter")
		robotgo.MilliSleep(100)

		robotgo.KeyUp("lcmd")
		return ctx, nil
	})

	registry.Register("desktop_toggle", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.KeyDown("lcmd")
		robotgo.MilliSleep(100)
		robotgo.KeyTap("tab")
		robotgo.MilliSleep(100)
		robotgo.KeyUp("lcmd")
		return ctx, nil
	})
}

// fuzzyFindApp searches for an application matching the query in /Applications and ~/Applications.
// Returns the best matching app name, or empty string if no match found.
func fuzzyFindApp(query string) string {
	var apps []string

	// Collect apps from /Applications and ~/Applications
	appDirs := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil {
		appDirs = append(appDirs, filepath.Join(home, "Applications"))
	}

	for _, dir := range appDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, ".app") {
				apps = append(apps, strings.TrimSuffix(name, ".app"))
			}
		}
	}

	// Use fuzzy search with case-insensitive ranking
	matches := fuzzy.RankFindFold(query, apps)
	if len(matches) > 0 {
		return matches[0].Target
	}
	return ""
}
