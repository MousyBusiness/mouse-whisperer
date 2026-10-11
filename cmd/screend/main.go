// screend is the standalone screen-perception server: it supervises the
// native screend-helper (ScreenCaptureKit capture + Vision OCR + AX element
// queries) and serves clickable targets over local HTTP. See internal/screend.
package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/ax"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/server"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/vision"
)

func main() {
	port := flag.Int("port", 51013, "listen port (loopback only)")
	helperPath := flag.String("helper", "build/screend-helper", "path to the screend-helper binary")
	ttl := flag.Duration("ttl", 5*time.Second, "default snapshot max age")
	axTTL := flag.Duration("ax-ttl", time.Second, "snapshot max age for the ax provider (UI trees go stale faster than captures are worth re-taking)")
	level := flag.String("level", "accurate", "vision recognition level: accurate|fast (fast has poor recall on editor themes)")
	includeImage := flag.Bool("include-image", false, "capture the screenshot PNG with every snapshot (debugging; slows captures)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	engine := helper.NewEngine(*helperPath)
	defer engine.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	if err := engine.Ping(pingCtx); err != nil {
		// not fatal: the helper respawns on demand, and the first capture
		// may still succeed once Screen Recording permission is granted
		log.Warnf("helper not responding yet: %v", err)
	} else if trusted := engine.AXTrusted(); trusted != nil && !*trusted {
		log.Warn("helper lacks Accessibility permission: ax queries will fail (grant it to the terminal/app that launches screend)")
	}
	cancel()

	srv := server.NewServer(*port, *ttl, engine.Running,
		vision.NewOCRProvider(engine, *level, *includeImage),
		ax.NewProvider(engine))
	srv.SetTTL(api.ProviderAX, *axTTL)
	srv.AXTrusted = engine.AXTrusted
	log.Infof("screend listening on 127.0.0.1:%d (helper: %s, level: %s, ttl: %s, ax-ttl: %s)", *port, *helperPath, *level, *ttl, *axTTL)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("screend: %v", err)
	}
}
