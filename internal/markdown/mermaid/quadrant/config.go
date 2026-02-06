// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package quadrant

type config struct {
	// title is the title of the quadrant chart.
	title string
}

func newConfig() *config {
	return &config{
		title: noTitle,
	}
}

const (
	// noTitle is a constant for no title.
	noTitle string = ""
)

type Option func(*config)

func WithTitle(title string) Option {
	return func(c *config) {
		c.title = title
	}
}
