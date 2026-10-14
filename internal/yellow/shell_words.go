package yellow

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type shellWord struct {
	value      string
	start, end int
	open       bool
}

// Parse arguments, never a system shell: no variable expansion, pipelines,
// command substitution or execution of pasted newlines.
func shellWords(line string, partial bool) ([]shellWord, error) {
	var words []shellWord
	for pos := 0; pos < len(line); {
		r, size := utf8.DecodeRuneInString(line[pos:])
		if unicode.IsSpace(r) {
			pos += size
			continue
		}
		word := shellWord{start: pos}
		var value strings.Builder
		var quote rune
		for pos < len(line) {
			r, size = utf8.DecodeRuneInString(line[pos:])
			if quote == 0 && unicode.IsSpace(r) {
				break
			}
			if r == '\'' || r == '"' {
				if quote == 0 {
					quote = r
					pos += size
					continue
				}
				if quote == r {
					quote = 0
					pos += size
					continue
				}
			}
			if r == '\\' && quote != '\'' {
				if pos+size == len(line) {
					if !partial {
						return nil, fmt.Errorf("finish the trailing escape")
					}
					pos += size
					break
				}
				next, n := utf8.DecodeRuneInString(line[pos+size:])
				if quote == 0 || next == '"' || next == '\\' {
					value.WriteRune(next)
					pos += size + n
					continue
				}
			}
			value.WriteRune(r)
			pos += size
		}
		if quote != 0 && !partial {
			return nil, fmt.Errorf("close the quoted argument before running the command")
		}
		word.value = value.String()
		word.end = pos
		word.open = quote != 0
		words = append(words, word)
	}
	return words, nil
}
func wordValues(words []shellWord) []string {
	values := make([]string, len(words))
	for i, w := range words {
		values[i] = w.value
	}
	return values
}
func quoteWord(word string) string {
	if word != "" && !strings.ContainsAny(word, " \t\n\r'\"\\") {
		return word
	}
	return "'" + strings.ReplaceAll(word, "'", "'\\''") + "'"
}
