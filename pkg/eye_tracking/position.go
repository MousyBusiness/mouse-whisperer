package eye_tracking

import (
	"sync"
	"time"

	"github.com/mousybusiness/mouse_whisperer/pkg/util"
)

// Global state for ghost position tracking
var (
	globalCache   *EyePositionQueue
	globalCacheMu sync.RWMutex
)

// GetPositionAtTime returns the eye position nearest to the given time from the cache
func GetPositionAtTime(t time.Time) (util.Vector, bool) {
	globalCacheMu.RLock()
	defer globalCacheMu.RUnlock()

	if globalCache == nil {
		return util.Vector{}, false
	}

	event, ok := globalCache.FindNearest(t)
	if !ok {
		return util.Vector{}, false
	}

	return event.Position, true
}

type EyeTrackingState interface {
	GetGhosting() bool
	GetNudging() bool
	SetNudging(b bool)
	GetNaturalScrolling() bool
	GetSnapping() bool
	SetSnapping(b bool)
	GetJumping() bool
	SetJumping(b bool)
	GetScrolling() bool
}
