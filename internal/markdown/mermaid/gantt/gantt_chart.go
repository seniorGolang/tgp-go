// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

// Package gantt is a mermaid Gantt chart builder.
package gantt

import (
	"fmt"
	"io"
	"strings"

	"tgp/internal/markdown/internal"
)

type Chart struct {
	// body is Gantt chart body.
	body []string
	// config is the configuration for the Gantt chart.
	config *config
	// dest is output destination for Gantt chart body.
	dest io.Writer
	// err manages errors that occur in all parts of the Gantt chart building.
	err error
}

func NewChart(w io.Writer, opts ...Option) *Chart {
	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	lines := []string{"gantt"}
	if c.title != noTitle {
		lines = append(lines, fmt.Sprintf("    title %s", c.title))
	}
	if c.dateFormat != "" {
		lines = append(lines, fmt.Sprintf("    dateFormat %s", c.dateFormat))
	}
	if c.axisFormat != "" {
		lines = append(lines, fmt.Sprintf("    axisFormat %s", c.axisFormat))
	}
	if c.tickInterval != "" {
		lines = append(lines, fmt.Sprintf("    tickInterval %s", c.tickInterval))
	}
	if c.todayMarker != "" {
		lines = append(lines, fmt.Sprintf("    todayMarker %s", c.todayMarker))
	}
	for _, exclude := range c.excludes {
		lines = append(lines, fmt.Sprintf("    excludes %s", exclude))
	}

	return &Chart{
		body:   lines,
		dest:   w,
		config: c,
	}
}

func (c *Chart) String() string {
	return strings.Join(c.body, internal.LineFeed())
}

func (c *Chart) Error() error {
	return c.err
}

func (c *Chart) Build() error {
	if _, err := fmt.Fprint(c.dest, c.String()); err != nil {
		if c.err != nil {
			return fmt.Errorf("failed to write: %w: %s", err, c.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (c *Chart) Section(name string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    section %s", name))
	return c
}

func (c *Chart) Task(name, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :%s, %s", name, startDate, duration))
	return c
}

func (c *Chart) TaskWithID(name, id, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :%s, %s, %s", name, id, startDate, duration))
	return c
}

func (c *Chart) CriticalTask(name, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, %s, %s", name, startDate, duration))
	return c
}

func (c *Chart) CriticalTaskWithID(name, id, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, %s, %s, %s", name, id, startDate, duration))
	return c
}

func (c *Chart) ActiveTask(name, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :active, %s, %s", name, startDate, duration))
	return c
}

func (c *Chart) ActiveTaskWithID(name, id, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :active, %s, %s, %s", name, id, startDate, duration))
	return c
}

func (c *Chart) DoneTask(name, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :done, %s, %s", name, startDate, duration))
	return c
}

func (c *Chart) DoneTaskWithID(name, id, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :done, %s, %s, %s", name, id, startDate, duration))
	return c
}

func (c *Chart) CriticalActiveTask(name, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, active, %s, %s", name, startDate, duration))
	return c
}

func (c *Chart) CriticalActiveTaskWithID(name, id, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, active, %s, %s, %s", name, id, startDate, duration))
	return c
}

func (c *Chart) CriticalDoneTask(name, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, done, %s, %s", name, startDate, duration))
	return c
}

func (c *Chart) CriticalDoneTaskWithID(name, id, startDate, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, done, %s, %s, %s", name, id, startDate, duration))
	return c
}

func (c *Chart) Milestone(name, date string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :milestone, %s, 0d", name, date))
	return c
}

func (c *Chart) MilestoneWithID(name, id, date string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :milestone, %s, %s, 0d", name, id, date))
	return c
}

func (c *Chart) CriticalMilestone(name, date string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, milestone, %s, 0d", name, date))
	return c
}

func (c *Chart) CriticalMilestoneWithID(name, id, date string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :crit, milestone, %s, %s, 0d", name, id, date))
	return c
}

func (c *Chart) TaskAfter(name, afterTaskID, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :after %s, %s", name, afterTaskID, duration))
	return c
}

func (c *Chart) TaskAfterWithID(name, id, afterTaskID, duration string) *Chart {
	c.body = append(c.body, fmt.Sprintf("    %s :%s, after %s, %s", name, id, afterTaskID, duration))
	return c
}

func (c *Chart) LF() *Chart {
	c.body = append(c.body, "")
	return c
}
