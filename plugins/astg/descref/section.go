// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package descref

import "strings"

func extractSectionFromMarkdown(content string, heading string) (s string) {

	lines := strings.Split(content, "\n")
	want := normalizeHeadingText(heading)
	var end int
	var level int
	var start int
	for i, line := range lines {
		l, t := parseATXHeading(line)
		if l == 0 {
			continue
		}
		if normalizeHeadingText(t) != want {
			continue
		}
		level = l
		start = i + 1
		break
	}
	if start == 0 {
		return ""
	}

	end = len(lines)
	for i := start; i < len(lines); i++ {
		l, _ := parseATXHeading(lines[i])
		if l > 0 && l <= level {
			end = i
			break
		}
		end = i + 1
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func parseATXHeading(line string) (level int, text string) {

	line = strings.TrimSpace(line)
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) {
		return 0, ""
	}
	if line[level] != ' ' && line[level] != '\t' {
		return 0, ""
	}
	text = strings.TrimSpace(line[level:])
	return level, text
}

func normalizeHeadingText(s string) (out string) {

	s = strings.TrimSpace(s)
	var b strings.Builder
	prev := ' '
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if prev != ' ' {
				b.WriteRune(' ')
			}
			prev = ' '
		} else {
			b.WriteRune(r)
			prev = r
		}
	}
	return strings.TrimSpace(b.String())
}
