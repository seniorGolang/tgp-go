// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package markdown

import "fmt"

func (m *Markdown) RedBadge(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("![Badge](https://img.shields.io/badge/%s-red)", text))
	return m
}

func (m *Markdown) RedBadgef(format string, args ...any) *Markdown {
	return m.RedBadge(fmt.Sprintf(format, args...))
}

func (m *Markdown) YellowBadge(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("![Badge](https://img.shields.io/badge/%s-yellow)", text))
	return m
}

func (m *Markdown) YellowBadgef(format string, args ...any) *Markdown {
	return m.YellowBadge(fmt.Sprintf(format, args...))
}

func (m *Markdown) GreenBadge(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("![Badge](https://img.shields.io/badge/%s-green)", text))
	return m
}

func (m *Markdown) GreenBadgef(format string, args ...any) *Markdown {
	return m.GreenBadge(fmt.Sprintf(format, args...))
}

func (m *Markdown) BlueBadge(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("![Badge](https://img.shields.io/badge/%s-blue)", text))
	return m
}

func (m *Markdown) BlueBadgef(format string, args ...any) *Markdown {
	return m.BlueBadge(fmt.Sprintf(format, args...))
}
