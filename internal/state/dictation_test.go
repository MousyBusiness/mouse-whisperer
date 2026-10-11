package state

import (
	"strings"
	"testing"
)

func TestDictationState(t *testing.T) {
	t.Run("append joins with single space, keeps punctuation and case", func(t *testing.T) {
		d := &DictationState{}
		d.Append("The quick brown fox,")
		d.Append("jumps over the lazy dog.")
		if got, want := d.Text(), "The quick brown fox, jumps over the lazy dog."; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("append trims and drops empty segments", func(t *testing.T) {
		d := &DictationState{}
		d.Append("  hello  ")
		d.Append("   ")
		d.Append("")
		if got := d.Text(); got != "hello" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("pop last removes most recent utterance", func(t *testing.T) {
		d := &DictationState{}
		d.Append("keep this")
		d.Append("remove this")
		last, ok := d.PopLast()
		if !ok || last != "remove this" {
			t.Fatalf("got %q ok=%v", last, ok)
		}
		if got := d.Text(); got != "keep this" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("pop last on empty buffer", func(t *testing.T) {
		d := &DictationState{}
		if _, ok := d.PopLast(); ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("take all returns and clears atomically", func(t *testing.T) {
		d := &DictationState{}
		d.Append("one")
		d.Append("two")
		if got := d.TakeAll(); got != "one two" {
			t.Fatalf("got %q", got)
		}
		if d.HasText() {
			t.Fatal("buffer should be empty after TakeAll")
		}
	})

	t.Run("pop words within a segment", func(t *testing.T) {
		d := &DictationState{}
		d.Append("the quick brown fox")
		if !d.PopWords(2) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "the quick" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("pop words crosses segment boundaries", func(t *testing.T) {
		d := &DictationState{}
		d.Append("first bit")
		d.Append("second")
		if !d.PopWords(2) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "first" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("over-popping empties the buffer", func(t *testing.T) {
		d := &DictationState{}
		d.Append("one two")
		if !d.PopWords(10) {
			t.Fatal("expected ok")
		}
		if d.HasText() {
			t.Fatal("expected empty buffer")
		}
		if d.PopWords(1) {
			t.Fatal("expected ok=false on empty buffer")
		}
	})

	t.Run("pop chars within a segment", func(t *testing.T) {
		d := &DictationState{}
		d.Append("hello world")
		if !d.PopChars(3) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "hello wo" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("pop chars counts the join space between segments", func(t *testing.T) {
		d := &DictationState{}
		d.Append("ab")
		d.Append("cd")
		// visual text "ab cd": deleting 3 removes "d", "c" and the space
		if !d.PopChars(3) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "ab" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("pop chars is rune-safe", func(t *testing.T) {
		d := &DictationState{}
		d.Append("héllo")
		if !d.PopChars(4) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "h" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("over-popping chars empties the buffer", func(t *testing.T) {
		d := &DictationState{}
		d.Append("hi")
		if !d.PopChars(10) {
			t.Fatal("expected ok")
		}
		if d.HasText() {
			t.Fatal("expected empty buffer")
		}
		if d.PopChars(1) {
			t.Fatal("expected ok=false on empty buffer")
		}
	})

	t.Run("set leading case both directions", func(t *testing.T) {
		d := &DictationState{}
		d.Append("the quick brown fox")
		d.Append("jumps over")
		if !d.SetLeadingCase(true) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "The quick brown fox jumps over" {
			t.Fatalf("got %q", got)
		}
		if !d.SetLeadingCase(false) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "the quick brown fox jumps over" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("set leading case on empty buffer", func(t *testing.T) {
		d := &DictationState{}
		if d.SetLeadingCase(true) {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("set leading case ignores non-letters", func(t *testing.T) {
		d := &DictationState{}
		d.Append("42 things")
		if !d.SetLeadingCase(true) {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "42 things" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("set trailing punct replaces the terminator", func(t *testing.T) {
		d := &DictationState{}
		d.Append("the meeting starts at nine")
		d.Append("is that too early.")
		if !d.SetTrailingPunct("?") {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "the meeting starts at nine is that too early?" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("set trailing punct appends when unpunctuated", func(t *testing.T) {
		d := &DictationState{}
		d.Append("is that too early")
		if !d.SetTrailingPunct("?") {
			t.Fatal("expected ok")
		}
		if got := d.Text(); got != "is that too early?" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("set trailing punct on empty buffer", func(t *testing.T) {
		d := &DictationState{}
		if d.SetTrailingPunct("?") {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("undo reverts set trailing punct", func(t *testing.T) {
		d := &DictationState{}
		d.Append("is that too early.")
		if !d.SetTrailingPunct("?") {
			t.Fatal("expected ok")
		}
		if !d.Undo() {
			t.Fatal("expected undo ok")
		}
		if got := d.Text(); got != "is that too early." {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("replace all swaps buffer for one segment", func(t *testing.T) {
		d := &DictationState{}
		d.Append("messy. text,with. breaks")
		d.Append("more. bits")
		d.ReplaceAll("Clean, cohesive text.")
		if got := d.Text(); got != "Clean, cohesive text." {
			t.Fatalf("got %q", got)
		}
		if _, ok := d.PopLast(); !ok {
			t.Fatal("expected one segment")
		}
		if d.HasText() {
			t.Fatal("expected empty after popping the single segment")
		}
	})

	t.Run("undo reverts an LLM replacement", func(t *testing.T) {
		d := &DictationState{}
		d.Append("the original dictated text")
		d.ReplaceAll("A mangled rewrite.")
		if !d.Undo() {
			t.Fatal("expected undo")
		}
		if got := d.Text(); got != "the original dictated text" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("undo redo cycle", func(t *testing.T) {
		d := &DictationState{}
		d.Append("one")
		d.Append("two")
		d.PopWords(1)
		if got := d.Text(); got != "one" {
			t.Fatalf("setup got %q", got)
		}
		if !d.Undo() {
			t.Fatal("expected undo")
		}
		if got := d.Text(); got != "one two" {
			t.Fatalf("after undo got %q", got)
		}
		if !d.Redo() {
			t.Fatal("expected redo")
		}
		if got := d.Text(); got != "one" {
			t.Fatalf("after redo got %q", got)
		}
	})

	t.Run("undo past first append empties buffer", func(t *testing.T) {
		d := &DictationState{}
		d.Append("only utterance")
		if !d.Undo() {
			t.Fatal("expected undo")
		}
		if d.HasText() {
			t.Fatal("expected empty buffer")
		}
		if d.Undo() {
			t.Fatal("nothing left to undo")
		}
	})

	t.Run("new mutation clears redo", func(t *testing.T) {
		d := &DictationState{}
		d.Append("one")
		d.Append("two")
		d.Undo()
		d.Append("three")
		if d.Redo() {
			t.Fatal("redo must be invalidated by a new mutation")
		}
		if got := d.Text(); got != "one three" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("commit and discard clear history", func(t *testing.T) {
		d := &DictationState{}
		d.Append("one")
		d.TakeAll()
		if d.Undo() {
			t.Fatal("history must not survive a commit")
		}
		d.Append("two")
		d.Clear()
		if d.Undo() {
			t.Fatal("history must not survive a discard")
		}
	})

	t.Run("has text and clear", func(t *testing.T) {
		d := &DictationState{}
		if d.HasText() {
			t.Fatal("new buffer should be empty")
		}
		d.Append("something")
		if !d.HasText() {
			t.Fatal("expected text")
		}
		d.Clear()
		if d.HasText() {
			t.Fatal("expected empty after Clear")
		}
	})

	t.Run("replay runs group recorded and unrecorded segments", func(t *testing.T) {
		d := &DictationState{}
		d.AppendClip("with the final", "a.wav")
		d.AppendClip("result being", "b.wav")
		d.Append("typed")
		d.AppendClip("a summary", "c.wav")
		runs := d.ReplayRuns()
		want := []ReplayRun{
			{Text: "with the final result being", Clips: []string{"a.wav", "b.wav"}},
			{Text: "typed"},
			{Text: "a summary", Clips: []string{"c.wav"}},
		}
		if len(runs) != len(want) {
			t.Fatalf("got %+v", runs)
		}
		for i := range want {
			if runs[i].Text != want[i].Text || strings.Join(runs[i].Clips, ",") != strings.Join(want[i].Clips, ",") {
				t.Fatalf("run %d got %+v, want %+v", i, runs[i], want[i])
			}
		}
	})

	t.Run("scratch drops the recording with the text", func(t *testing.T) {
		d := &DictationState{}
		d.AppendClip("keep", "a.wav")
		d.AppendClip("drop", "b.wav")
		d.PopLast()
		runs := d.ReplayRuns()
		if len(runs) != 1 || strings.Join(runs[0].Clips, ",") != "a.wav" {
			t.Fatalf("got %+v", runs)
		}
	})

	t.Run("replace all keeps every recording", func(t *testing.T) {
		d := &DictationState{}
		d.AppendClip("messy.", "a.wav")
		d.AppendClip("text", "b.wav")
		d.ReplaceAll("Tidy text.")
		runs := d.ReplayRuns()
		if len(runs) != 1 || runs[0].Text != "Tidy text." || strings.Join(runs[0].Clips, ",") != "a.wav,b.wav" {
			t.Fatalf("got %+v", runs)
		}
	})

	t.Run("replace runs is undoable and drops empty runs", func(t *testing.T) {
		d := &DictationState{}
		d.AppendClip("with the final", "a.wav")
		d.Append("typed")
		runs := d.ReplayRuns()
		runs[0].Text = "With the final result,"
		runs[1].Text = " "
		d.ReplaceRuns(runs)
		if got := d.Text(); got != "With the final result," {
			t.Fatalf("got %q", got)
		}
		if !d.Undo() || d.Text() != "with the final typed" {
			t.Fatalf("undo got %q", d.Text())
		}
	})
}
