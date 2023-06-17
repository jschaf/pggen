package texts

import (
	"bytes"
	"math"
	"strings"
	"unicode"
)

// Dedent removes leading whitespace indentation from each line in the text.
//
// Whitespace is removed according to smallest whitespace prefix of a
// determining line.  A determining line is a line that has at least 1 non-space
// character.  The algorithm is:
//
// - If the first line is whitespace, discard it.
// - If the last line is whitespace, discard it.
// - For each remaining line:
//   - If the line only has whitespace, replace it with a single newline.
//   - If the line has non-whitespace chars, find the common whitespace prefix
//     of all such lines.
//
// - Remove the common whitespace prefix from each line.
func Dedent(text string) string {
	indent := math.MaxInt32
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		lineIndent := len(line)
		for i, r := range line {
			if !unicode.IsSpace(r) {
				lineIndent = i
				break
			}
		}
		isBlank := lineIndent == len(line)
		isDetermining := !isBlank
		if isDetermining && lineIndent < indent {
			indent = lineIndent
		}
	}

	start := 1
	end := len(lines) - 1
	// Should we include the first line?
	for _, c := range lines[0] {
		if !unicode.IsSpace(c) {
			start = 0
		}
	}
	// Should we include the last line?
	for _, c := range lines[len(lines)-1] {
		if !unicode.IsSpace(c) {
			end = len(lines)
		}
	}

	if end < start {
		return text
	}

	b := new(bytes.Buffer)
	for i, line := range lines[start:end] {
		lo := 0
		for _, r := range line {
			if unicode.IsSpace(r) {
				lo++
			} else {
				break
			}
		}

		hi := len(line)
		for j := len(line) - 1; j >= 0; j-- {
			if unicode.IsSpace(rune(line[j])) {
				hi--
			} else {
				break
			}
		}

		if lo >= hi {
			b.WriteString("\n")
			continue
		}

		if lo > indent {
			lo = indent
		}

		b.WriteString(line[lo:hi])
		if i < end-start-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
