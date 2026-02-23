package main

import (
	"strings"
)

// wrap breaks s into lines of at most width runes, at word boundaries where possible.
func wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		runes := []rune(strings.TrimRight(line, " \t"))
		for len(runes) > 0 {
			if len(runes) <= width {
				out.WriteString(string(runes))
				runes = nil
				continue
			}
			// find last space in first width runes
			lastSpace := -1
			for j := 0; j < width && j < len(runes); j++ {
				if runes[j] == ' ' || runes[j] == '\t' {
					lastSpace = j
				}
			}
			cut := width
			if lastSpace > 0 {
				cut = lastSpace + 1
			}
			out.WriteString(string(runes[:cut]))
			out.WriteByte('\n')
			runes = runes[cut:]
			// trim leading spaces from next segment
			for len(runes) > 0 && (runes[0] == ' ' || runes[0] == '\t') {
				runes = runes[1:]
			}
		}
	}
	return out.String()
}
