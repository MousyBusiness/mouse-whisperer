---
name: write-command
description: Add a new voice command to the mouse-whisperer config. Use whenever the user asks to add, create, write, or wire up a voice command, trigger, hotkey, or shortcut in this project — phrases like "add a command for X", "make X a voice trigger", "wire up Y to a phrase". Encodes ASR-friendly trigger phrasing, conflict avoidance, the TOML action schema, and how to validate before committing.
tools: Read, Edit, Write, Bash, Grep, Glob
---

# Write a mouse-whisperer voice command

Voice commands live in `config/*.toml`. Each file groups commands by category; the loader auto-merges every `.toml` in the dir (files prefixed `_` are skipped — see `internal/config/config.go`).

A command looks like:

```toml
[commands.<category>.<name>]
triggers = ["spoken phrase", "alias phrase"]
description = "Shown in the UI"
actions = [
  { method = "...", args = { ... }, sleep = 100 },
]
```

## Workflow

1. **Decide trigger phrasing** — see the rules below; this is where commands usually fail.
2. **Pick the file** — extend an existing category if it fits (`coding`, `desktop`, `claude`, `git`, etc.); otherwise create `config/<category>.toml` and add `<category>` to `category_order` in `config/settings.toml`.
3. **Write the entry** — choose methods/args from the action reference below.
4. **Validate** — parse the TOML (see the validate step below). Don't commit a config that won't load.
5. **Report** to the user the triggers added and which file changed.

## Trigger phrasing (the most-broken rule)

These rules are derived from real failures in this project. Treat them as hard requirements unless the user overrides.

- **Real English words only.** The voice transcriber will not produce abbreviations, acronyms, or invented spellings. `"wip"` → use `"whip"`. `"enhan"` → use `"enhancement"`. `"ctx"` → use `"context"`. If the user dictates an abbreviation, swap to a real word that sounds like it; keep the typed output as the abbreviation.
- **Watch out for words the transcriber renders inconsistently.** Brand names and homophones often arrive as something else: `"claude"` comes back as `"clawed"` or `"cloud"` — use `"agent"` instead. When picking a leading word for a new namespace, prefer plain English over product names.
- **Spell digit suffixes as digits, not words, in triggers** (`"page 1"` not `"page one"`). `pkg/util/strip.go` `CommonMistakes` rewrites trailing number words to digits before lookup, so word-spelled triggers won't match the post-normalization input.
- **Avoid prefix/substring conflicts with existing triggers.** A phrase like `"tab one"` collides with the existing `"tab"` trigger because the matcher prefers the short exact hit. Before picking words, `grep 'triggers = \[' config/*.toml` and check for overlap on the leading word.
- **Two-word triggers beat one-word triggers** for anything new. They cut false positives and leave single words free for hot, single-purpose actions.
- **Offer synonyms when the user can't predict which word they'll say.** Example pattern in `config/desktop.toml`: tab-switch commands accept `page N` / `swap N` / `flip N`. Add aliases in the same `triggers = [...]` array.
- **For number-suffixed commands, spell the digit** (`"page one"` not `"page 1"`). The transcriber emits words, not glyphs.
- **Don't reuse a word from another command's trigger** even with a different suffix unless you're certain fuzzy match won't grab it. When in doubt, try a different leading word.

## Action reference

The `actions` array executes in order. Each action is `{ method, args, sleep }`. `sleep` is a post-action delay in ms.

Common methods (defined in `internal/impl/`):

| Method | Purpose | Args |
|---|---|---|
| `type_str` | Type a literal string | `text = "..."` |
| `type_num` | Type a number captured from speech | (no args; or `number = N` to inject) |
| `robotgo` | Send a keystroke via robotgo (lowercase keys) | `key = "tab"`, `modifiers = ["cmd","shift"]` |
| `keybd_event` | Send a keystroke via the system keybd_event API (VK_* codes) | `key = "VK_L"`, `modifiers = ["cmd"]` |
| `ai_ollama` | Local-LLM-generated text into overlay | see `config/ai.toml` |
| `ai_clip` / `ai_accept` / `ai_reject` | Manage AI overlay output | (none) |

Modifier strings: `"cmd"`, `"shift"`, `"alt"`, `"ctrl"`.

Optional top-level command fields:
- `disable_fuzzy = true` — require exact match (use when the trigger is short and prone to grabbing related phrases).
- `multiplier = true` — allow "<N> times" repetition (see `text.enter`).

### Patterns that come up a lot

**Type a slash-command and submit** (Claude Code, REPLs):
```toml
actions = [
  { method = "type_str", args = { text = "/clear" } },
  { method = "robotgo", args = { key = "enter" }, sleep = 100 },
]
```
The sleep gives the target app a beat to register the typed text before Enter.

**Modifier + key**:
```toml
actions = [{ method = "robotgo", args = { key = "1", modifiers = ["cmd"] } }]
```

**Per-digit commands (Cmd+1..9, etc.)**: write nine explicit entries. `type_num` types into a text field — it can't combine with modifiers. See `config/desktop.toml` `page_one`..`page_nine`.

**Chord with a brief hold** (e.g. double-shift): use two robotgo calls with `sleep` between, as in `config/ide.toml` `files`.

## Validate before reporting done

After editing, parse the changed file(s). The project depends on `github.com/BurntSushi/toml`, which is already in `go.sum`:

```bash
cat > /tmp/toml_check.go <<'EOF'
//go:build ignore
package main
import ("fmt"; "os"; "github.com/BurntSushi/toml")
func main() {
  var v map[string]any
  if _, err := toml.DecodeFile(os.Args[1], &v); err != nil { fmt.Println("ERROR:", err); os.Exit(1) }
  fmt.Println("OK:", os.Args[1])
}
EOF
cp /tmp/toml_check.go ./toml_check.go
go run toml_check.go config/<file>.toml
rm toml_check.go
```

(`go build ./...` also catches it but pulls in a heavy C dep — gosseract — that doesn't compile cleanly on every dev machine. The targeted parse above is enough.)

## New-category checklist

If you created `config/<new>.toml`:
1. The file starts with a top-level `[commands.<new>]` header before the first sub-table.
2. Add `"<new>"` to `category_order` in `config/settings.toml` (otherwise the UI hides it).
3. Validate `settings.toml` too — it's easy to break the array.

## What "done" looks like

A short message to the user listing:
- the trigger phrases added (each one, including aliases),
- which file they went into,
- any settings.toml change.

No summary of the diff itself — the user can read it.
