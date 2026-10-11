package symbols

import "strings"

// SplitTriggerAndSymbol separates a trigger word from any attached symbols.
// e.g., "put_" -> ("put", "_"), "put," -> ("put", ","), "scan." -> ("scan", ".")
func SplitTriggerAndSymbol(s string) (word, symbol string) {
	if len(s) == 0 {
		return "", ""
	}

	// Find where the word ends (last alphanumeric character)
	lastAlphaNum := -1
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			lastAlphaNum = i
		}
	}

	if lastAlphaNum == -1 {
		// No alphanumeric characters, entire string is symbols
		return "", s
	}

	// Split at the boundary
	word = s[:lastAlphaNum+1]
	symbol = s[lastAlphaNum+1:]
	return word, symbol
}


// symbolToWord maps literal symbols to their word names for normalization
// This handles cases where transcription captures the symbol instead of the word
var SymbolToWord = map[string]string{
	"!":  "exclamation",
	"@":  "at",
	"#":  "hash",
	"$":  "dollar",
	"%":  "percent",
	"^":  "carat up",
	"&":  "and",
	"*":  "asterisks",
	"|":  "pipe",
	"\\": "backslash",
	"/":  "slash",
	"\"": "quote",
	"'":  "single quote",
	"`":  "backtick",
	"~":  "tilde",
	"_":  "underscore",
	":":  "colon",
	"<":  "carat left",
	">":  "carat right",
	"=":  "equals",
	".":  "dot",
	",":  "comma",
	";":  "semicolon",
	"?":  "question",
	"(":  "bracket",
	")":  "right bracket",
	"[":  "square bracket",
	"]":  "right square bracket",
	"{":  "curly bracket",
	"}":  "right curly bracket",
	"-":  "dash",
	"+":  "plus",
}

// NormalizeSymbolInput converts literal symbols in transcription to word names
// Handles cases like "," -> "comma", "put," -> "comma", etc.
func NormalizeSymbolInput(input string) string {
	input = strings.TrimSpace(input)

	// If the input is just a single symbol, convert it
	if word, ok := SymbolToWord[input]; ok {
		return word
	}

	// Check if input ends with a symbol (e.g., "put," or "comma,")
	// Strip trailing punctuation that was likely transcribed literally
	for sym, word := range SymbolToWord {
		// If input ends with the symbol, check if removing it leaves empty or "put"
		if strings.HasSuffix(input, sym) {
			prefix := strings.TrimSuffix(input, sym)
			prefix = strings.TrimSpace(prefix)
			// If nothing left or just spaces, the symbol IS the command
			if prefix == "" {
				return word
			}
			// If prefix is a valid word already, keep it and ignore trailing symbol
			// This handles "comma," -> "comma"
			if _, isWord := WordToSymbol[prefix]; isWord {
				return prefix
			}
		}
	}

	// Check if input starts with a symbol (e.g., ",put" -> "comma")
	for sym, word := range SymbolToWord {
		if strings.HasPrefix(input, sym) {
			suffix := strings.TrimPrefix(input, sym)
			suffix = strings.TrimSpace(suffix)
			if suffix == "" {
				return word
			}
		}
	}

	return input
}

// wordToSymbol maps word names to symbols for insertSymbol
var WordToSymbol = map[string]string{
	"exclamation":          "!",
	"at":                   "@",
	"hash":                 "#",
	"dollar":               "$",
	"percent":              "%",
	"carat up":             "^",
	"and":                  "&",
	"ampersand":            "&",
	"asterisks":            "*",
	"asterisk":             "*",
	"star":                 "*",
	"pipe":                 "|",
	"backslash":            "\\",
	"slash":                "/",
	"forward slash":        "/",
	"quote":                "\"",
	"double quote":         "\"",
	"single quote":         "'",
	"apostrophe":           "'",
	"backtick":             "`",
	"tick":                 "`",
	"grave":                "`",
	"tilde":                "~",
	"underscore":           "_",
	"under score":          "_",
	"colon":                ":",
	"carat left":           "<",
	"less than":            "<",
	"carat right":          ">",
	"greater than":         ">",
	"not equals":           "!=",
	"not equal":            "!=",
	"equals":               "=",
	"equal":                "=",
	"exactly equals":       "==",
	"double equals":        "==",
	"triple equals":        "===",
	"walrus":               ":=",
	"arrow":                "->",
	"fat arrow":            "=>",
	"dot":                  ".",
	"period":               ".",
	"full stop":            ".",
	"comma":                ",",
	"semicolon":            ";",
	"semi colon":           ";",
	"simicolon":            ";", // common typo
	"question mark":        "?",
	"question":             "?",
	"bracket":              "(",
	"left bracket":         "(",
	"open bracket":         "(",
	"paren":                "(",
	"left paren":           "(",
	"open paren":           "(",
	"right bracket":        ")",
	"close bracket":        ")",
	"right paren":          ")",
	"close paren":          ")",
	"square bracket":       "[",
	"left square bracket":  "[",
	"open square bracket":  "[",
	"left square":          "[",
	"right square bracket": "]",
	"close square bracket": "]",
	"right square":         "]",
	"curly bracket":        "{",
	"left curly bracket":   "{",
	"open curly bracket":   "{",
	"left curly":           "{",
	"open curly":           "{",
	"brace":                "{",
	"left brace":           "{",
	"open brace":           "{",
	"right curly bracket":  "}",
	"close curly bracket":  "}",
	"right curly":          "}",
	"close curly":          "}",
	"right brace":          "}",
	"close brace":          "}",
	"dash":                 "-",
	"hyphen":               "-",
	"minus":                "-",
	"plus":                 "+",
	"newline":              "\n",
	"new line":             "\n",
}
