# Plan: Accessibility (AXUIElement) target provider

Status: implemented 2026-07-09 — phases 0–3 built and tested (phase 0's trust
check was inconclusive in the build sandbox; verify `ax_trusted` via
`curl -s localhost:51013/healthz` from the real launch topology, and measure
walk latency with `AX_INTEGRATION=1 go test ./internal/screend/ax/ -run
TestAXIntegration -v`). Phase 4 follow-ons remain open.

## Goal

Add a macOS Accessibility query path that returns the frontmost window's actionable UI
elements (role, title, bounds) as a first-class target source, so "click save" resolves
against the real UI tree in milliseconds instead of a screen capture + OCR round trip.
OCR remains the fallback for apps with poor AX trees.

Value beyond speed: AX sees elements OCR cannot (hidden by tooltip-only labels,
enabled/disabled state), reports exact clickable bounds rather than glyph boxes, and later
unlocks press-without-moving-the-mouse, focused-element awareness (secure-input detection
for dictation), and menu-bar navigation.

## Where it fits

screend already has exactly the right seam: providers are registered by name in
`cmd/screend/main.go:41`, selected per-request via `TargetsRequest.Provider`
(`internal/screend/api/types.go:32`), cached per-provider in `server/store.go`, and ranked
by `scan.Rank` (`internal/screend/scan/match.go:18`). The app consumes targets through
`internal/impl/ocr.go` (`actViaServer`, `scanViaServer`, `showTargetsOverlay`) and clicks
their centers with robotgo. None of that plumbing changes — this plan adds a second
`scan.Provider` named `ax`.

There is no accessibility code in the repo today; this is greenfield.

## Design decisions

1. **AX queries live in the existing screend-helper, as new ops.** The helper already has
   process supervision (lazy spawn, crash respawn, wedge kill — `vision/engine.go`), NDJSON
   framing, and a Makefile build. A second native process would duplicate all of that. The
   helper stays single-request-in-flight; AX tree walks are fast enough that queuing behind
   an OCR capture is acceptable for v1 (measure in the spike).
2. **The engine/codec move out of the `vision` package.** `vision.Engine` and the NDJSON
   codec are OCR-agnostic; extract them to `internal/screend/helper` so the `ax` provider
   and the OCR provider share one supervised process without an import of `vision` from
   `ax`. Mechanical refactor, no behavior change.
3. **AX frames bypass `ToLogical`.** Vision boxes are normalized, bottom-left,
   image-relative; AX positions (`kAXPositionAttribute`) are already global logical points,
   top-left origin — the exact space of `api.Rect` and robotgo. Running them through
   `convert.go` would corrupt them. The helper emits AX frames in a separate response field
   with its own documented coordinate contract.
4. **v1 clicks stay on robotgo.** Clicking the target rect center reuses
   `performTargetAction` (`impl/ocr.go:49`) untouched. `AXPress` (clicking occluded or
   scrolled-away elements without moving the cursor) needs element-reference caching across
   requests — deferred to phase 4.
5. **Fallback is client-side and sequential: ax → ocr → legacy Tesseract.** The app tries
   provider `ax`, falls back to `ocr` on zero/weak matches, then the existing local path.
   Server-side merging of AX and OCR targets (dedup by rect overlap) is deferred — an AX
   "Save" button and the OCR word "Save" are the same target and merging them correctly is
   its own problem.

## Protocol additions (helper NDJSON)

New op `ax_query`. Request fields (beyond `id`/`op`):

| field | type | meaning |
|---|---|---|
| `pid` | int, optional | target app; absent = frontmost application |
| `max_elements` | int, optional | walk cap, helper default 2000 |
| `max_depth` | int, optional | tree depth cap, helper default 30 |

Response carries a new `elements` array (the `observations`/`meta` fields stay OCR-only):

| field | type | meaning |
|---|---|---|
| `role` | string | AX role (AXButton, AXTextField, …) |
| `subrole` | string | e.g. AXSecureTextField; empty if none |
| `title` | string | first non-empty of title → description → value (text fields) → linked title element |
| `enabled` | bool | from the enabled attribute |
| `x`,`y`,`w`,`h` | double | **global logical points, top-left origin** — no conversion in Go |
| `actions` | []string | supported AX actions (press, confirm, …) |

Also: extend `ping`'s response with `ax_trusted` (from `AXIsProcessTrusted`) so `/healthz`
can surface permission state instead of queries failing opaquely.

Helper implementation constraints:

- Walk the frontmost app's focused window (`kAXFocusedWindowAttribute`) children,
  depth-first with the caps above; collect only roles in an actionable allowlist
  (buttons, links, text fields/areas, checkboxes, radio buttons, pop-up/combo boxes,
  menu items, tabs) plus any element advertising a press action.
- Set `AXUIElementSetMessagingTimeout` (~500ms) on the app element so an unresponsive app
  degrades to an error response instead of wedging the helper (the engine would kill and
  respawn it, which costs the backoff window).
- Skip zero-size and off-screen elements.
- Sort output by reading order (y, then x). `scan.Rank`'s multi-word phrase matching
  requires adjacent items in visual order (`match.go:99-131`); emitting sorted spares the
  Go side a re-sort.
- For Chromium/Electron apps, set the `AXManualAccessibility` attribute on the app element
  before walking, otherwise the tree is empty.

## Phases

### Phase 0 — Spike (decision gate, ~half day)

A throwaway `ax_query` that only counts elements and reports `ax_trusted`, exercised on:
Finder, a native app (Notes), an Electron app, Chrome or Safari, and a JetBrains IDE.
Answers three questions this plan depends on:

1. **TCC attribution.** Accessibility permission attaches to the *responsible process*.
   In dev the helper runs under `run` → terminal (likely already trusted for robotgo); a
   future bundled app changes attribution. Confirm `AXIsProcessTrusted` is true in the
   helper as actually spawned by screend, and document which TCC entry grants it.
2. **Latency envelope.** Tree-walk time on a heavyweight window (IDE, busy browser tab)
   with the caps applied. Target: p95 under ~150ms. If browsers blow the cap, tighten the
   role allowlist before touching depth.
3. **Title coverage.** What fraction of visibly-clickable controls have a non-empty title
   through the fallback chain. This calibrates how much the OCR fallback still matters
   (icon-only buttons stay uncovered — a shape-detection spike failed its gate 2026-07
   and was removed; see commit 83acc17 if it's ever worth revisiting).

Exit criteria: trusted in the dev topology, p95 walk under ~150ms with caps, and no
app in the sample set hangs or crashes the helper.

### Phase 1 — Helper op

Implement `ax_query` per the protocol above. Split the helper source into
`main.swift` + `ax.swift` (the Makefile `helper` target at `Makefile:35-39` compiles a
single file today; pass both to swiftc).

### Phase 2 — Engine extraction + Go provider

- Extract engine + codec from `vision` to `internal/screend/helper` (decision 2).
- New `internal/screend/ax` provider implementing `scan.Provider`
  (`scan/scan.go:12-16`): `Name()` returns `ax`; `Snapshot()` issues `ax_query` and maps
  elements to `scan.Item` (label = title, rect passed through, source = `ax`,
  disabled elements dropped); `Query()` delegates to `scan.Rank`.
- Verify `scan.Rank` tier behavior against multi-word AX titles (an AX item carries a
  whole title where OCR items carry single words) — substring/phrase tiers should cover
  it; add per-word splitting only if ranking proves wrong.
- Register in `cmd/screend/main.go:41`. `/healthz` and the `store.go` cache pick it up by
  name automatically. Give the AX provider a shorter cache TTL than OCR's 5s (flag,
  default ~1s) — snapshots are cheap and UI state moves fast.

### Phase 3 — App routing

- `internal/impl/ocr.go` hardcodes `Provider: "ocr"` at `:114` and `:157`; parameterize
  and try `ax` first, falling back per decision 5. Weak-match heuristic: fall through when
  the AX result set is empty or the best match is fuzzy-tier only.
- New `settings.toml` key `ax_enabled` (default false until the spike passes), plumbed in
  `internal/config/config.go` alongside `scan_server_enabled`.
- Overlay disambiguation works unchanged via `showTargetsOverlay` provided
  `api.Target.ID` stays equal to slice index (`server.go:148` invariant).
- Prewarm (`OnUtteranceStart`) keeps hitting the OCR provider only; AX needs no warmup.

### Phase 4 — Follow-ons (each its own small plan when scheduled)

- **`ax_focused` op**: role/subrole of the focused element. First consumer: suppress the
  dictation buffer when the focused element is a secure text field, so passwords are never
  transcribed or panel-displayed. Second consumer: prefer AX text insertion over robotgo
  keystrokes for the dictation `insert` commit when the focused element accepts value
  writes.
- **`AXPress` actions**: requires the helper to cache element references from the last
  `ax_query` keyed by snapshot, with a short TTL, and a new op invoking an action by
  element index. Solves occluded-target clicking.
- **Menu bar navigation**: walk the menu bar element, invoke items by name; menu items
  also expose their keyboard shortcuts.
- **AXObserver focus/value events**: would break the helper's strict one-request/one-
  response contract (`engine.go` assumes it). Needs a protocol extension for unsolicited
  event lines or a second event-stream helper instance. Explicitly out of scope here.

## Risks

- **TCC attribution surprises** — the whole reason phase 0 exists. A helper that isn't
  trusted returns empty trees with no error unless we surface `ax_trusted`.
- **Huge/hostile trees** (browsers, IDEs): caps + role allowlist + messaging timeout are
  the containment; the engine's kill-respawn is the backstop.
- **Title/visual-text divergence**: AX titles sometimes differ from rendered text
  (localized, truncated, or empty for icon buttons). The ocr fallback covers spoken-text
  misses; icon-only buttons are the companion vision plan's territory.
- **Stale elements between snapshot and click**: v1 clicks at captured coordinates, same
  exposure the OCR path already has; the short TTL narrows it.

## Testing

- Codec round-trip tests for the new op/fields, mirroring `vision/codec_test.go`.
- Provider tests against a fake helper process, mirroring `vision/engine_test.go`.
- An integration test gated the same way as `vision/integration_test.go`, asserting a
  known app (Finder window) yields titled, sanely-bounded elements.
- Manual checklist addition: "click ‹button›" in a native app with screend logs showing
  provider `ax` served it; kill the helper mid-session and confirm fallback to `ocr`.

## Out of scope

Server-side ax+ocr merging, AXPress, observers, menu bar, AX-based dictation insertion
(all listed in phase 4), and any window-management commands (raise/move/resize) — worth a
separate plan once this provider exists.
