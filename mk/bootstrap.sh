#!/bin/bash
# Interactive machine setup: checks everything first, then asks before
# installing anything missing. Idempotent - re-run any time.
# --yes auto-confirms (for non-interactive use).
# DISABLE="voice eye ..." (passed by make) skips what those features need.
set -euo pipefail
cd "$(dirname "$0")/.."

WHISPER_DIR=../whisper.cpp
TALON_APP=/Applications/Talon.app
TALON_USER=~/.talon/user
TALON_PLUGIN="$PWD/talon/mouse_whisperer_talon_plugin"
TALON_LINK=$TALON_USER/mouse_whisperer

DISABLE="${DISABLE:-}"
disabled() { [[ " $DISABLE " == *" $1 "* ]]; }

# portaudio + cmake (whisper.cpp build) serve voice; opencv + tesseract serve
# the local ocr pipeline
BREW_DEPS=()
disabled voice || BREW_DEPS+=(portaudio cmake)
disabled ocr || BREW_DEPS+=(opencv tesseract)

YES=0
[[ "${1:-}" == "-y" || "${1:-}" == "--yes" ]] && YES=1

if [ -t 1 ]; then GREEN=$'\e[32m'; RED=$'\e[31m'; DIM=$'\e[2m'; RESET=$'\e[0m'
else GREEN=; RED=; DIM=; RESET=; fi
ok()   { printf "  %s✓%s %s\n" "$GREEN" "$RESET" "$1"; }
miss() { printf "  %s✗%s %s\n" "$RED" "$RESET" "$1"; }

confirm() {
  [ "$YES" = 1 ] && return 0
  if [ ! -t 0 ]; then echo "  ${DIM}non-interactive shell, skipping (use --yes to auto-confirm)${RESET}"; return 1; fi
  # a failed read (EOF) must decline, or empty-reply-defaults-to-yes fires
  read -r -p "$1 [Y/n] " reply || return 1
  [[ -z "$reply" || "$reply" =~ ^[Yy] ]]
}

# ---- checks ----------------------------------------------------------------

[ -n "$DISABLE" ] && echo "Skipping what these need: $DISABLE"

echo "Toolchain"
FATAL=0
if command -v go >/dev/null; then ok "$(go version | awk '{print $3}')"; else miss "go - install from https://go.dev"; FATAL=1; fi
# swiftc builds the screend and space-helper sidecars only
if ! disabled screend || ! disabled workspace; then
  if command -v swiftc >/dev/null; then ok "swiftc"; else miss "swiftc - run: xcode-select --install"; FATAL=1; fi
fi
if command -v brew >/dev/null; then ok "brew"; else miss "brew - install from https://brew.sh"; FATAL=1; fi
HAVE_FLUTTER=0
if command -v flutter >/dev/null; then ok "flutter"; HAVE_FLUTTER=1; else miss "flutter (overlay panel)"; fi

echo "Brew packages"
MISSING_BREW=()
# ${arr[@]+...}: bash 3.2 with set -u rejects expanding an empty array
for dep in ${BREW_DEPS[@]+"${BREW_DEPS[@]}"}; do
  if brew list --versions "$dep" >/dev/null 2>&1; then ok "$dep"; else miss "$dep"; MISSING_BREW+=("$dep"); fi
done

NEED_CLONE=0; NEED_WHISPER_BUILD=0
NEED_MODELS=()
if ! disabled voice; then
  echo "whisper.cpp"
  if [ -d "$WHISPER_DIR" ]; then ok "sibling checkout $WHISPER_DIR"; else miss "sibling checkout $WHISPER_DIR"; NEED_CLONE=1; fi
  if [ -f "$WHISPER_DIR/build/bin/whisper-server" ]; then ok "whisper-server built"; else miss "whisper-server built"; NEED_WHISPER_BUILD=1; fi

  echo "Models"
  for model in tiny.en medium.en; do
    if [ -f "models/ggml-$model.bin" ]; then ok "ggml-$model.bin"; else miss "ggml-$model.bin"; NEED_MODELS+=("$model"); fi
  done
fi

HAVE_OLLAMA=0
MISSING_MODELS=()
if ! disabled ai; then
  echo "Optional (LLM commands: 'tidy that', 'revise that', spoken code)"
  # the models the config actually references; [ {] keeps voice_model= from matching
  OLLAMA_MODELS=$(grep -ohE '[ {]model = "[^"]*"' config/*.toml 2>/dev/null | sed 's/.*"\(.*\)"/\1/' | sort -u)
  if command -v ollama >/dev/null; then ok "ollama"; HAVE_OLLAMA=1; else miss "ollama"; fi
  OLLAMA_LIST="$(ollama list 2>/dev/null || true)"
  for m in $OLLAMA_MODELS; do
    if [[ "$OLLAMA_LIST" == *"$m"* ]]; then ok "$m"; else miss "$m"; MISSING_MODELS+=("$m"); fi
  done
fi

HAVE_TALON=0; GAZE_INSTALLED=0; HAVE_TOBII=0
OLD_PLUGINS=()
if ! disabled eye; then
  echo "Optional (gaze control: Tobii Eye Tracker 5 via Talon)"
  if [ -d "$TALON_APP" ] && [ -d "$TALON_USER" ]; then ok "Talon"; HAVE_TALON=1
  elif [ -d "$TALON_APP" ]; then miss "Talon user folder - launch Talon once to create $TALON_USER"
  else miss "Talon - install from https://talonvoice.com"; fi
  # Tobii AB's USB vendor id is 0x2104 (8452)
  if ioreg -p IOUSB -l -w0 2>/dev/null | grep -q '"idVendor" = 8452'; then ok "Tobii eye tracker connected"; HAVE_TOBII=1
  else miss "Tobii eye tracker connected"; fi
  if [ "$HAVE_TALON" = 1 ]; then
    if [ "$(readlink "$TALON_LINK" 2>/dev/null)" = "$TALON_PLUGIN" ]; then ok "plugin linked at $TALON_LINK"; GAZE_INSTALLED=1
    else miss "plugin linked at $TALON_LINK"; fi
    # older copies (any other folder serving the gaze port) would load the
    # same Talon actions twice
    for dir in "$TALON_USER"/*/; do
      dir=${dir%/}
      [ "$dir" = "$TALON_LINK" ] && continue
      if grep -qs '50055' "$dir/plugin/mouse/mouse.py"; then
        miss "old plugin copy $dir"; OLD_PLUGINS+=("$dir")
      fi
    done
  fi
fi

echo ""
if [ "$FATAL" = 1 ]; then
  echo "Install the missing toolchain above, then re-run make bootstrap."
  exit 1
fi

# ---- installs, each confirmed ----------------------------------------------

INCOMPLETE=0

if [ "$HAVE_FLUTTER" = 0 ]; then
  if confirm "Install Flutter (brew install --cask flutter)?"; then
    brew install --cask flutter
    echo "  run 'flutter doctor' to finish Flutter setup"
  else
    echo "  skipped - 'make run' needs the overlay; './run --dev' works without it"
    INCOMPLETE=1
  fi
fi

if [ ${#MISSING_BREW[@]} -gt 0 ]; then
  if confirm "Install brew packages: ${MISSING_BREW[*]}?"; then
    brew install "${MISSING_BREW[@]}"
  else
    echo "  skipped - the build will fail without them"
    INCOMPLETE=1
  fi
fi

if [ "$NEED_CLONE" = 1 ]; then
  if confirm "Clone whisper.cpp into $WHISPER_DIR?"; then
    git clone https://github.com/ggerganov/whisper.cpp.git "$WHISPER_DIR"
    NEED_CLONE=0
  else
    echo "  skipped - speech recognition needs the whisper server"
    INCOMPLETE=1
  fi
fi

if [ "$NEED_CLONE" = 0 ] && [ "$NEED_WHISPER_BUILD" = 1 ]; then
  if confirm "Build whisper-server (takes a few minutes)?"; then
    cmake -S "$WHISPER_DIR" -B "$WHISPER_DIR/build"
    cmake --build "$WHISPER_DIR/build" -j --config Release
    NEED_WHISPER_BUILD=0
  else
    INCOMPLETE=1
  fi
fi

if ! disabled voice && [ -f "$WHISPER_DIR/build/bin/whisper-server" ]; then
  install -d build
  cp "$WHISPER_DIR/build/bin/whisper-server" build/server
  echo "whisper-server -> build/server"
fi

# tiny.en (~75MB) drives the streaming ticks, medium.en (~1.5GB) the
# accurate finals. The download script lives in the whisper.cpp checkout.
if [ ${#NEED_MODELS[@]} -gt 0 ]; then
  if [ "$NEED_CLONE" = 1 ]; then
    echo "Skipping model downloads - no whisper.cpp checkout"
    INCOMPLETE=1
  else
    for model in "${NEED_MODELS[@]}"; do
      case $model in tiny.en) size="~75MB";; *) size="~1.5GB";; esac
      if confirm "Download $model model ($size)?"; then
        (cd "$WHISPER_DIR/models" && ./download-ggml-model.sh "$model")
        install -d models
        cp "$WHISPER_DIR/models/ggml-$model.bin" models/
      else
        INCOMPLETE=1
      fi
    done
  fi
fi

# ollama and its models are optional - declining doesn't mark the run incomplete
if ! disabled ai && [ "$HAVE_OLLAMA" = 0 ]; then
  if confirm "Install ollama (brew install ollama)? Optional - powers the LLM commands."; then
    brew install ollama
    HAVE_OLLAMA=1
    echo "  start it with: brew services start ollama"
  else
    echo "  skipped (optional)"
  fi
fi

if [ "$HAVE_OLLAMA" = 1 ] && [ ${#MISSING_MODELS[@]} -gt 0 ]; then
  for m in "${MISSING_MODELS[@]}"; do
    if confirm "Pull ollama model $m (a few GB)?"; then
      ollama pull "$m" || echo "  pull failed - is ollama running? (brew services start ollama)"
    else
      echo "  skipped (optional)"
    fi
  done
fi

# gaze control is optional - declining doesn't mark the run incomplete
if [ "$HAVE_TALON" = 1 ] && { [ "$GAZE_INSTALLED" = 0 ] || [ ${#OLD_PLUGINS[@]} -gt 0 ]; }; then
  note=""
  [ "$HAVE_TOBII" = 0 ] && note=" (no tracker connected right now)"
  if confirm "Set up gaze control with a Tobii Eye Tracker 5$note? Links the Talon plugin into $TALON_USER."; then
    if [ ${#OLD_PLUGINS[@]} -gt 0 ]; then
      # moved, not deleted: outside user/ so Talon stops loading them
      backup=~/.talon/mouse_whisperer_backup_$(date +%Y%m%d%H%M%S)
      mkdir -p "$backup"
      for dir in "${OLD_PLUGINS[@]}"; do
        mv "$dir" "$backup/"
        echo "  moved old copy $dir -> $backup/"
      done
    fi
    if [ -e "$TALON_LINK" ] && [ ! -L "$TALON_LINK" ]; then
      echo "  $TALON_LINK exists and isn't a link - move it aside and re-run"
    else
      ln -sfn "$TALON_PLUGIN" "$TALON_LINK"
      echo "  linked $TALON_LINK -> $TALON_PLUGIN"
      echo "  restart Talon, then say 'control mouse' to toggle gaze control"
    fi
  else
    echo "  skipped (optional)"
  fi
fi

echo ""
if [ "$INCOMPLETE" = 1 ]; then
  echo "Bootstrap incomplete (skipped items above) - re-run make bootstrap any time."
  exit 1
fi
echo "Bootstrap complete. Next: make build${DISABLE:+ DISABLE=\"$DISABLE\"} && make run"
