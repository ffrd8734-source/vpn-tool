package text

import (
	"strings"
)

// Dedent removes common leading whitespace from all lines in s
func Dedent(s string) string {
	lines := strings.Split(s, "\n")
	minIndent := -1

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent++
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	var sb strings.Builder
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			sb.WriteString("\n")
			continue
		}
		if len(line) >= minIndent {
			sb.WriteString(line[minIndent:])
		} else {
			sb.WriteString(line)
		}
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// Indent prepends prefix to each line in s
func Indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			sb.WriteString(prefix)
			sb.WriteString(line)
		} else {
			sb.WriteString(line)
		}
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
