//go:build !noeye

package eye_tracking

import (
	"bufio"
	"context"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/pkg/screen"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

// TODO interrupt mouse move if key up
// TODO cluster intention better, time out lastV

type mouseeye struct {
	sync.Mutex
	ctx           context.Context
	hkSmooth      *hotkey.Hotkey
	hkInteresting *hotkey.Hotkey
	hkJump        *hotkey.Hotkey
	delta         util.Vector
	cache         EyePositionQueue
	state         EyeTrackingState
	//drifting bool
	w int
	h int

	screen *screen.Screen
}

// System Preferences > Security and Privacy > Privacy > Accessibility and Screen Recording
func Run(ctx context.Context, state EyeTrackingState, screen *screen.Screen, w, h int) {
	me := &mouseeye{
		ctx:    ctx,
		state:  state,
		screen: screen,
		w:      w,
		h:      h,
		cache:  EyePositionQueue{size: 1000 / 20}, // event every 20ms, 1 seconds
	}

	// Set global cache so other packages can access it
	globalCache = &me.cache

	mainthread.Init(me.mainFn)
}

func (m *mouseeye) mainFn() {
	log.SetLevel(log.DebugLevel)

	m.registerHotKeys()
	m.watchHotKeys()

	moveCh := make(chan util.Vector)
	go m.doMoves(moveCh)

	eyePositionCh := make(chan util.Vector)
	go m.acceptMoveEvents(eyePositionCh)

	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case v := <-eyePositionCh:
				// log.Debugf("Eye position: %v", v)
				m.cache.Enqueue(PositionEvent{
					Position:  v,
					Timestamp: time.Now(),
				})

				// Always send eye position to UI (ghost visibility is controlled separately)
				go func() {
					overlay.SendEyesPosition(&v)
				}()

				if m.state.GetNudging() {
					select {
					case moveCh <- v:
					default:
						log.Debugf("Dropped %v", time.Now())
					}

					continue
				}

				if m.state.GetSnapping() {
					scaledV := v.Mul(robotgo.ScaleF(robotgo.DisplayID))
					// log.Debugf("Snapping: raw eye pos=%+v, scaled=%+v", v, scaledV)
					result, ok := m.screen.FindNearest(scaledV)
					if !ok {
						// no detected elements to snap to: follow the gaze
						result = v
					}
					// log.Debugf("Snapping: moving to %+v", result)
					robotgo.Move(int(result.X), int(result.Y))
					continue
				}

				if m.state.GetJumping() {
					robotgo.Move(int(v.X), int(v.Y))
					continue
				}

			default: // Non-blocking, dispose if busy
				time.Sleep(time.Millisecond * 5)
			}
		}
	}()

	<-m.ctx.Done()
	close(moveCh)
	if m.hkSmooth != nil {
		_ = m.hkSmooth.Unregister()
		log.Printf("hotkey: %v is unregistered\n", m.hkSmooth)
	}
	if m.hkInteresting != nil {
		_ = m.hkInteresting.Unregister()
	}
	if m.hkJump != nil {
		_ = m.hkJump.Unregister()
	}
}
func (m *mouseeye) acceptMoveEvents(ch chan util.Vector) {
	const (
		serverAddr        = "127.0.0.1:50055"
		initialBackoff    = 500 * time.Millisecond
		maxBackoff        = 30 * time.Second
		backoffMultiplier = 2.0
	)

	backoff := initialBackoff

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
		if err != nil {
			log.Warnf("Eye tracking server unavailable (%s), retrying in %v...", err, backoff)
			select {
			case <-m.ctx.Done():
				return
			case <-time.After(backoff):
				backoff = time.Duration(float64(backoff) * backoffMultiplier)
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		// Connection successful - reset backoff
		backoff = initialBackoff
		log.Infof("Connected to eye tracking server at %v", conn.RemoteAddr())

		// Handle the connection
		m.handleEyeTrackingConnection(conn, ch)

		// Connection closed or errored - will retry
		log.Warn("Eye tracking connection lost, attempting reconnect...")
	}
}

func (m *mouseeye) handleEyeTrackingConnection(conn net.Conn, ch chan util.Vector) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		// Set read deadline to detect stale connections
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		line, err := reader.ReadString('\n')
		if err != nil {
			if m.ctx.Err() != nil {
				// Context cancelled, clean exit
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout - connection might be stale, try to continue
				continue
			}
			log.Warnf("Error reading from eye tracking connection: %v", err)
			return // Return to trigger reconnection
		}

		s := strings.TrimSpace(line)
		split := strings.Split(s, ",")

		if len(split) != 4 {
			log.Warnf("Invalid split: %v", s)
			continue
		}
		x, _ := strconv.ParseFloat(strings.TrimSpace(strings.Replace(split[0], "x:", "", 1)), 32)
		// Clip X so don't hit handoff zone
		maxX := float64(m.w) * 0.9975
		if x >= maxX {
			x = maxX
		}

		y, _ := strconv.ParseFloat(strings.TrimSpace(strings.Replace(split[1], "y:", "", 1)), 32)
		v := util.Vector{X: x, Y: y}
		// A frame arrives about every 30ms
		ch <- v
	}
}

func nudge(target util.Vector) {
	//robotgo.Move(int(target.x), int(target.y))
	//return
	mx, my := robotgo.Location()
	mv := util.Vector{X: float64(mx), Y: float64(my)}
	normal := target.Sub(mv).Normalize()
	d := mv.Distance(target)
	speed := d / 200
	mul := 10 * speed
	if mul < 2 {
		mul = 2
	}
	dx := int(math.Round(normal.X * mul))
	dy := int(math.Round(normal.Y * mul))

	robotgo.MoveRelative(dx, dy)
}

func getDistance(target util.Vector) float64 {
	x, y := robotgo.GetMousePos()
	mouseV := util.Vector{X: float64(x), Y: float64(y)}
	return target.Distance(mouseV)
}

func getDirection(target util.Vector) util.Vector {
	x, y := robotgo.GetMousePos()
	mouseV := util.Vector{X: float64(x), Y: float64(y)}
	return target.Sub(mouseV).Normalize()
}

type sample struct {
	v  util.Vector
	ts time.Time
}

func (m *mouseeye) doMoves(ch chan util.Vector) {
	robotgo.MouseSleep = 0

	var samples []sample
	for latest := range ch {
		d := getDistance(latest)
		if d < 2 {
			samples = nil
			continue
		} else if d < 200 {
			nudge(latest)
			continue
		}

		temp := samples[:0]
		for _, x := range samples {
			since := time.Since(x.ts)
			if since < time.Millisecond*100 {
				temp = append(temp, x)
			} else {
				// Drop stale samples - e.g. if just starting pressing key and samples existed before
			}
		}
		samples = temp

		// Collect more point samples
		if len(samples) < 3 {
			samples = append(samples, sample{v: latest, ts: time.Now()})
			continue
		}

		l := float64(len(samples))
		// Determine grouping
		// 1. Find the average x, y
		var ax float64
		var ay float64
		for _, s := range samples {
			ax += s.v.X
			ay += s.v.Y
		}
		ax = ax / l
		ay = ay / l
		center := util.Vector{X: ax, Y: ay}

		threshold := float64(5) // px
		var inside []sample
		for _, s := range samples {
			d := math.Abs(s.v.Distance(center))
			if d <= threshold {
				inside = append(inside, s)
			}
		}

		ratioInside := float64(len(inside)) / l

		if ratioInside < 1 {
			samples = samples[1:] // Pop the first same to see if it improves
			continue
		}

		// TODO should target be latest or center?
		target := latest

		var speed = float64(.3)
		if d > 1500 {
			robotgo.Move(int(target.X), int(target.Y))
		} else {
			if d <= 1000 {
				speed = .8
			} else if d < 500 {
				speed = 1.5
			} else if d < 200 {
				// Don't use directional drift for small moves
				//direction = vector{}
				speed = 3.0
			}

			// Do move in backgroundj
			go robotgo.MoveSmooth(int(target.X), int(target.Y), speed, speed+.1, 2)
		}

		samples = nil

		timeout := time.NewTimer(time.Millisecond * 400)
		ticker := time.NewTicker(time.Millisecond * 5)
		for {
			select {
			case <-m.ctx.Done():
				return
			default:
			}

			d := getDistance(target)
			if d < 4 {
				//m.drifting = true
				break
			}
			select {
			case latest := <-ch:
				// Continue to consume events while moving
				samples = append(samples, sample{v: latest, ts: time.Now()})
				if len(samples) > 5 {
					samples = samples[1:] // Limit to the last 5 samples
				}
			case <-ticker.C:
			// Timeout
			case <-timeout.C:
				log.Warnf("Timeout waiting for move to complete")
				break
			}
		}

	}
}
