// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package gantt

type config struct {
	title        string
	excludes     []string
	axisFormat   string
	dateFormat   string
	tickInterval string
	todayMarker  string
}

func newConfig() (c *config) {
	return &config{
		title:      noTitle,
		dateFormat: "", // Mermaid default: YYYY-MM-DD
		axisFormat: "",
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

func WithDateFormat(format string) (o Option) {
	return func(c *config) {
		c.dateFormat = format
	}
}

func WithAxisFormat(format string) (o Option) {
	return func(c *config) {
		c.axisFormat = format
	}
}

func WithTickInterval(interval string) (o Option) {
	return func(c *config) {
		c.tickInterval = interval
	}
}

func WithExcludes(excludes ...string) (o Option) {
	return func(c *config) {
		c.excludes = excludes
	}
}

func WithTodayMarker(marker string) (o Option) {
	return func(c *config) {
		c.todayMarker = marker
	}
}
