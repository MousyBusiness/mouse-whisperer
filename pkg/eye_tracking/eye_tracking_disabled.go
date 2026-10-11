//go:build noeye

package eye_tracking

import (
	"context"

	"github.com/mousybusiness/mouse_whisperer/pkg/screen"
)

// Run never connects to an eye tracker in a noeye build; it only waits for
// shutdown. The position cache stays empty, so GetPositionAtTime always
// reports no gaze and callers fall back to the mouse position.
func Run(ctx context.Context, state EyeTrackingState, screen *screen.Screen, w, h int) {
	<-ctx.Done()
}
