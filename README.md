# Mouse Whisperer

Hands-free macOS control by voice. Commands fire **while you're still
speaking** ("up up down left" executes as a chain), on-screen text is
clickable by name ("click save"), and anything that isn't a command lands in
a live dictation buffer you commit with "insert". All speech recognition is
local (whisper.cpp).

## What you can say

| You say | What happens |
|---|---|
| "up up down", "scroll down 3" | keystroke chains, fired mid-utterance |
| "click save" / "move readme" / "select main" | finds the text on screen (accessibility tree first, OCR fallback) and acts on it — a unique or gaze-nearest match acts immediately, otherwise numbered boxes appear and you say the number |
| "scan handler" | full-screen search with numbered targets |
| *any prose* … "insert" | builds in the live dictation panel, then types it ("take" copies, "reject" discards, "scratch that", "tidy that", "question that") |
| "alias workspace mail" / "go to mail" | name and switch Mission Control spaces |

Commands are TOML — see `config/*.toml` (one file per category, knobs in
`config/settings.toml`). Personal commands go in `config/local.toml`, which
is gitignored. The gaze-nearest matching uses eye-tracker data
bridged from the optional Talon plugin — see `talon/README.md`.

## Requirements

- macOS on Apple Silicon (paths assume `/opt/homebrew`), Xcode Command Line
  Tools (`swiftc`), [Homebrew](https://brew.sh)
- Go, and Flutter if you want the overlay panel
- optional: [Ollama](https://ollama.com) for the LLM commands ("tidy that",
  "revise that", spoken code) — bootstrap offers to install it and pull the
  models the config uses (gemma3:4b, llama3.1, codellama:7b-instruct)

## Setup and build

```sh
make bootstrap   # one-time machine setup
make build       # everything: app, screend + native helper, space-helper, overlay
```

Bootstrap is interactive: it checks the toolchain (Go, Xcode Command Line
Tools, Homebrew, Flutter), the brew packages (portaudio, opencv, tesseract,
cmake), the [whisper.cpp](https://github.com/ggerganov/whisper.cpp) sibling
checkout (`../whisper.cpp` → `build/server`) and the models (tiny.en drives
the streaming ticks, medium.en the accurate finals) — then asks before
installing anything missing. It also offers gaze control for a Tobii Eye
Tracker 5: it checks for Talon and the tracker, and links the Talon plugin
into `~/.talon/user` (see `talon/README.md`). It's idempotent — re-run it any time;
`mk/bootstrap.sh --yes` auto-confirms.

### Leaving modules out

Every module is built by default. `DISABLE` lists modules to compile out —
pass it to `make bootstrap` and `make build`, or persist it in the untracked
`mk/local.mk`. Bootstrap then skips their dependencies and `./run` skips their
services. `make bootstrap-lite` / `make build-lite` use the lite profile
(`LITE_DISABLE` in `mk/features.mk`): eye tracking with snap-to-element and
local OCR only.

| Module | Removes | Drops |
|---|---|---|
| `voice` | mic capture, whisper transcription, voice commands | portaudio, whisper.cpp + models |
| `eye` | eye-tracker client, gaze hotkeys, gaze mouse commands | — |
| `ocr` | local OpenCV/tesseract: snap-to-element, local OCR scans | opencv, tesseract |
| `screend` | scan-server client (Vision OCR + accessibility targets) | screend + Swift helper |
| `companion` | companion phone app server | — |
| `ai` | Ollama commands (AI prompts, dictation tidy/revise) | Ollama + models |
| `workspace` | Mission Control space aliases | space-helper |

At startup, commands whose handlers are compiled out are dropped from the
grammar, and callbacks falling back to a missing handler are neutralised; both
are logged. Scan commands need local `ocr` or an enabled scan server.
`make -f mk/dev.mk check-features` vets each module off on its own and vets
and tests the lite profile and everything off. For ad-hoc Go commands each
module's build tag is `no<module>`.

## Run

```sh
make run     # dev mode + overlay panel  (= ./run --dev)
./run        # production mode (runs build/app + build/overlay.app; make build first)
```

`./run` flags: `--dev` (go run, and `flutter run` for the overlay, instead
of the built artifacts), `--no-overlay` (skip the Flutter panel, which starts
by default), `--debug`. The script starts the two whisper servers (:51010 fast,
:51011 accurate), screend (:51013) when built, the overlay (:51012), and the
app.

### First run: macOS permissions

Grant these to **the terminal you launch from** (permissions attach to the
launching process):

- **Microphone** — speech capture
- **Accessibility** — synthetic keystrokes/clicks, and the AX scan provider
  (verify with `curl -s localhost:51013/healthz` → `"ax_trusted": true`)
- **Screen Recording** — screenshots for OCR

## Development

```sh
# unit tests
CGO_CPPFLAGS="-I/opt/homebrew/include" CGO_LDFLAGS="-L/opt/homebrew/lib" \
  go test ./internal/... ./pkg/...

make -f mk/dev.mk help   # development commands: debug (rolling CSV logs),
                         # ps/kill (process management), generate (protobuf;
                         # a failed run needs `git checkout -- gen/`),
                         # single-artifact builds
```

The overlay window size is currently set manually in
`_/overlay/macos/Runner/MainFlutterWindow.swift`.

`HANDOVER.md` is the risk register for the streaming-voice internals —
read it before touching `pkg/mic`, `internal/asr`, or
`internal/evaluation/stream.go`. Feature plans live in `docs/plans/`.

## Troubleshooting

- **"Server not built"** — run `make bootstrap`.
- **App exits immediately** — usually a missing microphone permission; run
  `make -f mk/dev.mk debug` and read the output.
- **"click/scan X" slow or falling back** — screend isn't built
  (`make build`) or died (`tmp/screend.log`); `ax_trusted: false` on
  `/healthz` means the launching terminal lacks Accessibility permission.
- **Ports already in use** — `make -f mk/dev.mk kill`.
