---
name: update-dictation
description: Add or change dictation-mode behavior in mouse-whisperer — control words (like insert/yolo/tidy), buffer operations, panel behavior, or streaming evaluation tweaks. Use whenever the user asks to add/change what they can say WHILE DICTATING, modify the dictation buffer or its panel, or adjust streaming/prose behavior — phrases like "add a dictation command", "when I'm dictating I want to say X", "change how the buffer/panel works". For ordinary (non-dictation) voice commands use write-command instead. Encodes the control-word allowlist, the trigger-fork pattern, buffer invariants, and the test/validation checklist.
tools: Read, Edit, Write, Bash, Grep, Glob
---

# Update the dictation system

Dictation mode: while the buffer is non-empty ("sticky"), every utterance APPENDS as prose
except standalone control words. This skill is the checklist for changing that system without
breaking its invariants. For plain voice commands (no dictation involvement), use the
`write-command` skill instead.

## The file map (a change usually touches most of these)

| Concern | File |
|---|---|
| Control-word allowlist + streaming evaluation | `internal/evaluation/stream.go` (`controlMethods` map) |
| Dictation handlers (`dictation_*`) | `internal/impl/ai.go` |
| Buffer operations | `internal/state/state.go` (`DictationState`) |
| Command config (triggers, state gating) | `config/text.toml` (commit words `insert`/`take`/`reject` live in `config/ai.toml`) |
| Routing tests | `internal/evaluation/stream_test.go` (`streamLookup` + `TestStreamControlWords`, `TestControlHints`) |
| Buffer op tests | `internal/state/dictation_test.go` |
| Panel (rarely needed — hints are automatic) | `internal/overlay/overlay.go`, `_/overlay/lib/main.dart` |

## Adding a control word (the common case)

1. **Handler** in `internal/impl/ai.go`, method name `dictation_<verb>`. Read args from the
   action's `args` map; read a spoken count via `xctx.Get(ctx).Multiplier`. End every handler
   by refreshing the UI: `overlay.SendDictation(state.Dictation.Text(), "", nil)` if text
   remains, else clear `StateDictating` AND `overlay.DismissDictation()` (see invariant below).
2. **Allowlist** it in `controlMethods` (`internal/evaluation/stream.go`) mapping method →
   debuglog kind. **Skipping this step makes sticky mode treat the word as prose** — the #1
   silent failure.
3. **Config** entry:
   - New trigger → dedicated command gated `states = ["dictating"]` (see `text.tidy`,
     `text.yolo`). Use `disable_fuzzy = true` for short triggers.
   - Trigger already exists for a non-dictation command → DO NOT duplicate the trigger
     (the lookup map silently overwrites); fork the existing command with action-level
     states + skip callbacks (next section).
4. **Tests**: mirror the command into `streamLookup()` in `stream_test.go`; add a
   `TestStreamControlWords` case (pre-seed `h.state.Dictation.Append(...)` AND
   `h.state.SetState(state.StateDictating, true)`); `TestControlHints` count will need
   updating. Buffer ops get cases in `internal/state/dictation_test.go`.
5. The panel hint chips update automatically (`controlHints` derives them from config) —
   no Flutter change.

## Forking an existing trigger on dictation state

Pattern (from `text.delete_word` / `text.cap_that` in `config/text.toml`): one command, two
actions — the dictation action gated `states = ["dictating"]`, the original action gated
`excludes = ["dictating"]` — plus skip callbacks so the inactive one is passed over:

```toml
actions = [
  { id = "buffer", method = "dictation_<verb>", states = ["dictating"] },
  { id = "keys", method = "<original method>", args = { ... }, excludes = ["dictating"] },
]
action_callbacks = [
  { for = "buffer", result = "skip", method = "__continue__" },
  { for = "buffer", result = "success", method = "__done__" },
  { for = "keys", result = "skip", method = "__continue__" },
]
```

`controlWord` only inspects state-ACTIVE actions, so the keystroke action doesn't disqualify
the command as a control word while dictating.

## Invariants (violating these causes subtle breakage)

- **`StateDictating` ⟺ buffer non-empty.** `appendProse` sets it; every commit/discard path
  must clear it AND dismiss the panel. A handler that empties the buffer but leaves the state
  set strands `dictating`-gated commands active; the reverse makes control words stop matching.
- **Control words are exact-match, standalone, no fuzzy.** Trailing punctuation is stripped
  and a trailing spoken number becomes the multiplier for free. Never add fuzzy to a control
  trigger — prose containing the word must keep appending. If ASR mangles a word, add an alias
  trigger (homophones: the user's "prose" arrives as "pros" sometimes).
- **Buffer mutations happen on the stream exec worker** (handlers invoked through it) —
  don't mutate `DictationState` from new goroutines; ordering with "insert" depends on it.
- **`DictationState` methods take the internal mutex** — add new ops as methods, not external
  field access.
- Blocking work in a handler (LLM calls like `dictation_tidy`) stalls the worker — acceptable
  but show progress via the panel `pending` field first (see tidy's "· tidying ·").
- A handler may prompt for follow-up speech via `captureInput` with an override
  `commands.InputAction` (see `dictation_revise`) — the recording indicator shows
  automatically, "abort"/"cancel"/"nevermind" and pre-speech silence both cancel (empty
  string back), and the mic's prompted-takeover branch keeps this safe even if the user
  speaks before the prompt attaches. Always restore the panel on the cancel path.

## Validate before reporting done

```bash
export CGO_CPPFLAGS="-I/opt/homebrew/include" CGO_LDFLAGS="-L/opt/homebrew/lib"
go build ./internal/... ./cmd/...
go test ./internal/evaluation/ ./internal/state/
# TOML parse (see write-command skill for the toml_check.go snippet)
```

If the overlay proto changed: `make generate` needs a valid buf token; the local-plugin
fallback is documented in HANDOVER.md R13. Then `cd _/overlay && flutter analyze --no-pub`
(one pre-existing companion error is expected).

Report to the user: trigger phrases added/changed, files touched, and whether a restart
(`./run`) is needed (it always is — config loads once).

## Current control words (for conflict checks)

insert · take · reject · yolo · scratch that · delete that · delete word [N] · tidy that /
tidy · title that · lowercase that · replay that / replay — plus the panel shows whatever `controlHints` derives, so
check the panel after a restart rather than trusting this list.
