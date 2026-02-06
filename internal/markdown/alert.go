// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package markdown

import "fmt"

func (m *Markdown) Note(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("> [!NOTE]  \n> %s", text))
	return m
}

func (m *Markdown) Notef(format string, args ...any) *Markdown {
	return m.Note(fmt.Sprintf(format, args...))
}

func (m *Markdown) Tip(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("> [!TIP]  \n> %s", text))
	return m
}

func (m *Markdown) Tipf(format string, args ...any) *Markdown {
	return m.Tip(fmt.Sprintf(format, args...))
}

func (m *Markdown) Important(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("> [!IMPORTANT]  \n> %s", text))
	return m
}

func (m *Markdown) Importantf(format string, args ...any) *Markdown {
	return m.Important(fmt.Sprintf(format, args...))
}

func (m *Markdown) Warning(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("> [!WARNING]  \n> %s", text))
	return m
}

func (m *Markdown) Warningf(format string, args ...any) *Markdown {
	return m.Warning(fmt.Sprintf(format, args...))
}

func (m *Markdown) Caution(text string) *Markdown {
	m.body = append(m.body, fmt.Sprintf("> [!CAUTION]  \n> %s", text))
	return m
}

func (m *Markdown) Cautionf(format string, args ...any) *Markdown {
	return m.Caution(fmt.Sprintf(format, args...))
}
