package util

import (
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	mouseReg               = regexp.MustCompile(`^(zero|one|two|three|four|five|six|seven|eight|nine|center|top left|top right|bottom left|bottom right|\d)$`)
	lowercaseWithSpaceReg  = regexp.MustCompile(`[^a-z\s]`)
	alphanumerReg          = regexp.MustCompile(`[^a-zA-Z0-9]`)
	alphanumerWithSpaceReg = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	numberReg              = regexp.MustCompile(`(one|two|three|four|five|six|seven|eight|nine)$`)
)

func Commas(command string) string {
	return strings.TrimSpace(strings.ReplaceAll(command, ",", " "))
}

func Words(command string) []string {
	split := strings.Split(command, " ")

	for i := range split {
		split[i] = strings.TrimSpace(split[i])
	}
	filteredSplit := split[:0] // Allocate the resulting slice with the same length as the input slice
	for _, word := range split {
		if word != "" {
			filteredSplit = append(filteredSplit, word)
		}
	}
	return filteredSplit
}

func File(command string) string {
	return ToAlphanumeric(Commas(TrimTrailingPunctuation(strings.ToLower(command))))
}

func MultiplierCommand(command string) (string, int) {
	m := 1
	c := CommonMistakes(Commas(TrimTrailingPunctuation(strings.ToLower(command))))
	w := Words(c)
	log.Debugf("Multiplier command: %s, words: %v", command, strings.Join(w, " "))
	if len(w) > 1 {
		last := w[len(w)-1]
		var err error
		m, err = strconv.Atoi(last)
		if err != nil {
			return strings.Join(w, " "), m
		}
		return strings.Join(w[:len(w)-1], " "), m
	}

	return strings.Join(w, " "), m
}


var onlyNumbersReg = regexp.MustCompile(`[^0-9\s]`)
func OnlyNumbers(command string) string {
	return onlyNumbersReg.ReplaceAllString(command, "")
}

var onlySymbolsReg = regexp.MustCompile(`[^\p{P}\p{S}]`)
func OnlySymbols(command string) string {
	return strings.TrimSpace(onlySymbolsReg.ReplaceAllString(command, ""))
}

var noSymbolsReg = regexp.MustCompile(`[\p{P}\p{S}]`)
func NoSymbols(command string) string {
	return strings.TrimSpace(noSymbolsReg.ReplaceAllString(command, ""))
}
var noSymbolsExceptPunctuationReg = regexp.MustCompile(`[\p{S}]`)
func NoSymbolsExceptPunctuation(command string) string {
	return strings.TrimSpace(noSymbolsExceptPunctuationReg.ReplaceAllString(command, ""))
}

func ToAlphanumeric(command string) string {
	return alphanumerReg.ReplaceAllString(command, "")
}

func ToAlphanumericWithSpace(command string) string {
	return alphanumerWithSpaceReg.ReplaceAllString(command, "")
}

func ToWords(command string) string {
	stripped := lowercaseWithSpaceReg.ReplaceAllString(numberReg.ReplaceAllString(strings.ToLower(command), ""), "")
	return strings.TrimSpace(stripped)
}

func TrimTrailingPunctuation(s string) string {
	reg := regexp.MustCompile(`[\p{P}\p{S}]+$`)
	return reg.ReplaceAllString(s, "")
}

func ReplaceNumberWords(command string) string {
	command = strings.ReplaceAll(command, "zero", "0")
	command = strings.ReplaceAll(command, "one", "1")
	command = strings.ReplaceAll(command, "two", "2")
	command = strings.ReplaceAll(command, "three", "3")
	command = strings.ReplaceAll(command, "four", "4")
	command = strings.ReplaceAll(command, "five", "5")
	command = strings.ReplaceAll(command, "six", "6")
	command = strings.ReplaceAll(command, "seven", "7")
	command = strings.ReplaceAll(command, "eight", "8")
	command = strings.ReplaceAll(command, "nine", "9")
	return command
}

func CommonMistakes(s string) string {
	before := s
	// Only replace "under" when it's the complete word (not "underscore", etc.)
	s = regexp.MustCompile("^under$").ReplaceAllString(s, "undo")

	s = regexp.MustCompile("one$").ReplaceAllString(s, "1")
	s = regexp.MustCompile("two$").ReplaceAllString(s, "2")
	s = regexp.MustCompile("three$").ReplaceAllString(s, "3")
	s = regexp.MustCompile("four$").ReplaceAllString(s, "4")
	s = regexp.MustCompile("five$").ReplaceAllString(s, "5")
	s = regexp.MustCompile("six$").ReplaceAllString(s, "6")
	s = regexp.MustCompile("seven$").ReplaceAllString(s, "7")
	s = regexp.MustCompile("eight$").ReplaceAllString(s, "8")
	s = regexp.MustCompile("nine$").ReplaceAllString(s, "9")
	s = regexp.MustCompile("ten$").ReplaceAllString(s, "10")

	s = regexp.MustCompile("^sente\\s?").ReplaceAllString(s, "center ")
	s = regexp.MustCompile("^doc$").ReplaceAllString(s, "dock")

	s = regexp.MustCompile(`\bfor$`).ReplaceAllString(s, "4")
	s = regexp.MustCompile(`\btoo$`).ReplaceAllString(s, "2")
	s = regexp.MustCompile(`\bto$`).ReplaceAllString(s, "2")

	s = regexp.MustCompile(`\binto\b`).ReplaceAllString(s, "enter")
	s = regexp.MustCompile(`\band\b`).ReplaceAllString(s, "end")
	s = regexp.MustCompile(`\blift\b`).ReplaceAllString(s, "left")
	s = regexp.MustCompile(`\bwrite\b`).ReplaceAllString(s, "right")
	s = regexp.MustCompile(`go land$`).ReplaceAllString(s, "goland")

	if s != before {
		log.Infof("Corrected common mistakes from: %v, to: %v\n", before, s)
	}
	return s
}

var removeBlockHeaderFooter = regexp.MustCompile("(?m)^```[a-z]*\n|\n```$")

func RemoveMarkdownCodeBlock(input string) string {
	output := removeBlockHeaderFooter.ReplaceAllString(input, "")
	output = strings.TrimSpace(output)
	return output
}

var removeCodeQuotes = regexp.MustCompile("(?m)^`|`$")

func RemoveCodeQuotesBlock(input string) string {
	output := removeCodeQuotes.ReplaceAllString(input, "")
	output = strings.TrimSpace(output)
	return output
}

func StripAIFormating(input string) string {
	return RemoveCodeQuotesBlock(RemoveMarkdownCodeBlock(input))
}

// ExtractTagged returns the trimmed content of the first <tag>...</tag>
// block. Prompting a model to wrap its answer in tags and extracting only
// that is the reliable way to drop preambles/commentary from small models.
func ExtractTagged(s, tag string) (string, bool) {
	re := regexp.MustCompile(`(?is)<` + regexp.QuoteMeta(tag) + `>(.*?)</` + regexp.QuoteMeta(tag) + `>`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	out := strings.TrimSpace(m[1])
	return out, out != ""
}

var spokenDigits = map[string]int{
	"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4,
	"five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9,
	"ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14,
	"fifteen": 15, "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19,
}

var spokenTens = map[string]int{
	"twenty": 20, "thirty": 30, "forty": 40, "fifty": 50,
	"sixty": 60, "seventy": 70, "eighty": 80, "ninety": 90,
}

// ParseSpokenNumber converts numeric or spoken text ("50", "fifty",
// "fifty three", "twenty-one") into an integer. The whole input must be the
// number — any other word fails the parse.
func ParseSpokenNumber(s string) (int, bool) {
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == ' ':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r == '-':
			return ' '
		}
		return -1
	}, s)

	words := strings.Fields(clean)
	if len(words) == 0 {
		return 0, false
	}
	total, matched := 0, false
	for _, w := range words {
		if n, err := strconv.Atoi(w); err == nil {
			total += n
			matched = true
			continue
		}
		if n, ok := spokenTens[w]; ok {
			total += n
			matched = true
			continue
		}
		if n, ok := spokenDigits[w]; ok {
			total += n
			matched = true
			continue
		}
		return 0, false
	}
	return total, matched
}
