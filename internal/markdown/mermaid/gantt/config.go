// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package gantt

type config struct {
	// title is the title of the Gantt chart.
	title string
	// dateFormat is the date format for the Gantt chart.
	dateFormat string
	// axisFormat is the axis format for the Gantt chart.
	axisFormat string
	// tickInterval is the tick interval for the Gantt chart.
	tickInterval string
	// excludes specifies days to exclude (e.g., "weekends", "2024-01-01").
	excludes []string
	// todayMarker specifies the today marker style (e.g., "off" or CSS style).
	todayMarker string
}

func newConfig() *config {
	return &config{
		title:      noTitle,
		dateFormat: "", // Mermaid default: YYYY-MM-DD
		axisFormat: "",
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

func WithDateFormat(format string) Option {
	return func(c *config) {
		c.dateFormat = format
	}
}

func WithAxisFormat(format string) Option {
	return func(c *config) {
		c.axisFormat = format
	}
}

func WithTickInterval(interval string) Option {
	return func(c *config) {
		c.tickInterval = interval
	}
}

func WithExcludes(excludes ...string) Option {
	return func(c *config) {
		c.excludes = excludes
	}
}

func WithTodayMarker(marker string) Option {
	return func(c *config) {
		c.todayMarker = marker
	}
}
