// Package features reports which optional modules were compiled in.
//
// Every module is on by default and removed with a negative build tag, so a
// plain `go build`/`go test` (and gopls) always sees the full app:
//
//	novoice      mic capture (portaudio) and whisper transcription
//	noeye        eye-tracker client, gaze hotkeys and gaze mouse commands
//	noocr        local OpenCV/tesseract pipeline: snap-to-element and local OCR
//	noscreend    screend scan-server client (Vision OCR + accessibility targets)
//	nocompanion  companion phone app server (UDP + bonjour)
//	noai         ollama-backed commands (ai prompts, dictation tidy/revise)
//	noworkspace  Mission Control space aliases (space-helper)
//
// The mk/build.mk DISABLE variable maps names to these tags.
package features

// Feature is one optional module and whether this binary includes it.
type Feature struct {
	Name    string
	Enabled bool
}

// All lists every optional module in a stable order.
func All() []Feature {
	return []Feature{
		{"voice", Voice},
		{"eye", Eye},
		{"ocr", OCR},
		{"screend", Screend},
		{"companion", Companion},
		{"ai", AI},
		{"workspace", Workspace},
	}
}

// Disabled returns the names of modules compiled out of this binary.
func Disabled() []string {
	var names []string
	for _, f := range All() {
		if !f.Enabled {
			names = append(names, f.Name)
		}
	}
	return names
}
