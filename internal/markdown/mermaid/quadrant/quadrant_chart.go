// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

// Package quadrant is mermaid quadrant chart builder.
package quadrant

import (
	"fmt"
	"io"
	"strings"

	"tgp/internal/markdown/internal"
)

type Chart struct {
	// body is quadrant chart body.
	body []string
	// config is the configuration for the quadrant chart.
	config *config
	// dest is output destination for quadrant chart body.
	dest io.Writer
	// err manages errors that occur in all parts of the quadrant chart building.
	err error
}

func NewChart(w io.Writer, opts ...Option) *Chart {
	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	lines := []string{"quadrantChart"}
	if c.title != noTitle {
		lines = append(lines, fmt.Sprintf("    title %s", c.title))
	}

	return &Chart{
		body:   lines,
		dest:   w,
		config: c,
	}
}

func (ch *Chart) String() string {
	return strings.Join(ch.body, internal.LineFeed())
}

func (ch *Chart) Error() error {
	return ch.err
}

func (ch *Chart) Build() error {
	if _, err := fmt.Fprint(ch.dest, ch.String()); err != nil {
		if ch.err != nil {
			return fmt.Errorf("failed to write: %w: %s", err, ch.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (ch *Chart) XAxis(leftLabel string, rightLabel ...string) *Chart {
	if len(rightLabel) > 0 && rightLabel[0] != "" {
		ch.body = append(ch.body, fmt.Sprintf("    x-axis %s --> %s", leftLabel, rightLabel[0]))
	} else {
		ch.body = append(ch.body, fmt.Sprintf("    x-axis %s", leftLabel))
	}
	return ch
}

func (ch *Chart) YAxis(bottomLabel string, topLabel ...string) *Chart {
	if len(topLabel) > 0 && topLabel[0] != "" {
		ch.body = append(ch.body, fmt.Sprintf("    y-axis %s --> %s", bottomLabel, topLabel[0]))
	} else {
		ch.body = append(ch.body, fmt.Sprintf("    y-axis %s", bottomLabel))
	}
	return ch
}

func (ch *Chart) Quadrant1(label string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    quadrant-1 %s", label))
	return ch
}

func (ch *Chart) Quadrant2(label string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    quadrant-2 %s", label))
	return ch
}

func (ch *Chart) Quadrant3(label string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    quadrant-3 %s", label))
	return ch
}

func (ch *Chart) Quadrant4(label string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    quadrant-4 %s", label))
	return ch
}

func (ch *Chart) Point(name string, x, y float64) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    %s: [%.2f, %.2f]", name, x, y))
	return ch
}

type PointStyle struct {
	// Color is the fill color of the point (e.g., "#ff0000").
	Color string
	// Radius is the radius of the point.
	Radius int
	// StrokeColor is the border color of the point.
	StrokeColor string
	// StrokeWidth is the border width of the point (e.g., "5px").
	StrokeWidth string
}

func (ps PointStyle) String() string {
	var parts []string
	if ps.Color != "" {
		parts = append(parts, fmt.Sprintf("color: %s", ps.Color))
	}
	if ps.Radius > 0 {
		parts = append(parts, fmt.Sprintf("radius: %d", ps.Radius))
	}
	if ps.StrokeColor != "" {
		parts = append(parts, fmt.Sprintf("stroke-color: %s", ps.StrokeColor))
	}
	if ps.StrokeWidth != "" {
		parts = append(parts, fmt.Sprintf("stroke-width: %s", ps.StrokeWidth))
	}
	return strings.Join(parts, ", ")
}

func (ch *Chart) PointWithStyle(name string, x, y float64, style string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    %s: [%.2f, %.2f] %s", name, x, y, style))
	return ch
}

func (ch *Chart) PointStyled(name string, x, y float64, style PointStyle) *Chart {
	styleStr := style.String()
	if styleStr != "" {
		ch.body = append(ch.body, fmt.Sprintf("    %s: [%.2f, %.2f] %s", name, x, y, styleStr))
	} else {
		ch.body = append(ch.body, fmt.Sprintf("    %s: [%.2f, %.2f]", name, x, y))
	}
	return ch
}

func (ch *Chart) PointWithClass(name string, x, y float64, className string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    %s:::%s: [%.2f, %.2f]", name, className, x, y))
	return ch
}

func (ch *Chart) PointWithClassAndStyle(name string, x, y float64, className, style string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    %s:::%s: [%.2f, %.2f] %s", name, className, x, y, style))
	return ch
}

type ClassStyle struct {
	// Color is the fill color of the point (e.g., "#ff0000").
	Color string
	// Radius is the radius of the point.
	Radius int
	// StrokeColor is the border color of the point.
	StrokeColor string
	// StrokeWidth is the border width of the point (e.g., "10px").
	StrokeWidth string
}

func (cs ClassStyle) String() string {
	var parts []string
	if cs.Color != "" {
		parts = append(parts, fmt.Sprintf("color: %s", cs.Color))
	}
	if cs.Radius > 0 {
		parts = append(parts, fmt.Sprintf("radius: %d", cs.Radius))
	}
	if cs.StrokeColor != "" {
		parts = append(parts, fmt.Sprintf("stroke-color: %s", cs.StrokeColor))
	}
	if cs.StrokeWidth != "" {
		parts = append(parts, fmt.Sprintf("stroke-width: %s", cs.StrokeWidth))
	}
	return strings.Join(parts, ", ")
}

func (ch *Chart) ClassDef(className, style string) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    classDef %s %s", className, style))
	return ch
}

func (ch *Chart) ClassDefStyled(className string, style ClassStyle) *Chart {
	ch.body = append(ch.body, fmt.Sprintf("    classDef %s %s", className, style.String()))
	return ch
}

func (ch *Chart) LF() *Chart {
	ch.body = append(ch.body, "")
	return ch
}
