// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package quadrant

type config struct {
	title string
}

func newConfig() (c *config) {
	return &config{
		title: noTitle,
	}
}

const (
	noTitle string = ""
)

type Option func(*config)

func WithTitle(title string) (o Option) {
	return func(c *config) {
		c.title = title
	}
}
