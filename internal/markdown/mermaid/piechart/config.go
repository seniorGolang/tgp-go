// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package piechart

type config struct {
	title        string
	showData     bool
	textPosition float64
}

func newConfig() (c *config) {
	return &config{
		textPosition: defaultTextPosition,
	}
}

const (
	noTitle             string  = ""
	minTextPosition     float64 = 0.0
	maxTextPosition     float64 = 1.0
	defaultTextPosition float64 = 0.75
)

type Option func(*config)

func WithTextPosition(pos float64) (o Option) {
	return func(c *config) {
		if pos < minTextPosition || pos > maxTextPosition {
			pos = defaultTextPosition
		}
		c.textPosition = pos
	}
}

func WithShowData(showData bool) (o Option) {
	return func(c *config) {
		c.showData = showData
	}
}

func WithTitle(title string) (o Option) {
	return func(c *config) {
		c.title = title
	}
}
