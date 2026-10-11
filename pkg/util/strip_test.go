package util

import "testing"

func TestRemoveMarkdownCodeBlock(t *testing.T) {
	result := RemoveMarkdownCodeBlock("```bash\nstub\n```")
	if result != "stub" {
		t.Errorf("result should be stub, got: %s", result)
	}
}
func TestRemoveCodeQuotesBlock(t *testing.T) {
	result := RemoveCodeQuotesBlock("`bash\nstub\n`")
	if result != "bash\nstub" {
		t.Errorf("result should be bash\nstub, got: %s", result)
	}
}

func TestExtractTagged(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"clean", "<output>Hello world.</output>", "Hello world.", true},
		{"preamble and trailer dropped", "Sure! Here's the corrected text:\n<output>Hello world.</output>\nLet me know if you need more help!", "Hello world.", true},
		{"multiline content", "<output>Line one.\nLine two.</output>", "Line one.\nLine two.", true},
		{"case insensitive tags", "<OUTPUT>Hello.</OUTPUT>", "Hello.", true},
		{"no tags", "Hello world.", "", false},
		{"empty tags", "<output>  </output>", "", false},
		{"first block wins", "<output>first</output> junk <output>second</output>", "first", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ExtractTagged(c.in, "output")
			if got != c.want || ok != c.ok {
				t.Fatalf("got (%q, %v), want (%q, %v)", got, ok, c.want, c.ok)
			}
		})
	}
}

func TestParseSpokenNumber(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"50", 50, true},
		{"7", 7, true},
		{"fifty", 50, true},
		{"Fifty.", 50, true},
		{"fifty three", 53, true},
		{"twenty-one", 21, true},
		{"seventeen", 17, true},
		{"ten", 10, true},
		{"zero", 0, true},
		{"handler", 0, false},
		{"fifty handler", 0, false},
		{"", 0, false},
		{"  42  ", 42, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := ParseSpokenNumber(c.in)
			if got != c.want || ok != c.ok {
				t.Fatalf("ParseSpokenNumber(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
			}
		})
	}
}
