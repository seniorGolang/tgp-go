// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package flowchart

const (
	// noTitle is a constant for no title.
	noTitle string = ""
)

type config struct {
	// title is the title of the flowchart.
	title string
	// oriental is the oriental of the flowchart.
	// Default is top to bottom.
	oriental oriental
}

func newConfig() *config {
	return &config{
		oriental: tb,
	}
}

type Option func(*config)

func WithTitle(title string) Option {
	return func(c *config) {
		c.title = title
	}
}

func WithOrientalTopToBottom() Option {
	return func(c *config) {
		c.oriental = tb
	}
}

func WithOrientalTopDown() Option {
	return func(c *config) {
		c.oriental = td
	}
}

func WithOrientalBottomToTop() Option {
	return func(c *config) {
		c.oriental = bt
	}
}

func WithOrientalRightToLeft() Option {
	return func(c *config) {
		c.oriental = rl
	}
}

func WithOrientalLeftToRight() Option {
	return func(c *config) {
		c.oriental = lr
	}
}
