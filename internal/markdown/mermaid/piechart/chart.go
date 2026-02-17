// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

// Package piechart is mermaid pie chart builder.
package piechart

import (
	"fmt"
	"io"
	"strings"

	"tgp/internal/markdown/internal"
)

type PieChart struct {
	err    error
	body   []string
	dest   io.Writer
	config *config
}

func NewPieChart(w io.Writer, opts ...Option) (p *PieChart) {

	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	lines := []string{}
	lines = append(
		lines,
		fmt.Sprintf(
			"%%%%{init: {\"pie\": {\"textPosition\": %.2f}, \"themeVariables\": {\"pieOuterStrokeWidth\": \"5px\"}} }%%%%",
			c.textPosition,
		))

	baseLine := "pie"
	if c.showData {
		baseLine += " showData"
	}

	if c.title == noTitle {
		lines = append(lines, baseLine)
	} else {
		lines = append(lines, baseLine)
		lines = append(lines, fmt.Sprintf("    title %s", c.title))
	}

	return &PieChart{
		body:   lines,
		dest:   w,
		config: c,
	}
}

func (p *PieChart) String() (s string) {
	return strings.Join(p.body, internal.LineFeed())
}

func (p *PieChart) Build() (err error) {

	if _, writeErr := fmt.Fprint(p.dest, p.String()); writeErr != nil {
		if p.err != nil {
			return fmt.Errorf("failed to write: %w: %s", writeErr, p.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", writeErr)
	}
	return p.err
}

func (p *PieChart) LabelAndIntValue(label string, value uint64) *PieChart {

	p.body = append(p.body, fmt.Sprintf("    \"%s\" : %d", label, value))
	return p
}

func (p *PieChart) LabelAndFloatValue(label string, value float64) *PieChart {

	p.body = append(p.body, fmt.Sprintf("    \"%s\" : %f", label, value))
	return p
}
