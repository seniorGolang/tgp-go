// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package markdown

import "fmt"

func Link(text, url string) string {
	return fmt.Sprintf("[%s](%s)", text, url)
}

func Image(text, url string) string {
	return fmt.Sprintf("![%s](%s)", text, url)
}

func Strikethrough(text string) string {
	return fmt.Sprintf("~~%s~~", text)
}

func Bold(text string) string {
	return fmt.Sprintf("**%s**", text)
}

func Italic(text string) string {
	return fmt.Sprintf("*%s*", text)
}

func BoldItalic(text string) string {
	return fmt.Sprintf("***%s***", text)
}

func Code(text string) string {
	return fmt.Sprintf("`%s`", text)
}

func Highlight(text string) string {
	return fmt.Sprintf("==%s==", text)
}
