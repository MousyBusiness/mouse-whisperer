//go:build !noeye

package eye_tracking

import (
	"github.com/mousybusiness/mouse_whisperer/internal/debuglog"
	log "github.com/sirupsen/logrus"
	"golang.design/x/hotkey"
)

func (m *mouseeye) registerHotKeys() {
	m.hkSmooth = hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift, hotkey.ModCtrl, hotkey.ModOption}, hotkey.KeyJ)
	if err := m.hkSmooth.Register(); err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}
	m.hkInteresting = hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift, hotkey.ModCtrl, hotkey.ModOption}, hotkey.KeyK)
	if err := m.hkInteresting.Register(); err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}
	m.hkJump = hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift, hotkey.ModCtrl, hotkey.ModOption}, hotkey.KeyL)
	if err := m.hkJump.Register(); err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}

	log.Debugf("Registered hot keys")
}
func (m *mouseeye) watchHotKeys() {
	go m.watchSmoothHotkey()
	go m.watchInterestingPointsHotkey()
	go m.watchJumpHotkey()
}

func (m *mouseeye) watchSmoothHotkey() {
	for {
		select {
		case <-m.ctx.Done():
			m.state.SetNudging(false)
			return
		case <-m.hkSmooth.Keydown():
			if debuglog.Enabled() {
				debuglog.AppendCommand([]string{"hotkey", "hotkey", "", "nudge", ""})
			}
			m.state.SetNudging(true)
		case <-m.hkSmooth.Keyup():
			m.state.SetNudging(false)
		}

	}
}

func (m *mouseeye) watchInterestingPointsHotkey() {
	for {
		select {
		case <-m.ctx.Done():
			m.state.SetSnapping(false)
			return
		case <-m.hkInteresting.Keydown():
			if debuglog.Enabled() {
				debuglog.AppendCommand([]string{"hotkey", "hotkey", "", "snap", ""})
			}
			m.state.SetSnapping(true)
		case <-m.hkInteresting.Keyup():
			m.state.SetSnapping(false)
		}
	}
}

func (m *mouseeye) watchJumpHotkey() {
	for {
		select {
		case <-m.ctx.Done():
			m.state.SetJumping(false)
			return
		case <-m.hkJump.Keydown():
			if debuglog.Enabled() {
				debuglog.AppendCommand([]string{"hotkey", "hotkey", "", "jump", ""})
			}
			m.state.SetJumping(true)
		case <-m.hkJump.Keyup():
			m.state.SetJumping(false)
		}
	}
}
