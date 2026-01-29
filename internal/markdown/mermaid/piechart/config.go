// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package piechart

type config struct {
	// The axial position of the pie slice labels,
	// from 0.0 at the center to 1.0 at the outside edge of the circle.
	textPosition float64
	// showData is a flag to show the data in the pie chart.
	showData bool
	// title is the title of the pie chart.
	title string
}

func newConfig() *config {
	return &config{
		textPosition: defaultTextPosition,
	}
}

const (
	// defaultTextPosition is the default axial position of the pie slice labels.
	defaultTextPosition float64 = 0.75
	// minTextPosition is the minimum axial position of the pie slice labels.
	minTextPosition float64 = 0.0
	// maxTextPosition is the maximum axial position of the pie slice labels.
	maxTextPosition float64 = 1.0
	// noTitle is a constant for no title.
	noTitle string = ""
)

type Option func(*config)

func WithTextPosition(pos float64) Option {
	return func(c *config) {
		if pos < minTextPosition || pos > maxTextPosition {
			pos = defaultTextPosition
		}
		c.textPosition = pos
	}
}

func WithShowData(showData bool) Option {
	return func(c *config) {
		c.showData = showData
	}
}

func WithTitle(title string) Option {
	return func(c *config) {
		c.title = title
	}
}
